#!/usr/bin/env node

"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const packageInfo = require("../../package.json");

const packageRoot = path.resolve(__dirname, "..", "..");
const executable = path.join(
  packageRoot,
  "vendor",
  process.platform === "win32" ? "etherscan.exe" : "etherscan",
);

if (!fs.existsSync(executable)) {
  console.error(
    "Etherscan CLI is not installed in this npm package. " +
      "Reinstall @etherscan/cli without --ignore-scripts.",
  );
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
