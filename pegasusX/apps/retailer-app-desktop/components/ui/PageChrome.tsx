"use client";

import type { ReactNode } from "react";

export type PageChromeProps = {
  title: string;
  description?: string;
  icon?: ReactNode;
  actions?: ReactNode;
  loading?: boolean;
  error?: string | null;
  empty?: boolean;
  children: ReactNode;
  renderLoading?: () => ReactNode;
  renderError?: (message: string) => ReactNode;
  renderEmpty?: () => ReactNode;
};

export function PageChrome({
  title,
  description,
  icon,
  actions,
  loading,
  error,
  empty,
  children,
  renderLoading,
  renderError,
  renderEmpty,
}: PageChromeProps) {
  return (
    <div className="desk-page">
      <div className={`desk-page-header${icon ? " desk-page-header--with-icon" : ""}`}>
        {icon ? (
          <div className="desk-page-header-icon" aria-hidden>
            {icon}
          </div>
        ) : null}
        <div className="flex flex-1 flex-wrap items-start justify-between gap-4 min-w-0">
          <div className="min-w-0">
            <h1 className="desk-page-title">{title}</h1>
            {description ? <p className="desk-page-subtitle">{description}</p> : null}
          </div>
          {actions ? <div className="desk-toolbar shrink-0">{actions}</div> : null}
        </div>
      </div>

      {loading && renderLoading ? (
        renderLoading()
      ) : error && renderError ? (
        renderError(error)
      ) : empty && renderEmpty ? (
        renderEmpty()
      ) : (
        children
      )}
    </div>
  );
}
