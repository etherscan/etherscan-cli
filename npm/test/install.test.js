'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { targetFor, targets } = require('../lib/target');

test('maps all release targets', () => {
  assert.equal(targetFor('darwin', 'arm64').package, '@etherscan/cli-darwin-arm64');
  assert.equal(targetFor('linux', 'x64').package, '@etherscan/cli-linux-x64');
  assert.equal(targetFor('win32', 'x64').package, '@etherscan/cli-win32-x64');
  assert.equal(targets.length, 6);
});

test('rejects unsupported targets', () => {
  assert.throws(() => targetFor('freebsd', 'x64'), /unsupported platform/);
  assert.throws(() => targetFor('linux', 'ia32'), /unsupported platform/);
});
