import { defineConfig } from "@playwright/test";

const executablePath = process.env.MARGO_CHROMIUM_EXECUTABLE;
if (!executablePath) {
  throw new Error("margo.harness_chromium_path_required");
}

export default defineConfig({
  testDir: ".",
  testMatch: ["harness/*.spec.mjs"],
  workers: 1,
  retries: 0,
  fullyParallel: false,
  globalTimeout: 120_000,
  timeout: 60_000,
  reporter: [["line"]],
  use: {
    browserName: "chromium",
    launchOptions: {
      executablePath,
      args: [
        "--disable-background-networking",
        "--disable-component-update",
        "--disable-default-apps",
        "--disable-sync",
        "--metrics-recording-only",
        "--no-first-run",
        "--proxy-server=http://127.0.0.1:9",
        "--proxy-bypass-list=<-loopback>",
        "--safebrowsing-disable-auto-update",
      ],
    },
    headless: true,
    serviceWorkers: "block",
  },
});
