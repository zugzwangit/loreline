import { cp, mkdir, rm } from "node:fs/promises";
import { resolve } from "node:path";
import type { Plugin } from "vite";

// Copies the hosting manifest into the compiled Sites artifact.
export function sites(): Plugin {
  let root = process.cwd();

  return {
    name: "loreline-sites-metadata",
    apply: "build",
    configResolved(config) {
      root = config.root;
    },
    async closeBundle() {
      const outputDirectory = resolve(root, "dist", ".openai");
      const hostingConfig = resolve(root, ".openai", "hosting.json");

      await rm(outputDirectory, { recursive: true, force: true });
      await mkdir(outputDirectory, { recursive: true });
      await cp(hostingConfig, resolve(outputDirectory, "hosting.json"));
    },
  };
}
