"use client";

import { TECH_ICONS, type TechIconId } from "./TechIcons";

type TechIconGridProps = {
  icons: TechIconId[];
  activeIndex?: number;
  columns?: number;
  className?: string;
};

export function TechIconGrid({
  icons,
  activeIndex = -1,
  columns = 3,
  className = "",
}: TechIconGridProps) {
  return (
    <div
      className={`tech-icon-grid ${className}`.trim()}
      style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}
    >
      {icons.map((id, index) => {
        const { label, Icon } = TECH_ICONS[id];
        const isActive = activeIndex === index;
        return (
          <div
            key={id}
            className={`tech-icon-cell ${isActive || activeIndex === -1 ? "tech-icon-cell--active" : "tech-icon-cell--inactive"}`}
          >
            <Icon size={32} className="text-[var(--mkt-text)]" />
            <span className="tech-icon-label">{label}</span>
          </div>
        );
      })}
    </div>
  );
}

export function TechIconLadder({
  icons,
  activeIndex = 0,
  className = "",
}: {
  icons: TechIconId[];
  activeIndex?: number;
  className?: string;
}) {
  return (
    <div className={`flex flex-col gap-2 ${className}`.trim()}>
      {icons.map((id, index) => {
        const { label, Icon } = TECH_ICONS[id];
        const isActive = activeIndex === index;
        return (
          <div
            key={id}
            className={`flex items-center gap-4 rounded-lg border px-4 py-3 transition-colors ${
              isActive
                ? "border-[var(--mkt-text)] bg-[var(--mkt-elevated)]"
                : "border-[var(--mkt-border)] opacity-40"
            }`}
          >
            <Icon size={24} />
            <span className="font-mono text-xs uppercase tracking-wider">{label}</span>
          </div>
        );
      })}
    </div>
  );
}
