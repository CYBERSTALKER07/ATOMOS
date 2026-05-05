import { dirname } from "node:path";
import { fileURLToPath } from "node:url";
import type { NextConfig } from "next";

const isTauriBuild = process.env.TAURI_BUILD === "1";
const appRoot = dirname(fileURLToPath(import.meta.url));

const imageConfig: NonNullable<NextConfig["images"]> = {
  remotePatterns: [
    { protocol: "https", hostname: "**" },
    { protocol: "http", hostname: "**" },
  ],
  ...(isTauriBuild ? { unoptimized: true } : {}),
};

const nextConfig: NextConfig = {
	turbopack: {
		root: appRoot,
	},
  images: imageConfig,
  // SSG export for Tauri desktop builds; standard server mode for web
  ...(isTauriBuild
    ? {
        output: "export",
      }
    : {}),
};

export default nextConfig;
