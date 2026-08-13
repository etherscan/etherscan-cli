"use strict";

// Pack the real tarball and inspect every shipped text file. This retains the
// line-ending regression gate that caught the broken 1.0.1 Linux/macOS package,
// while the package itself no longer ships or executes installer scripts.
//
// This walks everything the tarball actually contains rather than re-reading
// package.json "files". The tarball is the ground truth of what ships, and
// walking it stays correct no matter how "files" is expressed.

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { checkFileLineEndings } = require("./prepublish-check");

const repositoryRoot = path.resolve(__dirname, "..");

function run(command, args, cwd) {
  let executable = command;
  let commandArgs = args;
  if (command === "npm" && process.env.npm_execpath) {
    executable = process.execPath;
    commandArgs = [process.env.npm_execpath, ...args];
  }
  const result = spawnSync(executable, commandArgs, { cwd, encoding: "utf8" });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`${executable} ${commandArgs.join(" ")} failed with status ${result.status}\n${result.stderr || ""}`);
  }
  return result.stdout;
}

function listFiles(rootDir) {
  const found = [];
  const walk = (dir) => {
    for (const item of fs.readdirSync(dir, { withFileTypes: true })) {
      const absolute = path.join(dir, item.name);
      if (item.isDirectory()) {
        walk(absolute);
      } else if (item.isFile()) {
        found.push(path.relative(rootDir, absolute).split(path.sep).join("/"));
      }
    }
  };
  walk(rootDir);
  return found.sort();
}

function checkTree(rootDir) {
  const files = listFiles(rootDir);
  if (files.length === 0) {
    throw new Error(`no files found under ${rootDir}`);
  }
  for (const relative of files) {
    checkFileLineEndings(path.join(rootDir, relative), relative);
  }
  return files;
}

function main() {
  const workDir = fs.mkdtempSync(path.join(os.tmpdir(), "etherscan-tarball-eol-"));
  try {
    const stdout = run("npm", ["pack", "--pack-destination", workDir], repositoryRoot);
    const tarballName = stdout.trim().split(/\r?\n/).pop();
    const tarball = path.join(workDir, tarballName);
    if (!fs.existsSync(tarball)) {
      throw new Error(`npm pack did not produce ${tarball}`);
    }

    // Extract by relative name from workDir: GNU tar treats the colon in an
    // absolute Windows path as a remote host specification.
    run("tar", ["-xzf", tarballName], workDir);
    const packageDir = path.join(workDir, "package");

    // Guard against a silent pass if the tarball layout ever changes.
    const launcher = path.join(packageDir, "npm", "bin", "etherscan.js");
    if (!fs.existsSync(launcher)) {
      throw new Error(`the packed tarball does not contain npm/bin/etherscan.js (looked in ${packageDir})`);
    }

    const checked = checkTree(packageDir);
    console.log(`${tarballName}: ${checked.length} files checked, all LF line endings.`);
  } finally {
    fs.rmSync(workDir, { recursive: true, force: true });
  }
}

module.exports = { listFiles, checkTree };

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(`Packed tarball line-ending check failed: ${error.message}`);
    process.exit(1);
  }
}
