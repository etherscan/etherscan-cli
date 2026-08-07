"use strict";

const packageInfo = require("../package.json");

try {
  if (packageInfo.version === "0.0.0-development") {
    throw new Error("the development placeholder version cannot be published");
  }
  if (!/^(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(packageInfo.version)) {
    throw new Error(`invalid release version: ${packageInfo.version}`);
  }
} catch (error) {
  console.error(`Refusing to publish @etherscan/cli: ${error.message}`);
  process.exit(1);
}
