import { Octokit } from '@octokit/rest';
import { createHash } from 'node:crypto';
import { S3Client, PutObjectCommand } from '@aws-sdk/client-s3';
import { marked } from 'marked';
import { convert } from 'html-to-text';
import semver from 'semver';
const GITHUB_REPO_OWNER = 'anfragment';
const GITHUB_REPO = 'zen';
const PLATFORM_ASSETS = {
    darwin: {
        arm64: 'Zen_darwin_arm64.tar.gz',
        amd64: 'Zen_darwin_amd64.tar.gz',
    },
    windows: {
        arm64: 'Zen_windows_arm64.zip',
        amd64: 'Zen_windows_amd64.zip',
    },
    linux: {
        amd64: 'Zen_linux_amd64.tar.gz',
        arm64: 'Zen_linux_arm64.tar.gz',
    },
};
const MANIFESTS_HOST = 'update-manifests.zenprivacy.net';
const RELEASE_TRACK = 'stable';
const MANIFESTS_BASE_URL = `https://${MANIFESTS_HOST}/${RELEASE_TRACK}`;
const BUCKET_BASE_KEY = RELEASE_TRACK;
const S3_BUCKET_NAME = process.env.S3_BUCKET_NAME;
const S3_API_ENDPOINT = process.env.S3_API_ENDPOINT;
const S3_API_REGION = process.env.S3_API_REGION || 'auto';
const S3_ACCESS_KEY_ID = process.env.S3_ACCESS_KEY_ID;
const S3_SECRET_ACCESS_KEY = process.env.S3_SECRET_ACCESS_KEY;
const CF_ZONE_ID = process.env.CF_ZONE_ID;
const CF_API_KEY = process.env.CF_API_KEY;
const s3Client = new S3Client({
    endpoint: S3_API_ENDPOINT,
    region: S3_API_REGION,
    credentials: {
        accessKeyId: S3_ACCESS_KEY_ID,
        secretAccessKey: S3_SECRET_ACCESS_KEY,
    },
});
(async () => {
    if (process.argv.includes('--help') || process.argv.includes('-h')) {
        console.log(`Usage: ${process.argv[1]} [--force] [--dry-run]`);
        console.log('Options:');
        console.log(`  --force   Don't check existing manifest versions before uploading.`);
        console.log('  --dry-run Print what would have been uploaded to S3 without actually doing it.');
        process.exit(0);
    }
    const octokit = new Octokit();
    const res = await octokit.rest.repos.getLatestRelease({ owner: GITHUB_REPO_OWNER, repo: GITHUB_REPO });
    if (res.status !== 200) {
        console.error('API returned non-200 status, dumping response:\n', JSON.stringify(res, null, 2));
        process.exit(1);
    }
    const release = res.data;
    const releaseVersion = release.tag_name;
    const releaseBody = release.body;
    if (releaseBody == undefined) {
        throw new Error('Release body is empty');
    }
    if (!process.argv.includes('--force')) {
        for (const platform of Object.keys(PLATFORM_ASSETS)) {
            for (const arch of Object.keys(PLATFORM_ASSETS[platform])) {
                await fetchAndCompareManifest({ platform, arch, releaseVersion });
            }
        }
    }
    for (const platform of Object.keys(PLATFORM_ASSETS)) {
        for (const arch of Object.keys(PLATFORM_ASSETS[platform])) {
            const assetName = PLATFORM_ASSETS[platform][arch];
            const asset = release.assets.find((a) => a.name === assetName);
            if (asset === undefined) {
                throw new Error(`${assetName} missing from release assets`);
            }
            const manifest = await createManifestForSysArch({ releaseBody, releaseVersion, asset });
            const op = process.argv.includes('--dry-run') ? printManifestS3OpDryRun : uploadManifestToBucket;
            await op({ manifest, platform, arch });
        }
    }
    if (!process.argv.includes('--dry-run')) {
        await purgeCloudflareCache();
    }
})();
async function fetchAndCompareManifest({ platform, arch, releaseVersion, }) {
    const url = `${MANIFESTS_BASE_URL}/${platform}/${arch}/manifest.json`;
    const res = await fetch(url);
    if (res.status === 404) {
        console.warn(`Manifest at ${url} returned 404. Considering it as yet uncreated.`);
        return;
    }
    else if (!res.ok) {
        throw new Error(`Failed to fetch ${url}: ${res.statusText}`);
    }
    const manifest = (await res.json());
    if (semver.lte(releaseVersion, manifest.version)) {
        throw new Error(`${platform}/${arch} manifest version is ${manifest.version}, release version is ${releaseVersion}`);
    }
}
async function createManifestForSysArch({ releaseVersion, releaseBody, asset, }) {
    return {
        assetURL: asset.browser_download_url,
        description: await markdownToPlaintext(releaseBody),
        sha256: await sha256RemoteAsset(asset.browser_download_url),
        version: releaseVersion,
    };
}
async function sha256RemoteAsset(url) {
    const res = await fetch(url);
    if (!res.ok) {
        throw new Error(`Failed to fetch ${url}: ${res.statusText}`);
    }
    const data = await res.arrayBuffer();
    const hash = createHash('sha256');
    hash.update(Buffer.from(data));
    return hash.digest('hex');
}
async function printManifestS3OpDryRun({ manifest, arch, platform, }) {
    const key = `${BUCKET_BASE_KEY}/${platform}/${arch}/manifest.json`;
    const data = JSON.stringify(manifest, null, 2);
    console.log(`${data} will be uploaded to ${key}`);
    return key;
}
async function uploadManifestToBucket({ manifest, arch, platform, }) {
    const key = `${BUCKET_BASE_KEY}/${platform}/${arch}/manifest.json`;
    const data = JSON.stringify(manifest);
    console.log(`Uploading ${data} to ${key}`);
    await s3Client.send(new PutObjectCommand({
        Bucket: S3_BUCKET_NAME,
        Key: key,
        Body: data,
        ContentType: 'application/json',
        ACL: 'public-read',
    }));
    return key;
}
async function purgeCloudflareCache() {
    console.log('Purging Cloudflare cache');
    const url = `https://api.cloudflare.com/client/v4/zones/${CF_ZONE_ID}/purge_cache`;
    const res = await fetch(url, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${CF_API_KEY}`,
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ hosts: [MANIFESTS_HOST] }),
    });
    if (!res.ok) {
        throw new Error(`Failed to purge Cloudflare cache: ${res.statusText}`);
    }
    const data = (await res.json());
    if (!data.success) {
        throw new Error(`Failed to purge Cloudflare cache: ${JSON.stringify(data.errors)}`);
    }
    console.log('Cloudflare cache purged successfully');
}
async function markdownToPlaintext(input) {
    return convert(await marked(input), { wordwrap: false });
}
