import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import type { NextConfig } from "next";

const isTauriBuild = process.env.TAURI_BUILD === "1";
const appRoot = dirname(fileURLToPath(import.meta.url));
const workspaceRoot = resolve(appRoot, "../..");
const authEmulatorProxy =
  process.env.FIREBASE_AUTH_EMULATOR_PROXY_TARGET || "http://127.0.0.1:9099";

const nextConfig: NextConfig = {
  typedRoutes: false,
  turbopack: {
    root: workspaceRoot,
  },
  outputFileTracingRoot: workspaceRoot,
  ...(isTauriBuild
    ? {
        output: "export",
        images: { unoptimized: true },
      }
    : {
        async rewrites() {
          return [
            {
              source: "/identitytoolkit.googleapis.com/:path*",
              destination: `${authEmulatorProxy}/identitytoolkit.googleapis.com/:path*`,
            },
          ];
        },
      }),
};

export default nextConfig;
