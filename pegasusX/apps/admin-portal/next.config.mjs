/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() {
    const backend = process.env.BACKEND_URL || 'http://localhost:8080';
    return [
      {
        source: '/api/auth/supplier/:path*',
        destination: `${backend}/v1/auth/supplier/:path*`,
      },
      {
        source: '/api/supplier/billing/:path*',
        destination: `${backend}/v1/supplier/billing/:path*`,
      },
    ];
  },
};

export default nextConfig;
