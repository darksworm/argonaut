#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const https = require('https');
const { spawn } = require('child_process');

const BINARY_NAME = 'a9s';
const REPO_OWNER = 'darksworm';
const REPO_NAME = 'argonaut';

// Map of platform/arch to GitHub release asset names
const PLATFORM_MAP = {
  'darwin-arm64': 'a9s_Darwin_arm64',
  'darwin-x64': 'a9s_Darwin_x86_64',
  'linux-arm64': 'a9s_Linux_arm64',
  'linux-x64': 'a9s_Linux_x86_64',
  'win32-x64': 'a9s_Windows_x86_64.exe',
};

function getPlatformKey() {
  const platform = process.platform;
  const arch = process.arch === 'x64' ? 'x64' : process.arch;
  return `${platform}-${arch}`;
}

function getBinaryName() {
  const platformKey = getPlatformKey();
  const binaryName = PLATFORM_MAP[platformKey];

  if (!binaryName) {
    console.error(`Unsupported platform: ${platformKey}`);
    console.error('Supported platforms:', Object.keys(PLATFORM_MAP).join(', '));
    process.exit(1);
  }

  return binaryName;
}

async function getLatestRelease() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: `/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest`,
      headers: {
        'User-Agent': 'argonaut-npm-installer',
      },
    };

    https.get(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          const release = JSON.parse(data);
          resolve(release);
        } catch (err) {
          reject(err);
        }
      });
    }).on('error', reject);
  });
}

async function downloadBinary(url, destPath) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(destPath);

    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        downloadBinary(response.headers.location, destPath)
          .then(resolve)
          .catch(reject);
        return;
      }

      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download: ${response.statusCode}`));
        return;
      }

      const totalSize = parseInt(response.headers['content-length'], 10);
      let downloadedSize = 0;

      response.on('data', (chunk) => {
        downloadedSize += chunk.length;
        const percent = Math.round((downloadedSize / totalSize) * 100);
        process.stdout.write(`\rDownloading: ${percent}%`);
      });

      response.pipe(file);

      file.on('finish', () => {
        file.close();
        console.log('\nDownload complete!');
        resolve();
      });
    }).on('error', (err) => {
      fs.unlink(destPath, () => {});
      reject(err);
    });
  });
}

async function installFromSource() {
  console.log('No pre-built binary available. Building from source...');

  return new Promise((resolve, reject) => {
    const build = spawn('go', ['build', '-o', path.join(__dirname, '..', 'bin', BINARY_NAME), './cmd/app'], {
      stdio: 'inherit',
      cwd: path.join(__dirname, '..'),
    });

    build.on('close', (code) => {
      if (code !== 0) {
        reject(new Error(`Build failed with code ${code}`));
      } else {
        resolve();
      }
    });

    build.on('error', (err) => {
      if (err.code === 'ENOENT') {
        console.error('Go is not installed. Please install Go from https://golang.org/');
      }
      reject(err);
    });
  });
}

async function main() {
  try {
    const binDir = path.join(__dirname, '..', 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    const binaryPath = path.join(binDir, BINARY_NAME);
    const argonautPath = path.join(binDir, 'argonaut');

    // Try to download pre-built binary
    try {
      const binaryName = getBinaryName();
      console.log(`Detected platform: ${getPlatformKey()}`);
      console.log(`Looking for binary: ${binaryName}`);

      const release = await getLatestRelease();
      const asset = release.assets.find(a => a.name === binaryName);

      if (asset) {
        console.log(`Downloading ${BINARY_NAME} v${release.tag_name}...`);
        await downloadBinary(asset.browser_download_url, binaryPath);

        // Make binary executable
        fs.chmodSync(binaryPath, '755');

        // Create symlink for 'argonaut' command
        if (fs.existsSync(argonautPath)) {
          fs.unlinkSync(argonautPath);
        }
        fs.symlinkSync(binaryPath, argonautPath);

        console.log(`✓ ${BINARY_NAME} installed successfully!`);
        console.log(`You can now use 'argonaut' or 'a9s' commands.`);
      } else {
        console.log('No pre-built binary found for your platform.');
        await installFromSource();
      }
    } catch (err) {
      console.warn('Failed to download pre-built binary:', err.message);
      console.log('Attempting to build from source...');
      await installFromSource();
    }

  } catch (err) {
    console.error('Installation failed:', err);
    process.exit(1);
  }
}

main();