"use client";

import type { ComponentType, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from "react";

export function PortalField({
  id,
  label,
  optional,
  error,
  hint,
  children,
}: {
  id: string;
  label: string;
  optional?: boolean;
  error?: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <div className="portal-field">
      <label htmlFor={id} className="portal-label">
        {label}
        {optional ? <span style={{ fontWeight: 500, color: "var(--desk-text-tertiary)" }}> (optional)</span> : null}
      </label>
      {children}
      {error ? (
        <p className="portal-helper portal-helper--error" role="alert">
          {error}
        </p>
      ) : hint ? (
        <p className="portal-helper">{hint}</p>
      ) : null}
    </div>
  );
}

export function PortalInput({
  id,
  error,
  className = "",
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { error?: string }) {
  return (
    <input
      id={id}
      className={`portal-input ${className}`.trim()}
      aria-invalid={error ? true : undefined}
      {...props}
    />
  );
}

export function PortalSelect({
  id,
  error,
  className = "",
  children,
  ...props
}: SelectHTMLAttributes<HTMLSelectElement> & { error?: string }) {
  return (
    <select
      id={id}
      className={`portal-input ${className}`.trim()}
      aria-invalid={error ? true : undefined}
      {...props}
    >
      {children}
    </select>
  );
}

export function PortalSection({
  icon,
  title,
  description,
  actions,
  children,
  className = "",
}: {
  icon?: ReactNode;
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={`desk-card overflow-hidden ${className}`.trim()}>
      <div
        className="flex flex-wrap items-start justify-between gap-3 px-5 py-4"
        style={{ borderBottom: "1px solid var(--desk-border)", background: "var(--desk-surface-raised)" }}
      >
        <div className="flex items-start gap-3 min-w-0">
          {icon ? (
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px]"
              style={{ background: "var(--desk-surface-subtle)", color: "var(--desk-text-secondary)" }}
              aria-hidden
            >
              {icon}
            </div>
          ) : null}
          <div className="min-w-0">
            <h2 className="bento-card-title">{title}</h2>
            {description ? (
              <p className="mt-1 text-sm" style={{ color: "var(--desk-text-secondary)" }}>
                {description}
              </p>
            ) : null}
          </div>
        </div>
        {actions ? <div className="desk-toolbar shrink-0">{actions}</div> : null}
      </div>
      <div className="space-y-4 p-5">{children}</div>
    </section>
  );
}

export function PortalActions({
  back,
  skip,
  primary,
  backIcon,
  primaryIcon,
}: {
  back?: { label: string; onClick: () => void; disabled?: boolean };
  skip?: { label: string; onClick: () => void; disabled?: boolean };
  primary: { label: string; onClick: () => void; disabled?: boolean; loading?: boolean };
  backIcon?: ReactNode;
  primaryIcon?: ReactNode;
}) {
  return (
    <footer
      className="mt-6 flex items-center justify-between gap-4 border-t pt-6"
      style={{ borderColor: "var(--desk-border)" }}
    >
      <div className="flex items-center gap-3">
        {back ? (
          <button
            type="button"
            className="portal-btn portal-btn--outline"
            onClick={back.onClick}
            disabled={back.disabled}
          >
            {backIcon}
            {back.label}
          </button>
        ) : skip ? (
          <button
            type="button"
            className="portal-btn portal-btn--ghost"
            onClick={skip.onClick}
            disabled={skip.disabled}
          >
            {skip.label}
          </button>
        ) : (
          <span />
        )}
      </div>
      <button
        type="button"
        className="portal-btn portal-btn--primary"
        onClick={primary.onClick}
        disabled={primary.disabled || primary.loading}
      >
        {primary.loading ? "Saving…" : primary.label}
        {!primary.loading ? primaryIcon : null}
      </button>
    </footer>
  );
}

export function SelectionCard({
  selected,
  title,
  description,
  icon,
  onClick,
  checkType = "single",
}: {
  selected: boolean;
  title: string;
  description?: string;
  icon?: ReactNode;
  onClick: () => void;
  checkType?: "single" | "multi";
}) {
  return (
    <button
      type="button"
      className={`setup-option${selected ? " setup-option--selected" : ""}`}
      aria-pressed={selected}
      onClick={onClick}
    >
      {icon ? (
        <div className="setup-option-icon" aria-hidden>
          {icon}
        </div>
      ) : null}
      <div className="setup-option-body">
        <span className="setup-option-title">{title}</span>
        {description ? <span className="setup-option-desc">{description}</span> : null}
      </div>
      <span className="setup-option-check" aria-hidden>
        {selected ? (
          checkType === "multi" ? (
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3">
              <path d="M5 12l5 5L20 7" />
            </svg>
          ) : (
            <span style={{ width: 8, height: 8, borderRadius: "50%", background: "currentColor" }} />
          )
        ) : null}
      </span>
    </button>
  );
}

export function DataList({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`portal-data-list ${className}`.trim()}>{children}</div>;
}

export function DataListRow({
  children,
  onClick,
  className = "",
}: {
  children: ReactNode;
  onClick?: () => void;
  className?: string;
}) {
  const Tag = onClick ? "button" : "div";
  return (
    <Tag
      type={onClick ? "button" : undefined}
      className={`portal-data-list-row${onClick ? " portal-data-list-row--clickable" : ""} ${className}`.trim()}
      onClick={onClick}
    >
      {children}
    </Tag>
  );
}

type LinkProps = { href: string; className?: string; children: ReactNode };

export function HubCard({
  href,
  title,
  description,
  icon,
  onClick,
  LinkComponent,
}: {
  href?: string;
  title: string;
  description: string;
  icon?: ReactNode;
  onClick?: () => void;
  LinkComponent?: ComponentType<LinkProps>;
}) {
  const content = (
    <>
      <div className="flex items-center gap-3">
        {icon ? (
          <div
            className="flex h-10 w-10 items-center justify-center rounded-[10px]"
            style={{ background: "var(--desk-accent-soft)", color: "var(--desk-accent-strong)" }}
            aria-hidden
          >
            {icon}
          </div>
        ) : null}
        <h3 className="portal-hub-card-title">{title}</h3>
      </div>
      <p className="portal-hub-card-desc">{description}</p>
    </>
  );

  if (href && LinkComponent) {
    return (
      <LinkComponent href={href} className="portal-hub-card">
        {content}
      </LinkComponent>
    );
  }

  return (
    <button type="button" className="portal-hub-card w-full text-left" onClick={onClick}>
      {content}
    </button>
  );
}

export function FormAlert({
  variant = "info",
  icon,
  children,
}: {
  variant?: "info" | "error";
  icon?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className={`portal-alert portal-alert--${variant}`} role={variant === "error" ? "alert" : undefined}>
      {icon}
      <p className="portal-alert-text">{children}</p>
    </div>
  );
}
