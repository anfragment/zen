import { Octokit } from '@octokit/rest';
import { createHash } from 'node:crypto';
import markdownToTxt from 'markdown-to-txt';
import { S3Client, PutObjectCommand } from '@aws-sdk/client-s3';
import semver from 'semver';

const OWNER = 'anfragment';
const REPO = 'zen';
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
    am64: 'Zen_linux_amd64.tar.gz',
  }
};
const MANIFESTS_BASE_URL = 'https://zenprivacy.net/update-manifests/stable'; // hardcode the release track for now
const BUCKET_ENDPOINT = process.env.BUCKET_ENDPOINT;
const BUCKET_ACCESS_KEY_ID = process.env.BUCKET_ACCESS_KEY_ID;
const BUCKET_SECRET_ACCESS_KEY = process.env.BUCKET_SECRET_ACCESS_KEY;


type Manifest = {
  version: string;
  description: string;
  assetURL: string;
  sha256: string;
}

(async () => {
  const octokit = new Octokit();
  const res = await octokit.rest.repos.getLatestRelease({ owner: OWNER, repo: REPO });
  if (res.status !== 200) {
    console.error('API returned non-200 status, dumping response:\n', JSON.stringify(res, null, 2));
    process.exit(1);
  }

  const release = res.data;

  const releaseVersion = semver.clean(release.tag_name);
  if (releaseVersion === null) {
    throw new Error(`Failed to parse release version: ${release.tag_name}`);
  }
  const releaseBody = release.body;
  if (releaseBody == undefined) {
    throw new Error('Release body is empty');
  }

  // Check all release assets before starting to update manifest files.
  for (const platform of Object.keys(PLATFORM_ASSETS)) {
    for (const arch of Object.keys(PLATFORM_ASSETS[platform as keyof typeof PLATFORM_ASSETS])) {
      await fetchAndCompareManifest({ platform, arch, releaseVersion });
    }
  }

  for (const platform of Object.keys(PLATFORM_ASSETS)) {
    for (const arch of Object.keys(PLATFORM_ASSETS[platform as keyof typeof PLATFORM_ASSETS])) {
      const assetName = PLATFORM_ASSETS[platform as keyof typeof PLATFORM_ASSETS][arch as keyof typeof PLATFORM_ASSETS[keyof typeof PLATFORM_ASSETS]];
      const asset = release.assets.find(a => a.name === assetName);
      if (asset === undefined) {
        throw new Error(`${assetName} missing from release assets`)
      }

      const manifest = await createManifestForSysArch({ releaseBody, releaseVersion, asset });

    }
  }
})();

/**
 * fetchAndCompareManifest fetches the manifest for a given platform/arch and compares it against the release version.
 * If the manifest version is newer than the release version, it throws an error.
 */
async function fetchAndCompareManifest({ platform, arch, releaseVersion }: { platform: string; arch: string; releaseVersion: string }): Promise<void> {
  const url = `${MANIFESTS_BASE_URL}/${platform}/${arch}/manifest.json`;
  const res = await fetch(url);
  if (res.status === 404) {
    console.warn(`Manifest at ${url} returned 404. Considering it as yet uncreated.`);
  }
  if (!res.ok) {
    throw new Error(`Failed to fetch ${url}: ${res.statusText}`);
  }

  const manifest: Manifest = await res.json();
  const manifestVersion = semver.clean(manifest.version);
  if (manifestVersion === null) {
    throw new Error(`Failed to parse manifest release version: ${manifest.version}`);
  }
  if (semver.lt(releaseVersion, manifestVersion)) {
    throw new Error(`${platform}/${arch} manifest version is ${manifest.version}, release version is ${releaseVersion}`);
  }
}

async function createManifestForSysArch({
  releaseVersion,
  releaseBody,
  asset
}: {
  releaseVersion: string;
  releaseBody: string;
  asset: { browser_download_url: string; name: string }
}): Promise<Manifest> {
  return {
    assetURL: asset.browser_download_url,
    description: markdownToTxt(releaseBody),
    sha256: await sha256RemoteAsset(asset.browser_download_url),
    version: releaseVersion,
  }
}

async function sha256RemoteAsset(url: string) {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`Failed to fetch ${url}: ${res.statusText}`);
  }

  const data = await res.arrayBuffer();

  const hash = createHash('sha256');
  hash.update(Buffer.from(data));
  return hash.digest('hex');
}

async function uploadManifestToBucket(manifest: Manifest): Promise<string> {

}
