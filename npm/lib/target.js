'use strict';

const targets = [
  { platform: 'darwin', arch: 'x64', goos: 'darwin', goarch: 'amd64', package: '@etherscan/cli-darwin-x64' },
  { platform: 'darwin', arch: 'arm64', goos: 'darwin', goarch: 'arm64', package: '@etherscan/cli-darwin-arm64' },
  { platform: 'linux', arch: 'x64', goos: 'linux', goarch: 'amd64', package: '@etherscan/cli-linux-x64' },
  { platform: 'linux', arch: 'arm64', goos: 'linux', goarch: 'arm64', package: '@etherscan/cli-linux-arm64' },
  { platform: 'win32', arch: 'x64', goos: 'windows', goarch: 'amd64', package: '@etherscan/cli-win32-x64' },
  { platform: 'win32', arch: 'arm64', goos: 'windows', goarch: 'arm64', package: '@etherscan/cli-win32-arm64' }
];

function targetFor(platform = process.platform, arch = process.arch) {
  const target = targets.find((candidate) => candidate.platform === platform && candidate.arch === arch);
  if (!target) {
    throw new Error(`unsupported platform: ${platform}/${arch}`);
  }
  return target;
}

module.exports = { targetFor, targets };
