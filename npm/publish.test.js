"use strict";

const assert = require("node:assert/strict");
const crypto = require("node:crypto");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { parseReleaseTag, platforms, preparePackages } = require("./publish");
const { PLATFORMS, platformPackage } = require("./platform");
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

test("all release targets have npm platform packages", () => {
  assert.equal(platforms.length, 6);
  assert.deepEqual(
    platforms.map((platform) => platform.packageDir).sort(),
    [
      "cli-darwin-arm64",
      "cli-darwin-x64",
      "cli-linux-arm64",
      "cli-linux-x64",
      "cli-win32-arm64",
      "cli-win32-x64",
    ],
  );
});

test("launcher maps every supported Node platform and architecture", () => {
  assert.equal(platformPackage("darwin", "arm64"), "@etherscan-npm/cli-darwin-arm64");
  assert.equal(platformPackage("darwin", "x64"), "@etherscan-npm/cli-darwin-x64");
  assert.equal(platformPackage("linux", "arm64"), "@etherscan-npm/cli-linux-arm64");
  assert.equal(platformPackage("linux", "x64"), "@etherscan-npm/cli-linux-x64");
  assert.equal(platformPackage("win32", "arm64"), "@etherscan-npm/cli-win32-arm64");
  assert.equal(platformPackage("win32", "x64"), "@etherscan-npm/cli-win32-x64");
  assert.equal(platformPackage("freebsd", "x64"), null);
  assert.equal(Object.keys(PLATFORMS).length, 6);
});

// The mapping table above is pure, so it cannot prove the launcher acts on a nil
// lookup. Run the real launcher to cover the branch that the retired postinstall
// suite covered with ETHERSCAN_INSTALL_TEST_ARCH.
test("the launcher exits nonzero and names supported platforms on an unsupported one", () => {
  const launcher = path.join(__dirname, "bin", "etherscan.js");
  const script = [
    'const os = require("node:os");',
    'os.platform = () => "freebsd";',
    'os.arch = () => "x64";',
    `require(${JSON.stringify(launcher)});`,
  ].join("\n");
  const result = spawnSync(process.execPath, ["-e", script], {
    encoding: "utf8",
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /does not support freebsd x64/);
  assert.match(result.stderr, /Supported platforms: darwin arm64/);
});

test("launcher reinstall guidance names the umbrella package it ships in", () => {
  const packageInfo = require("../package.json");
  const source = fs.readFileSync(path.join(__dirname, "bin", "etherscan.js"), "utf8");
  // A hardcoded name here is how 1.0.3 shipped a message naming a package that
  // appeared nowhere in the source.
  assert.ok(
    !source.includes(`Reinstall ${packageInfo.name}`),
    "the reinstall hint must be derived from packageInfo.name, not hardcoded",
  );
  assert.match(source, /Reinstall \$\{packageInfo\.name\} without --omit=optional\./);
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

test("umbrella package ships only the launcher and documentation", () => {
  const packageInfo = require("../package.json");
  assert.equal(packageInfo.name, "@etherscan-npm/cli");
  assert.equal(packageInfo.scripts.postinstall, undefined);
  for (const entry of ["npm/bin/etherscan.js", "npm/platform.js", "README.md", "LICENSE"]) {
    assert.ok(packageInfo.files.includes(entry), `${entry} must stay in package.json files`);
  }
  assert.equal(Object.keys(packageInfo.optionalDependencies).length, 6);
  for (const name of Object.values(PLATFORMS)) {
    assert.equal(packageInfo.optionalDependencies[name], packageInfo.version);
  }
  checkLineEndings(path.resolve(__dirname, ".."));
});

test("platform manifests enforce the intended os and cpu", () => {
  for (const [key, name] of Object.entries(PLATFORMS)) {
    const [expectedOS, expectedCPU] = key.split(" ");
    const directory = name.replace("@etherscan-npm/", "");
    const manifest = require(path.join(__dirname, directory, "package.json"));
    assert.equal(manifest.name, name);
    assert.deepEqual(manifest.os, [expectedOS]);
    assert.deepEqual(manifest.cpu, [expectedCPU]);
    assert.equal(manifest.publishConfig.access, "public");
    assert.equal(manifest.version, require("../package.json").version);
  }
});

test("publisher verifies and stages all six release archives before the umbrella", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "etherscan-publish-fixture-"));
  const dist = path.join(root, "dist");
  const source = path.join(root, "source");
  const stage = path.join(root, "stage");
  fs.mkdirSync(dist);
  fs.mkdirSync(source);
  const checksums = [];

  const runArchive = (command, args) => {
    const result = spawnSync(command, args, { encoding: "utf8" });
    if (result.error || result.status !== 0) {
      throw result.error || new Error(`${command} failed: ${result.stderr}`);
    }
  };

  try {
    for (const platform of platforms) {
      const binary = path.join(source, platform.binary);
      fs.writeFileSync(binary, `${platform.packageDir}\n`);
      const archiveName = `etherscan_1.2.3_${platform.os}_${platform.arch}.${platform.extension}`;
      const archive = path.join(dist, archiveName);
      if (platform.extension === "zip") {
        if (process.platform === "win32") {
          runArchive("tar", ["-a", "-cf", archive, "-C", source, platform.binary]);
        } else {
          runArchive("zip", ["-jq", archive, binary]);
        }
      } else {
        runArchive("tar", ["-czf", archive, "-C", source, platform.binary]);
      }
      const hash = crypto.createHash("sha256").update(fs.readFileSync(archive)).digest("hex");
      checksums.push(`${hash}  ${archiveName}`);
      fs.rmSync(binary);
    }
    fs.writeFileSync(path.join(dist, "checksums.txt"), `${checksums.join("\n")}\n`);

    const prepared = preparePackages("1.2.3", dist, stage);
    assert.equal(prepared.length, 7);
    assert.equal(prepared.at(-1).name, "@etherscan-npm/cli");
    for (const pkg of prepared) {
      const manifest = JSON.parse(fs.readFileSync(path.join(pkg.directory, "package.json"), "utf8"));
      assert.equal(manifest.version, "1.2.3");
    }
    const umbrella = JSON.parse(
      fs.readFileSync(path.join(prepared.at(-1).directory, "package.json"), "utf8"),
    );
    for (const dependencyVersion of Object.values(umbrella.optionalDependencies)) {
      assert.equal(dependencyVersion, "1.2.3");
    }

    const firstArchive = `etherscan_1.2.3_${platforms[0].os}_${platforms[0].arch}.${platforms[0].extension}`;
    fs.writeFileSync(
      path.join(dist, "checksums.txt"),
      `${"0".repeat(64)}  ${firstArchive}\n${checksums.slice(1).join("\n")}\n`,
    );
    assert.throws(
      () => preparePackages("1.2.3", dist, path.join(root, "invalid-stage")),
      /checksum verification failed/,
    );
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
