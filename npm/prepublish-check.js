"use strict";

const fs = require("node:fs");
const path = require("node:path");
const packageInfo = require("../package.json");

const versionPattern =
  /^(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;

// npm treats these as glob metacharacters in package.json "files".
const globPattern = /[*?[\]{}]/;

function checkVersion(version) {
  if (version === "0.0.0-development") {
    throw new Error("the development placeholder version cannot be published");
  }
  if (!versionPattern.test(version)) {
    throw new Error(`invalid release version: ${version}`);
  }
}

// `npm pack` copies the working tree, not the committed blobs. On Windows a
// checkout with core.autocrlf=true turns the LF blobs into CRLF, and a CRLF
// scripts/install.sh cannot be run by /bin/sh on Linux or macOS, which is how
// @etherscan-npm/cli@1.0.1 shipped an installer that failed on both.
function checkFileLineEndings(filePath, label) {
  const contents = fs.readFileSync(filePath);
  // Skip binaries using the same NUL-byte heuristic git applies for text=auto.
  if (contents.includes(0x00)) {
    return;
  }
  if (contents.includes(0x0d)) {
    throw new Error(`${label} contains CRLF line endings`);
  }
}

// Checks the literal entries of package.json "files". This fails closed: npm
// also accepts globs and directories there, and silently skipping what it cannot
// resolve would let a future "files" edit disable the gate while leaving the
// publish green — the same silent-pass shape that let 1.0.1 ship.
function checkLineEndings(baseDir, files = packageInfo.files) {
  for (const entry of files) {
    if (globPattern.test(entry)) {
      throw new Error(
        `${entry} is a glob pattern; this check only understands literal file paths, so update it before publishing`,
      );
    }
    const filePath = path.join(baseDir, entry);
    let stats;
    try {
      stats = fs.statSync(filePath);
    } catch (error) {
      if (error.code === "ENOENT") {
        throw new Error(`${entry} is listed in package.json "files" but does not exist`);
      }
      throw error;
    }
    if (stats.isDirectory()) {
      throw new Error(
        `${entry} is a directory; this check only understands literal file paths, so update it before publishing`,
      );
    }
    checkFileLineEndings(filePath, entry);
  }
}

function main() {
  checkVersion(packageInfo.version);
  checkLineEndings(path.resolve(__dirname, ".."));
}

module.exports = { checkVersion, checkFileLineEndings, checkLineEndings };

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(`Refusing to publish @etherscan-npm/cli: ${error.message}`);
    process.exit(1);
  }
}
