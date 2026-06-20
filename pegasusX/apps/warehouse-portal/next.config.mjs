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
  typedRoutes: false,
  ...(isTauriBuild
    ? { output: "export", images: { unoptimized: true } }
    : {
        async rewrites() {
          // Proxy Firebase Auth emulator through the Next dev server so browser/Tauri
          // webviews avoid cross-origin calls to :9099 (CSP + IPv6 localhost issues).
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
