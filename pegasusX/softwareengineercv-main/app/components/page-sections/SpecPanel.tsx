'use client';

import PageSectionBlock from './PageSectionBlock';

type Spec = { label: string; value: string };

type SpecPanelProps = {
  specs: Spec[];
  variant?: 'terminal' | 'grid';
};

export default function SpecPanel({ specs, variant = 'terminal' }: SpecPanelProps) {
  if (variant === 'grid') {
    return (
      <PageSectionBlock eyebrow="Specs" title="Technical details">
        <dl className="grid gap-px bg-white/10 md:grid-cols-2">
          {specs.map((spec) => (
            <div key={spec.label} className="flex justify-between bg-black p-4 font-mono text-xs">
              <dt className="uppercase text-white/50">{spec.label}</dt>
              <dd className="text-white/90">{spec.value}</dd>
            </div>
          ))}
        </dl>
      </PageSectionBlock>
    );
  }

  return (
    <PageSectionBlock eyebrow="Specs" title="Technical details">
      <div className="overflow-hidden border border-white/15 bg-[#050505] font-mono text-xs">
        <div className="flex items-center gap-2 border-b border-white/10 px-4 py-2 text-white/40">
          <span className="h-2 w-2 rounded-full bg-[#FE5934]/80" />
          <span className="h-2 w-2 rounded-full bg-[#FBFF63]/80" />
          <span className="h-2 w-2 rounded-full bg-[#8DDC96]/80" />
          <span className="ml-2">pegasus.spec</span>
        </div>
        <dl className="divide-y divide-white/10">
          {specs.map((spec) => (
            <div key={spec.label} className="flex flex-col gap-1 px-4 py-3 sm:flex-row sm:justify-between">
              <dt className="text-white/45">{spec.label}</dt>
              <dd className="text-[#A9EBF9]">{spec.value}</dd>
            </div>
          ))}
        </dl>
      </div>
    </PageSectionBlock>
  );
}
