import path from "node:path";
import type { NextConfig } from "next";

const workspaceRoot = path.resolve(__dirname, "../..");

const nextConfig: NextConfig = {
  reactStrictMode: true,
  outputFileTracingRoot: workspaceRoot,
  typedRoutes: true,
  transpilePackages: [
    "@pegasusx/motion-tokens",
    "@pegasusx/ui-kit",
    "@pegasusx/pulse-ui",
    "@pegasusx/explain-ui",
    "@pegasusx/types",
  ],
};

export default nextConfig;
