import type { NextConfig } from "next";

const isTauriBuild = process.env.TAURI_BUILD === "1";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  ...(isTauriBuild
    ? {
        output: "export",
        images: { unoptimized: true },
        eslint: { ignoreDuringBuilds: true },
        typescript: { ignoreBuildErrors: true },
      }
    : {}),
};

export default nextConfig;
