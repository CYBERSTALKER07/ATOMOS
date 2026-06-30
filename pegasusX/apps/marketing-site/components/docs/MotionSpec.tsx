type MotionSpecProps = {
  spec: string;
};

export function MotionSpec({ spec }: MotionSpecProps) {
  return (
    <div className="mkt-card p-4">
      <p className="text-xs font-semibold uppercase tracking-wider text-[var(--mkt-text-tertiary)]">
        Motion spec
      </p>
      <p className="mt-2 text-sm text-[var(--mkt-text-secondary)]">{spec}</p>
    </div>
  );
}
