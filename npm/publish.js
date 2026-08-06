"use strict";

const { spawnSync } = require("node:child_process");

const numericIdentifier = "(?:0|[1-9]\\d*)";
const releasePattern = new RegExp(
  `^v(${numericIdentifier}\\.${numericIdentifier}\\.${numericIdentifier}(?:-[0-9A-Za-z.-]+)?(?:\\+[0-9A-Za-z.-]+)?)$`,
);

function parseReleaseTag(tag) {
  const match = releasePattern.exec(tag || "");
  if (!match) {
    throw new Error(`invalid release tag for npm: ${tag || "<empty>"}`);
  }
  return {
    version: match[1],
    distTag: match[1].includes("-") ? "next" : "latest",
  };
}

function run(command, args) {
  const result = spawnSync(command, args, { stdio: "inherit", shell: process.platform === "win32" });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    process.exit(result.status === null ? 1 : result.status);
  }
}

function main() {
  const release = parseReleaseTag(process.env.GITHUB_REF_NAME);
  run("npm", ["version", "--no-git-tag-version", release.version]);
  run("npm", ["pack", "--dry-run"]);
  const publishArgs = ["publish", "--access", "public"];
  if (release.distTag !== "latest") {
    publishArgs.push("--tag", release.distTag);
  }
  run("npm", publishArgs);
}

module.exports = { parseReleaseTag };

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(error.message);
    process.exit(1);
  }
}
