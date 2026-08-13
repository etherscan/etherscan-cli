"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { parseReleaseTag } = require("./publish");
const { checkVersion, checkLineEndings } = require("./prepublish-check");
const { listFiles, checkTree } = require("./check-tarball-eol");

function test(name, callback) {
  try {
    callback();
    console.log(`ok - ${name}`);
  } catch (error) {
    console.error(`not ok - ${name}`);
    throw error;
  }
}

test("stable releases use the latest dist-tag", () => {
  assert.deepEqual(parseReleaseTag("v1.2.3"), { version: "1.2.3", distTag: "latest" });
});

test("prereleases use the next dist-tag", () => {
  assert.deepEqual(parseReleaseTag("v1.2.3-rc.1"), { version: "1.2.3-rc.1", distTag: "next" });
});

test("build metadata is accepted", () => {
  assert.deepEqual(parseReleaseTag("v1.2.3+build.4"), { version: "1.2.3+build.4", distTag: "latest" });
});

test("invalid release tags are rejected", () => {
  for (const tag of ["", "1.2.3", "v1.2", "v01.2.3", "v1.2.3-"]) {
    assert.throws(() => parseReleaseTag(tag));
  }
});

test("the development placeholder version cannot be published", () => {
  assert.throws(() => checkVersion("0.0.0-development"), /placeholder/);
});

test("release versions are accepted and malformed ones rejected", () => {
  checkVersion("1.0.2");
  checkVersion("1.0.2-rc.1");
  for (const version of ["", "1.0", "01.0.2", "v1.0.2"]) {
    assert.throws(() => checkVersion(version), /invalid release version/);
  }
});

function withFixture(files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "etherscan-prepublish-"));
  for (const [name, contents] of Object.entries(files)) {
    const target = path.join(dir, name);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, contents);
  }
  return dir;
}

test("LF shipped files are accepted", () => {
  const dir = withFixture({
    "scripts/install.sh": "#!/bin/sh\nset -eu\n",
    "npm/bin/etherscan.js": "\"use strict\";\n",
  });
  checkLineEndings(dir, ["scripts/install.sh", "npm/bin/etherscan.js"]);
});

test("a CRLF shipped file is rejected and named", () => {
  const dir = withFixture({
    "scripts/install.sh": "#!/bin/sh\r\nset -eu\r\n",
  });
  assert.throws(
    () => checkLineEndings(dir, ["scripts/install.sh"]),
    /^Error: scripts\/install\.sh contains CRLF line endings$/,
  );
});

test("a lone CR is rejected", () => {
  const dir = withFixture({ "README.md": "line\rnext\n" });
  assert.throws(() => checkLineEndings(dir, ["README.md"]), /CRLF line endings/);
});

// npm accepts globs and directories in "files". Silently skipping what this
// check cannot resolve would let a future "files" edit disable the gate while
// leaving the publish green, so each unresolvable shape must fail closed.
test("a missing entry fails closed", () => {
  const dir = withFixture({ "LICENSE": "MIT\n" });
  assert.throws(
    () => checkLineEndings(dir, ["LICENSE", "scripts/does-not-exist.sh"]),
    /scripts\/does-not-exist\.sh is listed in package\.json "files" but does not exist/,
  );
});

test("a directory entry fails closed", () => {
  const dir = withFixture({ "scripts/install.sh": "#!/bin/sh\r\n" });
  assert.throws(() => checkLineEndings(dir, ["scripts"]), /scripts is a directory/);
});

test("a glob entry fails closed", () => {
  const dir = withFixture({ "scripts/install.sh": "#!/bin/sh\r\n" });
  assert.throws(() => checkLineEndings(dir, ["scripts/*.sh"]), /is a glob pattern/);
});

test("binary entries are skipped", () => {
  const dir = withFixture({
    "vendor/etherscan.exe": Buffer.from([0x4d, 0x5a, 0x00, 0x0d, 0x0a, 0x00]),
  });
  checkLineEndings(dir, ["vendor/etherscan.exe"]);
});

// The tarball check walks the extracted tree, so it stays correct however
// package.json "files" is written.
test("the tree walk finds nested files and ignores how files[] is expressed", () => {
  const dir = withFixture({
    "scripts/install.sh": "#!/bin/sh\n",
    "npm/bin/etherscan.js": "\"use strict\";\n",
    "README.md": "# hi\n",
  });
  assert.deepEqual(listFiles(dir), ["README.md", "npm/bin/etherscan.js", "scripts/install.sh"]);
  checkTree(dir);
});

test("the tree walk catches a CRLF file nested anywhere", () => {
  const dir = withFixture({
    "README.md": "# hi\n",
    "scripts/nested/deep/install.sh": "#!/bin/sh\r\nset -eu\r\n",
  });
  assert.throws(
    () => checkTree(dir),
    /scripts\/nested\/deep\/install\.sh contains CRLF line endings/,
  );
});

test("the tree walk refuses an empty tree instead of passing", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "etherscan-empty-"));
  assert.throws(() => checkTree(dir), /no files found/);
});

test("every file the package ships is checked by default", () => {
  const packageInfo = require("../package.json");
  for (const entry of ["scripts/install.sh", "scripts/install.ps1", "npm/postinstall.js"]) {
    assert.ok(packageInfo.files.includes(entry), `${entry} must stay in package.json files`);
  }
  // The repository working tree must already satisfy the gate.
  checkLineEndings(path.resolve(__dirname, ".."));
});
