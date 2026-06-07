import path from "node:path";
import { fileURLToPath } from "node:url";

const isTauriBuild = process.env.TAURI_BUILD === "1";
const appRoot = path.dirname(fileURLToPath(import.meta.url));
const workspaceRoot = path.resolve(appRoot, "../..");

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  outputFileTracingRoot: workspaceRoot,
  typedRoutes: true,
  experimental: {
  },
  ...(isTauriBuild
    ? { output: "export", images: { unoptimized: true } }
    : {}),
};

export default nextConfig;
