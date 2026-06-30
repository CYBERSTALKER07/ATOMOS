"use client";

import { PageChrome } from "@pegasusx/ui-kit/portal";

export function PageChromePreview() {
  return (
    <PageChrome
      title="Dispatch overview"
      description="Manual truck assignment and smart AUTO assist."
      actions={<button type="button" className="portal-btn portal-btn--primary">Execute</button>}
    >
      <p className="text-sm text-[var(--mkt-text-secondary)]">
        Portal chrome wraps operational pages with title, subtitle, and action slots.
      </p>
    </PageChrome>
  );
}
