"use client";

import type { ReactNode } from "react";
import { PortalSection } from "@/components/portal";

type PageSectionProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  icon?: string;
};

/** @deprecated Prefer PortalSection — thin wrapper for backward compatibility */
export function PageSection({ title, description, actions, children, className = "", icon }: PageSectionProps) {
  return (
    <PortalSection
      icon={icon}
      title={title}
      description={description}
      actions={actions}
      className={className}
    >
      {children}
    </PortalSection>
  );
}
