import path from 'node:path';

/** Marketing site public root — rendered MP4s land here for Next.js to serve */
export const SITE_PUBLIC_ROOT = path.resolve(
  __dirname,
  '../../../softwareengineercv-main/public'
);

export const MEDIA_ROOT = path.join(SITE_PUBLIC_ROOT, 'media');

export function mediaOutputPath(category: string, slug: string): string {
  return path.join(MEDIA_ROOT, category, `${slug}.mp4`);
}
