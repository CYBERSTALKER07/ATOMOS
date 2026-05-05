import { dirname } from "node:path";
import { fileURLToPath } from "node:url";
import type { NextConfig } from "next";

const isTauriBuild = process.env.TAURI_BUILD === "1";
const appRoot = dirname(fileURLToPath(import.meta.url));

const nextConfig: NextConfig = {
  turbopack: {
    root: appRoot,
  },
  ...(isTauriBuild ? { output: "export", images: { unoptimized: true } } : {}),
};

export default nextConfig;
