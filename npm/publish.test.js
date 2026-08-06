"use strict";

const assert = require("node:assert/strict");
const { parseReleaseTag } = require("./publish");

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
