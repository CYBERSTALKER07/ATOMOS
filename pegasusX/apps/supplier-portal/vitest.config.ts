import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: [
      { find: "@", replacement: path.resolve(__dirname, ".") },
      {
        find: "@pegasusx/types",
        replacement: path.resolve(__dirname, "../../packages/types/index.ts"),
      },
      {
        find: "@pegasusx/api-client",
        replacement: path.resolve(__dirname, "../../packages/api-client/index.ts"),
      },
      {
        find: "@pegasusx/ui-kit/portal",
        replacement: path.resolve(__dirname, "../../packages/ui-kit/src/portal/index.ts"),
      },
      {
        find: "@pegasusx/ws-refresh-contract",
        replacement: path.resolve(__dirname, "../../packages/ws-refresh-contract/index.ts"),
      },
    ],
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: "./vitest.setup.ts",
  },
});
