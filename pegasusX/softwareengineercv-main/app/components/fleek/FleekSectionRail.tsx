'use client';

const SECTIONS = ['01', '02', '03', '04', '05', '06'] as const;

type FleekSectionRailProps = {
  active?: string;
};

export default function FleekSectionRail({ active }: FleekSectionRailProps) {
  return (
    <nav className="fleek-rail" aria-label="Page sections">
      {SECTIONS.map((num) => (
        <a
          key={num}
          href={`#fleek-section-${num}`}
          className={`fleek-rail__item ${active === num ? 'is-active' : ''}`}
        >
          {num}
        </a>
      ))}
    </nav>
  );
}
