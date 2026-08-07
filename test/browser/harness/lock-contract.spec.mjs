import { readFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test } from "@playwright/test";

const browserRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");

async function json(name) {
  return JSON.parse(await readFile(join(browserRoot, name), "utf8"));
}

test("@margo-harness package Node and browser locks are exact", async () => {
  const packageJSON = await json("package.json");
  expect(packageJSON.engines).toEqual({ node: "26.5.0", npm: "11.17.0" });
  expect(packageJSON.devDependencies).toEqual({
    "@playwright/browser-chromium": "1.52.0",
    "@playwright/test": "1.52.0",
    "css-tree": "3.1.0",
    playwright: "1.52.0",
  });

  const packageLock = await json("package-lock.json");
  const pinnedIntegrities = new Map([
    ["node_modules/@playwright/browser-chromium", "sha512-n2/e2Q0dFACFg/1JZ0t2IYLorDdno6q1QwKnNbPICHwCkAtW7+fSMqCvJ9FSMWSyPugxZqIFhownSpyATxtiTw=="],
    ["node_modules/@playwright/test", "sha512-uh6W7sb55hl7D6vsAeA+V2p5JnlAqzhqFyF0VcJkKZXkgnFcVG9PziERRHQfPLfNGx1C292a4JqbWzhR8L4R1g=="],
    ["node_modules/css-tree", "sha512-0eW44TGN5SQXU1mWSkKwFstI/22X2bG1nYzZTYMAWjylYURhse752YgbE4Cx46AC+bAvI+/dYTPRk1LqSUnu6w=="],
    ["node_modules/playwright", "sha512-JAwMNMBlxJ2oD1kce4KPtMkDeKGHQstdpFPcPH3maElAXon/QZeTvtsfXmTMRyO9TslfoYOXkSsvao2nE1ilTw=="],
  ]);
  for (const [path, integrity] of pinnedIntegrities) {
    expect(packageLock.packages[path].version).toBe(path.endsWith("css-tree") ? "3.1.0" : "1.52.0");
    expect(packageLock.packages[path].integrity).toBe(integrity);
  }

  const nodeLock = await json("node-toolchain.lock");
  expect(nodeLock).toMatchObject({
    schemaVersion: "margo/node-toolchain/v1",
    nodeVersion: "v26.5.0",
    npmVersion: "11.17.0",
    manifest: {
      sha256: "c293d34153b5d2357e6c1e521907dbf6bd3833a18565e3eb19839e5589a2bd9d",
      signatureSHA256: "1e7c8789bbd3e1628f851948db17172faa5f943b990612617b9bd2681494cfc3",
      signingKeyFingerprint: "C82FA3AE1CBEDC6BE46B9360C43CEC45C17AB93C",
      keySourceSHA256: "6030d4e0cd53330acf2ab68acd455b7ca98bb5d5975376f0b7c0892308ba2d57",
    },
  });
  expect(nodeLock.runners.map(({ id, archiveSHA256 }) => [id, archiveSHA256])).toEqual([
    ["darwin-arm64", "ee920559aaa2391569cff4d737e3b83963430e3a14dedd91bfe0ff53171b5af9"],
    ["darwin-x64", "98293394c945a24e64e00b4177bf075ec963ea70b34d1d2e24bd4a71716d334f"],
    ["linux-x64", "9f619528f1db5ddc41dccf54211066fb42228d69a156733c69cb9d6cc92e358c"],
    ["windows-x64", "d3b2277dbcccfdf24ef6302928f64f484cff1d77a6d3caa3a28f4d20ce9158f6"],
  ]);

  const browserLock = await json("browser-lock.json");
  expect(browserLock).toMatchObject({ schemaVersion: "margo/browser-lock/v1", revision: "1169", version: "136.0.7103.25" });
  expect(browserLock.runners.map(({ id, sha256 }) => [id, sha256])).toEqual([
    ["darwin-arm64", "6755dab7021ac7aeadceab8f3cd183f05f9c20736c456573b937e5e22212db65"],
    ["darwin-x64", "22c56a4fdc5b9de64a510176bcf8eb930e597c0985964736067ed505cd0109b2"],
    ["linux-x64", "1a2f6e3e519049b51c59b2503ecf808af4ff1fcf13ffa1177dacfda4a02d7a59"],
    ["windows-x64", "241f8aa5c0fde70fb0cd9fdedfb65ee34e422fef8c30b39bd1158d0e10fcb884"],
  ]);
  for (const runner of browserLock.runners) {
    expect(runner.urls).toHaveLength(3);
    expect(runner.urls[0]).toContain(`/chromium/1169/${runner.archive}`);
  }
});
