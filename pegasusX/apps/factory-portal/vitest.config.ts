import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: [
      { find: "@", replacement: path.resolve(__dirname, ".") },
      {
        find: "@pegasusx/i18n",
        replacement: path.resolve(__dirname, "../../packages/i18n/index.ts"),
      },
      {
        find: /^@pegasusx\/i18n\/(.*)$/,
        replacement: path.resolve(__dirname, "../../packages/i18n/$1"),
      },
      {
        find: "@pegasusx/types",
        replacement: path.resolve(__dirname, "../../packages/types/index.ts"),
      },
      {
        find: /^@pegasusx\/types\/(.*)$/,
        replacement: path.resolve(__dirname, "../../packages/types/$1"),
      },
      {
        find: '@pegasusx/api-core',
        replacement: path.resolve(__dirname, "../../packages/api-core/index.ts"),
      },
      {
        find: /^@pegasusx\/api-core\/(.*)$/,
        replacement: path.resolve(__dirname, "../../packages/api-core/$1"),
      },
    ],
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
    include: ["**/__tests__/**/*.test.{ts,tsx}", "**/*.test.{ts,tsx}"],
  },
});
