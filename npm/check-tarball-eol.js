"use strict";

// `npm pack` copies the working tree, so a checkout that rewrote the LF blobs to
// CRLF produces a tarball whose scripts/install.sh cannot be run by /bin/sh.
// That is how @etherscan-npm/cli@1.0.1 shipped an installer that failed on Linux
// and macOS: every CI job either packed on a platform that checks out LF, or
// packed on Windows and then installed through scripts/install.ps1, so the broken
// combination was never exercised. Pack the real tarball and inspect it here.
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

function quote(value) {
  return `"${value}"`;
}

// A single command string with shell:true keeps this portable across cmd.exe and
// sh without tripping the DEP0190 warning that args plus shell:true now raises.
function run(command, cwd) {
  const result = spawnSync(command, { cwd, encoding: "utf8", shell: true });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`${command} failed with status ${result.status}\n${result.stderr || ""}`);
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
    const stdout = run(`npm pack --pack-destination ${quote(workDir)}`, repositoryRoot);
    const tarballName = stdout.trim().split(/\r?\n/).pop();
    const tarball = path.join(workDir, tarballName);
    if (!fs.existsSync(tarball)) {
      throw new Error(`npm pack did not produce ${tarball}`);
    }

    // Extract by relative name from workDir: GNU tar treats the colon in an
    // absolute Windows path as a remote host specification.
    run(`tar -xzf ${quote(tarballName)}`, workDir);
    const packageDir = path.join(workDir, "package");

    // Guard against a silent pass if the tarball layout ever changes.
    const installer = path.join(packageDir, "scripts", "install.sh");
    if (!fs.existsSync(installer)) {
      throw new Error(`the packed tarball does not contain scripts/install.sh (looked in ${packageDir})`);
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
