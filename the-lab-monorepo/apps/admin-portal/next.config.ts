import type { NextConfig } from "next";

const isTauriBuild = process.env.TAURI_BUILD === "1";

const imageConfig: NonNullable<NextConfig["images"]> = {
  remotePatterns: [
    { protocol: "https", hostname: "**" },
    { protocol: "http", hostname: "**" },
  ],
  ...(isTauriBuild ? { unoptimized: true } : {}),
};

const nextConfig: NextConfig = {
  images: imageConfig,
  // SSG export for Tauri desktop builds; standard server mode for web
  ...(isTauriBuild
    ? {
        output: "export",
      }
    : {}),
};

export default nextConfig;
