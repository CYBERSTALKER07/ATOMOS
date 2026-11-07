"use client";

import { usePortalT } from "@/lib/i18n";
import { usePathname } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { useTheme } from "@/components/ThemeProvider";
import Icon from "@/components/Icon";
import { SETUP_STEPS, setupStepIndex } from "./constants";

function SetupThemeToggle() {
  const { resolved, setMode } = useTheme();
  const isDark = resolved === "dark";

  const toggle = useCallback(() => {
    setMode(isDark ? "light" : "dark");
  }, [isDark, setMode]);

  return (
    <button
      type="button"
      className="setup-theme-toggle"
      onClick={toggle}
      aria-label={isDark ? "Switch to light mode" : "Switch to dark mode"}
    >
      <Icon name={isDark ? "lightMode" : "darkMode"} size={18} />
    </button>
  );
}

function SetupStepList({ currentIndex }: { currentIndex: number }) {
  const t = usePortalT();
  return (
    <ol className="setup-step-list" aria-label={t("factory_portal.setup.setup_wizard_shell.text.onboarding_progress")}>
      {SETUP_STEPS.map((step, index) => {
        const done = index < currentIndex;
        const active = index === currentIndex;
        const stateClass = done ? "setup-step-item--done" : active ? "setup-step-item--active" : "";
        return (
          <li key={step.id} className={`setup-step-item ${stateClass}`} aria-current={active ? "step" : undefined}>
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
              <span className="setup-step-label">{step.label}</span>
              <span className="setup-step-desc">{step.description}</span>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

function MobileProgress({ currentIndex }: { currentIndex: number }) {
  const total = SETUP_STEPS.length;
  const pct = Math.round(((currentIndex + 1) / total) * 100);
  const step = SETUP_STEPS[currentIndex];

  return (
    <div className="setup-mobile-progress">
      <p style={{ margin: 0, font: "var(--type-caption-sm)", color: "var(--desk-text-tertiary)" }}>
        Step {currentIndex + 1} of {total}
      </p>
      <p style={{ margin: "4px 0 0", font: "600 15px/1.3 var(--font-sans)" }}>{step.label}</p>
      <div className="setup-progress-bar" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}>
        <div className="setup-progress-fill" style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

export default function SetupWizardShell({ children }: { children: React.ReactNode }) {
  const t = usePortalT();
  const pathname = usePathname();
  const currentIndex = setupStepIndex(pathname);
  const [entered, setEntered] = useState(false);

  useEffect(() => {
    setEntered(true);
  }, [pathname]);

  return (
    <div className="setup-shell">
      <aside className="setup-rail" aria-label={t("factory_portal.setup.setup_wizard_shell.text.setup_progress")}>
        <div>
          <div className="setup-rail-brand">
            <div className="setup-rail-mark" aria-hidden>
              <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20 4H4v2h16V4zm1 10v-2l-1-5H4l-1 5v2h1v6h10v-6h4v6h2v-6h1zm-9 4H6v-4h6v4z" />
              </svg>
            </div>
            <div>
              <h2 className="setup-rail-title">{t("factory_portal.setup.setup_wizard_shell.text.factory_setup")}</h2>
              <p className="setup-rail-sub">
                Name your factory and set the facility address so loading bay, transfers, and manifests can start.
              </p>
            </div>
          </div>
          <SetupStepList currentIndex={currentIndex} />
        </div>
        <p className="setup-rail-footer">{t("factory_portal.auth.text.pegasusx_and_copy_2026")}</p>
      </aside>

      <div className="setup-main">
        <div className="setup-main-top">
          <SetupThemeToggle />
        </div>
        <div className="setup-main-scroll">
          <div className={`setup-inner${entered ? " auth-form-enter" : ""}`}>
            <MobileProgress currentIndex={currentIndex} />
            {children}
          </div>
        </div>
      </div>
    </div>
  );
}
