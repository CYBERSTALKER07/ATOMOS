import { BentoGrid, BentoSkeleton } from '@/components/BentoGrid';

export default function Loading() {
  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <BentoGrid>
        <BentoSkeleton size="control" />
        <BentoSkeleton size="list" />
        <BentoSkeleton size="anchor" />
        <BentoSkeleton size="wide" />
      </BentoGrid>
    </div>
  );
}
