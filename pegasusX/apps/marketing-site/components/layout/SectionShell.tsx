import type { ReactNode } from "react";

type SectionShellProps = {
  id: string;
  titleId?: string;
  children: ReactNode;
  className?: string;
  minHeight?: string;
};

export function SectionShell({
  id,
  titleId,
  children,
  className = "",
  minHeight = "min-h-screen",
}: SectionShellProps) {
  return (
    <section
      id={id}
      aria-labelledby={titleId}
      className={`relative w-full ${minHeight} ${className}`.trim()}
    >
      {children}
    </section>
  );
}
