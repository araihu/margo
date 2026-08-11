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
    "@playwright/browser-chromium": "1.55.1",
    "@playwright/test": "1.55.1",
    "css-tree": "3.1.0",
    playwright: "1.55.1",
  });

  const packageLock = await json("package-lock.json");
  const pinnedIntegrities = new Map([
    ["node_modules/@playwright/browser-chromium", "sha512-T4Iyhcv38bdOrtCxpjHA6WaQk4EHeud38J+SnLaOoIuGFPK7zNgSqEuilqTGB6xqfcVHYKHWePkXcYFVOux8ew=="],
    ["node_modules/@playwright/test", "sha512-IVAh/nOJaw6W9g+RJVlIQJ6gSiER+ae6mKQ5CX1bERzQgbC1VSeBlwdvczT7pxb0GWiyrxH4TGKbMfDb4Sq/ig=="],
    ["node_modules/css-tree", "sha512-0eW44TGN5SQXU1mWSkKwFstI/22X2bG1nYzZTYMAWjylYURhse752YgbE4Cx46AC+bAvI+/dYTPRk1LqSUnu6w=="],
    ["node_modules/playwright", "sha512-cJW4Xd/G3v5ovXtJJ52MAOclqeac9S/aGGgRzLabuF8TnIb6xHvMzKIa6JmrRzUkeXJgfL1MhukP0NK6l39h3A=="],
  ]);
  for (const [path, integrity] of pinnedIntegrities) {
    expect(packageLock.packages[path].version).toBe(path.endsWith("css-tree") ? "3.1.0" : "1.55.1");
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
  expect(browserLock).toMatchObject({ schemaVersion: "margo/browser-lock/v1", revision: "1193", version: "140.0.7339.186" });
  expect(browserLock.runners.map(({ id, sha256 }) => [id, sha256])).toEqual([
    ["darwin-arm64", "bfba74e7bd40db5cc2a0ef603ec9ba91736cf6b6a0836c27a5969b57b8044b61"],
    ["darwin-x64", "b847ddba0145a3bc1eb9e0d4709c3a194d522d7c8b990fc8adb58bfd31c63f37"],
    ["linux-x64", "033556f38ba72e5fb07c1a9c9a6e0ff56b5823c2cc405e034ba5763a62e2ad12"],
    ["windows-x64", "2a90e488c4e86e65ef694e3cd2834de8361dced490a5aaa40c3de148937d1cf8"],
  ]);
  for (const runner of browserLock.runners) {
    expect(runner.urls).toHaveLength(3);
    expect(runner.urls[0]).toContain(`/chromium/1193/${runner.archive}`);
  }
});
