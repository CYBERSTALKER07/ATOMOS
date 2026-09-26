import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  images: {
    // Netlify IPX requires native `sharp`; without it `/_ipx/*` returns 500 and next/image breaks.
    // Public assets are served directly when unoptimized (still fine for this marketing site).
    unoptimized: process.env.NETLIFY === 'true',
    formats: ['image/avif', 'image/webp'],
    deviceSizes: [640, 750, 828, 1080, 1200, 1920],
    imageSizes: [16, 32, 48, 64, 96, 128, 256, 384],
    minimumCacheTTL: 60,
  },
  compress: true,
  compiler: {
    removeConsole: process.env.NODE_ENV === 'production',
  },
  async redirects() {
    return [
      { source: '/roles/gate', destination: '/roles/payload-gate', permanent: true },
      // Mega-nav solutions topics → live O9 explore pages (accordion /solutions/[use-case] stays as-is)
      { source: '/solutions/dispatch-the-right-load', destination: '/capabilities/smarter-dispatch', permanent: false },
      { source: '/solutions/visual-dispatch-engine', destination: '/capabilities/smarter-dispatch', permanent: false },
      { source: '/solutions/fleet-visibility', destination: '/capabilities/live-fleet-tracking', permanent: false },
      { source: '/solutions/live-fleet-tracking', destination: '/capabilities/live-fleet-tracking', permanent: false },
      { source: '/solutions/payment-confidence', destination: '/capabilities/payment-confidence', permanent: false },
      { source: '/solutions/treasury-integrity', destination: '/roles/finance', permanent: false },
      { source: '/solutions/network-coordination', destination: '/capabilities/instant-coordination', permanent: false },
      { source: '/solutions/warehouse-operations', destination: '/roles/warehouse', permanent: false },
    ];
  },
  experimental: {
    optimizePackageImports: [
      'gsap',
      'lucide-react',
      'react-icons',
      'three',
      '@react-three/drei',
      '@react-three/fiber',
      '@react-three/rapier',
    ],
    webpackMemoryOptimizations: true,
  },
  // Prevent Next from treating the parent V.O.I.D monorepo as the workspace root
  // (multiple lockfiles up the tree otherwise inflate Turbopack memory).
  turbopack: {
    root: path.join(__dirname),
  },
};

export default nextConfig;
