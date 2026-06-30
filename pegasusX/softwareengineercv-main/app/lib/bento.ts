export const BENTO_PLACEMENTS = [
  'editorial-bento__4-2',
  'editorial-bento__2-1',
  'editorial-bento__2-2',
  'editorial-bento__4-1',
  'editorial-bento__2-1',
  'editorial-bento__2-1',
  'editorial-bento__3-1',
  'editorial-bento__3-1',
  'editorial-bento__6-1',
  'editorial-bento__2-1',
  'editorial-bento__2-1',
  'editorial-bento__2-1',
] as const;

export const BENTO_THREE = [
  'editorial-bento__3-1',
  'editorial-bento__3-2',
  'editorial-bento__3-1',
] as const;

export function bentoPlacement(
  index: number,
  pattern: readonly string[] = BENTO_PLACEMENTS
) {
  return pattern[index % pattern.length];
}

export function bentoVariant(index: number): 'featured' | 'split' | 'vertical' {
  const mod = index % 6;
  if (mod === 0) return 'featured';
  if (mod === 2 || mod === 3) return 'split';
  return 'vertical';
}
