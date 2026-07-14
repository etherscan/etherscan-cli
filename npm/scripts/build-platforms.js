#!/usr/bin/env node

'use strict';

const { mkdirSync, readFileSync, rmSync, writeFileSync } = require('node:fs');
const { spawnSync } = require('node:child_process');
const path = require('node:path');
const { targets } = require('../lib/target');

const root = path.resolve(__dirname, '..', '..');
const rootPackagePath = path.join(root, 'package.json');
const rootPackage = JSON.parse(readFileSync(rootPackagePath, 'utf8'));
const version = process.argv[2] || rootPackage.version;

if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(version)) {
  throw new Error(`invalid package version: ${version}`);
}

rootPackage.version = version;
for (const target of targets) {
  rootPackage.optionalDependencies[target.package] = version;
}
writeFileSync(rootPackagePath, `${JSON.stringify(rootPackage, null, 2)}\n`);

const platformsDir = path.join(root, 'npm', 'platforms');
rmSync(platformsDir, { recursive: true, force: true });

for (const target of targets) {
  const packageDir = path.join(platformsDir, target.package.split('/')[1]);
  const binDir = path.join(packageDir, 'bin');
  const binary = target.goos === 'windows' ? 'etherscan.exe' : 'etherscan';
  mkdirSync(binDir, { recursive: true });

  console.log(`Building ${target.package} ${version}...`);
  const result = spawnSync('go', [
    'build',
    '-trimpath',
    '-ldflags',
    `-s -w -X main.version=${version}`,
    '-o',
    path.join(binDir, binary),
    './cmd/etherscan'
  ], {
    cwd: root,
    env: { ...process.env, CGO_ENABLED: '0', GOOS: target.goos, GOARCH: target.goarch },
    stdio: 'inherit'
  });
  if (result.error || result.status !== 0) {
    throw result.error || new Error(`Go build failed for ${target.package}`);
  }

  const platformPackage = {
    name: target.package,
    version,
    description: `Etherscan CLI native binary for ${target.platform}/${target.arch}`,
    repository: {
      type: 'git',
      url: 'git+https://github.com/etherscan/etherscan-cli.git'
    },
    os: [target.platform],
    cpu: [target.arch],
    preferUnplugged: true,
    files: [`bin/${binary}`],
    publishConfig: { access: 'public' }
  };
  writeFileSync(path.join(packageDir, 'package.json'), `${JSON.stringify(platformPackage, null, 2)}\n`);
}
