"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const packageInfo = require("../package.json");

const packageRoot = path.resolve(__dirname, "..");
const installDir = path.join(packageRoot, "vendor");
const version = `v${packageInfo.version}`;

let command;
let args;
if (process.platform === "win32") {
  command = "powershell.exe";
  args = [
    "-NoLogo",
    "-NoProfile",
    "-NonInteractive",
    "-ExecutionPolicy",
    "Bypass",
    "-File",
    path.join(packageRoot, "scripts", "install.ps1"),
    "-Version",
    version,
    "-InstallDir",
    installDir,
    "-NoPathUpdate",
  ];
} else {
  command = "sh";
  args = [
    path.join(packageRoot, "scripts", "install.sh"),
    "--version",
    version,
    "--install-dir",
    installDir,
    "--no-path-update",
  ];
}

const result = spawnSync(command, args, { stdio: "inherit", env: process.env });
if (result.error) {
  console.error(`Unable to run the Etherscan CLI installer: ${result.error.message}`);
  process.exit(1);
}
if (result.status !== 0) {
  process.exit(result.status === null ? 1 : result.status);
}

const executable = path.join(
  installDir,
  process.platform === "win32" ? "etherscan.exe" : "etherscan",
);
if (!fs.existsSync(executable)) {
  console.error(`The installer did not create ${executable}.`);
  process.exit(1);
}
