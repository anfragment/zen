var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
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
    },
};
const MANIFESTS_BASE_URL = 'https://zenprivacy.net/update-manifests/stable';
const BUCKET_BASE_KEY = 'update-manifests/stable';
const S3_BUCKET_NAME = process.env.S3_BUCKET_NAME;
const S3_API_ENDPOINT = process.env.S3_API_ENDPOINT;
const S3_API_REGION = process.env.S3_API_REGION || 'auto';
const S3_ACCESS_KEY_ID = process.env.S3_ACCESS_KEY_ID;
const S3_SECRET_ACCESS_KEY = process.env.S3_SECRET_ACCESS_KEY;
const s3Client = new S3Client({
    endpoint: S3_API_ENDPOINT,
    region: S3_API_REGION,
    credentials: {
        accessKeyId: S3_ACCESS_KEY_ID,
        secretAccessKey: S3_SECRET_ACCESS_KEY,
    },
});
(() => __awaiter(void 0, void 0, void 0, function* () {
    if (process.argv.includes('--help') || process.argv.includes('-h')) {
        console.log(`Usage: ${process.argv[1]} [--force] [--dry-run]`);
        console.log('Options:');
        console.log(`  --force   Don't check existing manifest versions before uploading.`);
        console.log('  --dry-run Print what would have been uploaded to S3 without actually doing it.');
        process.exit(0);
    }
    const octokit = new Octokit();
    const res = yield octokit.rest.repos.getLatestRelease({ owner: GITHUB_REPO_OWNER, repo: GITHUB_REPO });
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
                yield fetchAndCompareManifest({ platform, arch, releaseVersion });
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
            const manifest = yield createManifestForSysArch({ releaseBody, releaseVersion, asset });
            const op = process.argv.includes('--dry-run') ? printManifestS3OpDryRun : uploadManifestToBucket;
            yield op({ manifest, platform, arch });
        }
    }
}))();
function fetchAndCompareManifest(_a) {
    return __awaiter(this, arguments, void 0, function* ({ platform, arch, releaseVersion }) {
        const url = `${MANIFESTS_BASE_URL}/${platform}/${arch}/manifest.json`;
        const res = yield fetch(url);
        if (res.status === 404) {
            console.warn(`Manifest at ${url} returned 404. Considering it as yet uncreated.`);
            return;
        }
        else if (!res.ok) {
            throw new Error(`Failed to fetch ${url}: ${res.statusText}`);
        }
        const manifest = yield res.json();
        if (semver.lte(releaseVersion, manifest.version)) {
            throw new Error(`${platform}/${arch} manifest version is ${manifest.version}, release version is ${releaseVersion}`);
        }
    });
}
function createManifestForSysArch(_a) {
    return __awaiter(this, arguments, void 0, function* ({ releaseVersion, releaseBody, asset }) {
        return {
            assetURL: asset.browser_download_url,
            description: yield markdownToPlaintext(releaseBody),
            sha256: yield sha256RemoteAsset(asset.browser_download_url),
            version: releaseVersion,
        };
    });
}
function sha256RemoteAsset(url) {
    return __awaiter(this, void 0, void 0, function* () {
        const res = yield fetch(url);
        if (!res.ok) {
            throw new Error(`Failed to fetch ${url}: ${res.statusText}`);
        }
        const data = yield res.arrayBuffer();
        const hash = createHash('sha256');
        hash.update(Buffer.from(data));
        return hash.digest('hex');
    });
}
function printManifestS3OpDryRun(_a) {
    return __awaiter(this, arguments, void 0, function* ({ manifest, arch, platform, }) {
        const key = `${BUCKET_BASE_KEY}/${platform}/${arch}/manifest.json`;
        const data = JSON.stringify(manifest, null, 2);
        console.log(`${data} will be uploaded to ${key}`);
        return key;
    });
}
function uploadManifestToBucket(_a) {
    return __awaiter(this, arguments, void 0, function* ({ manifest, arch, platform, }) {
        const key = `${BUCKET_BASE_KEY}/${platform}/${arch}/manifest.json`;
        const data = JSON.stringify(manifest);
        console.log(`Uploading ${data} to ${key}`);
        yield s3Client.send(new PutObjectCommand({
            Bucket: S3_BUCKET_NAME,
            Key: key,
            Body: data,
            ContentType: 'application/json',
            ACL: 'public-read',
        }));
        return key;
    });
}
function markdownToPlaintext(input) {
    return __awaiter(this, void 0, void 0, function* () {
        return convert(yield marked(input));
    });
}
