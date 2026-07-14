#!/usr/bin/env node

'use strict';

const { spawnSync } = require('node:child_process');
const { targetFor } = require('../lib/target');

let target;
try {
  target = targetFor();
} catch (error) {
  console.error(error.message);
  process.exit(1);
}

let executable;
try {
  const binary = target.platform === 'win32' ? 'etherscan.exe' : 'etherscan';
  executable = require.resolve(`${target.package}/bin/${binary}`);
} catch {
  console.error(`The Etherscan CLI binary for ${target.platform}/${target.arch} is missing. Reinstall @etherscan/cli without omitting optional dependencies.`);
  process.exit(1);
}

const result = spawnSync(executable, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  console.error(`Unable to start Etherscan CLI: ${result.error.message}`);
  process.exit(1);
}

if (result.signal) {
  process.kill(process.pid, result.signal);
} else {
  process.exit(result.status ?? 1);
}
