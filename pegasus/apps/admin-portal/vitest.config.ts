import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: [
      { find: "@", replacement: path.resolve(__dirname, ".") },
      {
        find: "@pegasus/types",
        replacement: path.resolve(__dirname, "../../packages/types/index.ts"),
      },
      {
        find: /^@pegasus\/types\/(.*)$/,
        replacement: path.resolve(__dirname, "../../packages/types/$1"),
      },
    ],
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: [],
    include: ["**/__tests__/**/*.test.{ts,tsx}", "**/*.test.{ts,tsx}"],
  },
});
