type KpiCardProps = {
  label: string;
  value: string;
  delta?: string;
  deltaPositive?: boolean;
};

export function KpiCard({ label, value, delta, deltaPositive }: KpiCardProps) {
  return (
    <div className="flex flex-col gap-3 border border-white/10 bg-[#0a0a0a] p-5">
      <p className="font-mono text-[10px] uppercase tracking-widest text-white/40">{label}</p>
      <p className="text-3xl font-light">{value}</p>
      {delta ? (
        <p
          className={`font-mono text-xs ${
            deltaPositive === false ? 'text-[#FE5934]' : 'text-[#8DDC96]'
          }`}
        >
          {delta}
        </p>
      ) : null}
    </div>
  );
}

export function DemoPageHeader({ title, subtitle, label = 'Demo' }: { title: string; subtitle: string; label?: string }) {
  return (
    <div className="mb-8 border-b border-white/10 pb-6">
      <p className="font-mono text-[10px] uppercase tracking-widest text-white/40">{label}</p>
      <h1 className="mt-2 text-2xl font-semibold tracking-tight md:text-3xl">{title}</h1>
      <p className="mt-2 max-w-2xl text-sm text-white/55">{subtitle}</p>
    </div>
  );
}
