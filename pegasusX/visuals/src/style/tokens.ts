/** Pegasus marketing video design tokens — monochrome line art only */
export const PEGASUS_VIDEO = {
  width: 1920,
  height: 1080,
  fps: 24,
  background: '#000000',
  stroke: '#FFFFFF',
  label: 'rgba(255, 255, 255, 0.6)',
  strokeWidth: 1.5,
  holdSeconds: 0.5,
} as const;

export const DURATION = {
  short: 10,
  chapter: 75,
  ecosystem: 600,
} as const;

export function secondsToFrames(seconds: number, fps: number = PEGASUS_VIDEO.fps): number {
  return Math.round(seconds * fps);
}
