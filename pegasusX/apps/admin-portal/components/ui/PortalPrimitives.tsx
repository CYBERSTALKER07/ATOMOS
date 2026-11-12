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
      {...(props as any)}
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
      {...(props as any)}
    >
      {children}
    </select>
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
