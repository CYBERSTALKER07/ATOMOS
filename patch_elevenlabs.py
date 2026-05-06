with open('pegasus/apps/admin-portal/app/globals.css', 'a') as f:
    f.write("""
/* ═══════════════════════════════════════════════════════════════════════════ */
/* ELEVENLABS DESIGN SYSTEM OVERRIDES (Applied constraints)                   */
/* ═══════════════════════════════════════════════════════════════════════════ */

/* Typography Overrides */
/* Display mega/xl/lg all map to Garamond 400 (which acts as our 300) */
.md-typescale-display-large { font-family: var(--font-display) \!important; font-size: 64px \!important; line-height: 1.05 \!important; font-weight: 400 \!important; letter-spacing: -1.92px \!important; }
.md-typescale-display-medium { font-family: var(--font-display) \!important; font-size: 48px \!important; line-height: 1.08 \!important; font-weight: 400 \!important; letter-spacing: -0.96px \!important; }
.md-typescale-display-small { font-family: var(--font-display) \!important; font-size: 36px \!important; line-height: 1.17 \!important; font-weight: 400 \!important; letter-spacing: -0.36px \!important; }
.md-typescale-headline-large { font-family: var(--font-display) \!important; font-size: 32px \!important; line-height: 1.13 \!important; font-weight: 400 \!important; letter-spacing: -0.32px \!important; }
.md-typescale-headline-medium { font-family: var(--font-display) \!important; font-size: 24px \!important; line-height: 1.2 \!important; font-weight: 400 \!important; letter-spacing: 0 \!important; }
.md-typescale-headline-small { font-family: var(--font-sans) \!important; font-size: 20px \!important; line-height: 1.35 \!important; font-weight: 500 \!important; letter-spacing: 0 \!important; }

.md-typescale-title-large { font-family: var(--font-sans) \!important; font-size: 20px \!important; line-height: 1.35 \!important; font-weight: 500 \!important; letter-spacing: 0 \!important; }
.md-typescale-title-medium { font-family: var(--font-sans) \!important; font-size: 18px \!important; line-height: 1.44 \!important; font-weight: 500 \!important; letter-spacing: 0.18px \!important; }
.md-typescale-title-small { font-family: var(--font-sans) \!important; font-size: 16px \!important; line-height: 1.5 \!important; font-weight: 500 \!important; letter-spacing: 0.16px \!important; }

.md-typescale-body-large { font-family: var(--font-sans) \!important; font-size: 16px \!important; line-height: 1.5 \!important; font-weight: 400 \!important; letter-spacing: 0.16px \!important; }
.md-typescale-body-medium { font-family: var(--font-sans) \!important; font-size: 15px \!important; line-height: 1.47 \!important; font-weight: 400 \!important; letter-spacing: 0.15px \!important; }
.md-typescale-body-small { font-family: var(--font-sans) \!important; font-size: 14px \!important; line-height: 1.5 \!important; font-weight: 400 \!important; letter-spacing: 0px \!important; }

.md-typescale-label-large { font-family: var(--font-sans) \!important; font-size: 15px \!important; line-height: 1.0 \!important; font-weight: 500 \!important; letter-spacing: 0px \!important; } /* Button */
.md-typescale-label-medium { font-family: var(--font-sans) \!important; font-size: 14px \!important; line-height: 1.5 \!important; font-weight: 500 \!important; letter-spacing: 0px \!important; }
.md-typescale-label-small { font-family: var(--font-sans) \!important; font-size: 12px \!important; line-height: 1.4 \!important; font-weight: 600 \!important; letter-spacing: 0.96px \!important; text-transform: uppercase \!important; } /* Caption uppercase */

/* Components Overrides */
.md-btn, .md-btn-filled, .md-btn-tonal, .md-btn-outlined {
  border-radius: var(--radius-pill) \!important;
  font-family: var(--font-sans) \!important;
  font-size: 15px \!important;
  font-weight: 500 \!important;
  padding: 10px 20px \!important;
  height: 40px \!important;
  display: inline-flex;  
  align-items: center; 
  justify-content: center;
  transition: all 0.2s ease \!important;
}

.md-btn-filled {
  background: var(--color-primary) \!important;
  color: var(--color-on-primary) \!important;
  border: none \!important;
}
.md-btn-filled:hover, .md-btn-filled:active {
  background: var(--color-primary-active) \!important;
  box-shadow: var(--shadow-soft-drop) \!important;
}

.md-btn-outlined {
  background: transparent \!important;
  color: var(--color-ink) \!important;
  border: 1px solid var(--color-hairline-strong) \!important;
}

.md-card, .md-card-elevated, .bento-card {
  border-radius: var(--radius-xl) \!important;
  background: var(--color-surface-card) \!important;
  border: 1px solid var(--color-hairline) \!important;
  box-shadow: none \!important;
  transition: box-shadow 0.2s ease, transform 0.2s ease \!important;
}
.md-card-elevated:hover, .bento-card:hover {
  box-shadow: var(--shadow-soft-drop) \!important;
  border: 1px solid var(--color-hairline-strong) \!important;
}

.md-input-outlined {
  border-radius: var(--radius-md) \!important;
  background: var(--color-surface-card) \!important;
  color: var(--color-ink) \!important;
  border: 1px solid var(--color-hairline-strong) \!important;
  height: 44px \!important;
  padding: 12px 16px \!important;
}
.md-input-outlined:focus, .md-input-outlined:focus-within {
  border: 2px solid var(--color-ink) \!important;
}
.md-chip {
  border-radius: var(--radius-pill) \!important;
  background: var(--color-surface-strong) \!important;
  color: var(--color-ink) \!important;
  font-family: var(--font-sans) \!important;
  font-size: 12px \!important;
  font-weight: 600 \!important;
  letter-spacing: 0.96px \!important;
  text-transform: uppercase \!important;
  padding: 4px 10px \!important;
  border: none \!important;
}

/* Base Body Application */
body {
  background-color: var(--color-canvas) \!important;
  color: var(--color-ink) \!important;
  font-family: var(--font-sans) \!important;
}

/* Atmospheric Orbs Base Setup */
.gradient-orb-mint { background: radial-gradient(circle, var(--color-gradient-mint) 0%, transparent 70%); }
.gradient-orb-peach { background: radial-gradient(circle, var(--color-gradient-peach) 0%, transparent 70%); }
.gradient-orb-lavender { background: radial-gradient(circle, var(--color-gradient-lavender) 0%, transparent 70%); }
.gradient-orb-sky { background: radial-gradient(circle, var(--color-gradient-sky) 0%, transparent 70%); }
.gradient-orb-rose { background: radial-gradient(circle, var(--color-gradient-rose) 0%, transparent 70%); }

.orb-container {
  position: absolute;
  width: 400px;
  height: 400px;
  filter: blur(40px);
  opacity: 0.6;
  z-index: 0;
  pointer-events: none;
}
""")
