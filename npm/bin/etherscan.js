#!/usr/bin/env node

"use strict";

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const packageInfo = require("../../package.json");
const { PLATFORMS, platformPackage } = require("../platform");

const packageRoot = path.resolve(__dirname, "..", "..");

// Resolve these once so the package lookup and executable name cannot disagree.
const platform = os.platform();
const arch = os.arch();
const binaryName = platform === "win32" ? "etherscan.exe" : "etherscan";
const reinstallHint = `Reinstall ${packageInfo.name} without --omit=optional.`;

function getExecutable() {
  const packageName = platformPackage(platform, arch);
  if (!packageName) {
    console.error(
      `Etherscan CLI does not support ${platform} ${arch}. ` +
        `Supported platforms: ${Object.keys(PLATFORMS).join(", ")}.`,
    );
    process.exit(1);
  }

  try {
    const manifest = require.resolve(`${packageName}/package.json`, {
      paths: [packageRoot],
    });
    return path.join(path.dirname(manifest), binaryName);
  } catch {
    console.error(`The platform package ${packageName} is not installed. ${reinstallHint}`);
    process.exit(1);
  }
}

const executable = getExecutable();

if (!fs.existsSync(executable)) {
  console.error(`The platform package executable is missing: ${executable}. ${reinstallHint}`);
  process.exit(1);
}

const result = spawnSync(executable, process.argv.slice(2), {
  stdio: "inherit",
  env: {
    ...process.env,
    ETHERSCAN_INSTALL_METHOD: "npm",
    ETHERSCAN_NPM_PACKAGE: packageInfo.name,
    ETHERSCAN_NPM_WRAPPER_PID: String(process.pid),
  },
});

if (result.error) {
  console.error(`Failed to start Etherscan CLI: ${result.error.message}`);
  process.exit(1);
}

if (result.signal) {
  console.error(`Etherscan CLI terminated by signal ${result.signal}.`);
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);
