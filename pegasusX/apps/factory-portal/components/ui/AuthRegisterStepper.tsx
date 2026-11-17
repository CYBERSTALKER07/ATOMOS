"use client";

import type { ReactNode } from "react";

export function AuthRegisterStepper({
  stepOrder,
  stepLabels,
  currentIndex,
}: {
  stepOrder: readonly string[];
  stepLabels: Record<string, string>;
  currentIndex: number;
}) {
  return (
    <ol className="setup-step-list !mt-0 !mb-0" aria-label="Onboarding progress">
      {stepOrder.map((id, index) => {
        const done = index < currentIndex;
        const active = index === currentIndex;
        const stateClass = done ? "setup-step-item--done" : active ? "setup-step-item--active" : "";
        return (
          <li key={id} className={`setup-step-item ${stateClass}`} aria-current={active ? "step" : undefined}>
            <span className="setup-step-badge" aria-hidden>
              {done ? (
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                  <path d="M5 12l5 5L20 7" />
                </svg>
              ) : (
                index + 1
              )}
            </span>
            <div className="setup-step-copy">
              <span className="setup-step-label">{stepLabels[id]}</span>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

export function AuthRegisterShell({
  title,
  subtitle,
  stepOrder,
  stepLabels,
  currentIndex,
  children,
  footer,
  error,
}: {
  title: string;
  subtitle: string;
  stepOrder: readonly string[];
  stepLabels: Record<string, string>;
  currentIndex: number;
  children: ReactNode;
  footer: ReactNode;
  error?: ReactNode;
}) {
  return (
    <div className="auth-card">
      <header className="mb-6">
        <h1 className="md-typescale-headline-large" style={{ margin: 0 }}>
          {title}
        </h1>
        <p className="desk-page-subtitle">{subtitle}</p>
        <AuthRegisterStepper stepOrder={stepOrder} stepLabels={stepLabels} currentIndex={currentIndex} />
      </header>

      <section className="md-card p-6">{children}</section>

      {error}

      <footer className="mt-6 flex items-center justify-between gap-4">{footer}</footer>
    </div>
  );
}
