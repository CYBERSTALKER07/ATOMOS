declare module 'next-pwa' {
  import { NextConfig } from 'next';

  interface PwaOptions {
    dest?: string;
    disable?: boolean;
    register?: boolean;
    skipWaiting?: boolean;
    scope?: string;
    sw?: string;
    runtimeCaching?: Array<{
      urlPattern: RegExp | ((options: { url: URL; request: Request }) => boolean);
      handler: 'CacheFirst' | 'CacheOnly' | 'NetworkFirst' | 'NetworkOnly' | 'StaleWhileRevalidate';
      options?: {
        cacheName?: string;
        expiration?: {
          maxEntries?: number;
          maxAgeSeconds?: number;
        };
        cacheableResponse?: {
          statuses?: number[];
        };
        rangeRequests?: boolean;
      };
    }>;
    publicExcludes?: string[];
    buildExcludes?: (string | RegExp)[];
    fallbacks?: {
      document?: string;
      image?: string;
      audio?: string;
      video?: string;
      font?: string;
    };
    cacheOnFrontEndNav?: boolean;
    reloadOnOnline?: boolean;
    customWorkerDir?: string;
  }

  function withPWA(options: PwaOptions): (config: NextConfig) => NextConfig;
  
  export default withPWA;
}
