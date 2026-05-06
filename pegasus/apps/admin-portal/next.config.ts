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
		resolveAlias: {
			"@pegasus/i18n": "../../packages/i18n/index.ts",
			"@pegasus/i18n/*": "../../packages/i18n/*",
			"@pegasus/types": "../../packages/types/index.ts",
			"@pegasus/types/*": "../../packages/types/*",
			"@pegasus/api-client": "../../packages/api-client/index.ts",
		},
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
