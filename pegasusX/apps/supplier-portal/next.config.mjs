import path from "node:path";
import { fileURLToPath } from "node:url";

const isTauriBuild = process.env.TAURI_BUILD === "1";
const appRoot = path.dirname(fileURLToPath(import.meta.url));
const workspaceRoot = path.resolve(appRoot, "../..");
const authEmulatorProxy =
  process.env.FIREBASE_AUTH_EMULATOR_PROXY_TARGET || "http://127.0.0.1:9099";

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  outputFileTracingRoot: workspaceRoot,
  typedRoutes: true,
  experimental: {
  },
  ...(isTauriBuild
    ? { output: "export", images: { unoptimized: true } }
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
