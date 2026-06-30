"use client";

export function PortalButtonPreview() {
  return (
    <div className="flex flex-wrap gap-3">
      <button type="button" className="portal-btn portal-btn--primary">
        Primary
      </button>
      <button type="button" className="portal-btn portal-btn--outline">
        Outline
      </button>
      <button type="button" className="portal-btn portal-btn--ghost">
        Ghost
      </button>
      <button type="button" className="portal-btn portal-btn--primary" disabled>
        Disabled
      </button>
    </div>
  );
}
