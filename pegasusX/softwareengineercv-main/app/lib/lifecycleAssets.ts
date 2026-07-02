import { ORDER_LIFECYCLE_POSTER } from '@/app/lib/siteAssets';

export const ORDER_LIFECYCLE_VIDEO_MP4 =
  'https://www.dropbox.com/scl/fi/dxt89uydvufuz5y6po2mb/Minimal_black_and_white_line_a.mp4?rlkey=8lymbhwowk6gsx9dmqjdnep0o&st=2vgxg4na&raw=1';

/** Source: Gemini-generated lifecycle animation */
export const ORDER_LIFECYCLE_GEMINI_SHARE =
  'https://share.gemini.google/vwldPADatLy5';

export { ORDER_LIFECYCLE_POSTER };

export const ORDER_LIFECYCLE_STEPS = [
  'ORDER PLACED',
  'VETTED',
  'LOADED',
  'SEALED',
  'IN TRANSIT',
  'ARRIVED',
  'PAID',
  'COMPLETED',
] as const;
