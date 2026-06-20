"use client";

import Icon from "@/components/Icon";
import { PortalField, PortalInput, FormAlert as PortalFormAlert } from "@/components/portal";
import type { InputHTMLAttributes, ReactNode } from "react";

export function SetupPageHeader({
  icon,
  title,
  subtitle,
}: {
  icon: string;
  title: string;
  subtitle: string;
}) {
  return (
    <header className="setup-header">
      <div className="setup-header-icon" aria-hidden>
        <Icon name={icon} size={22} />
      </div>
      <div>
        <h1>{title}</h1>
        <p className="setup-header-sub">{subtitle}</p>
      </div>
    </header>
  );
}

export function SetupSection({
  icon,
  title,
  description,
  children,
}: {
  icon: string;
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <section className="setup-card">
      <div className="setup-section-head">
        <div className="setup-section-icon" aria-hidden>
          <Icon name={icon} size={18} />
        </div>
        <div>
          <h2 className="setup-section-title">{title}</h2>
          {description ? <p className="setup-section-desc">{description}</p> : null}
        </div>
      </div>
      {children}
    </section>
  );
}

export function SetupField({
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
    <PortalField id={id} label={label} optional={optional} error={error} hint={hint}>
      {children}
    </PortalField>
  );
}

export function SetupInput({
  id,
  error,
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { error?: string }) {
  return <PortalInput id={id} error={error} className="setup-input" {...props} />;
}

export function SetupCallout({
  variant = "info",
  children,
}: {
  variant?: "info" | "error";
  children: ReactNode;
}) {
  return <PortalFormAlert variant={variant}>{children}</PortalFormAlert>;
}

export function SetupFooter({
  back,
  skip,
  primary,
}: {
  back?: { label: string; onClick: () => void; disabled?: boolean };
  skip?: { label: string; onClick: () => void; disabled?: boolean };
  primary: { label: string; onClick: () => void; disabled?: boolean; loading?: boolean };
}) {
  return (
    <footer className="setup-footer">
      <div className="setup-footer-left">
        {back ? (
          <button
            type="button"
            className="setup-btn setup-btn--outline"
            onClick={back.onClick}
            disabled={back.disabled}
          >
            <Icon name="arrowBack" size={16} />
            {back.label}
          </button>
        ) : skip ? (
          <button
            type="button"
            className="setup-btn setup-btn--ghost"
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
        className="setup-btn setup-btn--primary"
        onClick={primary.onClick}
        disabled={primary.disabled || primary.loading}
      >
        {primary.loading ? "Saving…" : primary.label}
        {!primary.loading ? <Icon name="arrow_forward" size={16} /> : null}
      </button>
    </footer>
  );
}

export function SelectionOption({
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
  icon: string;
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
      <div className="setup-option-icon" aria-hidden>
        <Icon name={icon} size={18} />
      </div>
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
