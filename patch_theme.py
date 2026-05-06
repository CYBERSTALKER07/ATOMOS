import re

with open('pegasus/apps/admin-portal/app/globals.css', 'r') as f:
    content = f.read()

# Replace the M3 Token theme block
new_theme = """
@theme inline {
  /* ── ElevenLabs Colors ─────────────────────────── */
  --color-canvas: #f5f5f5;
  --color-canvas-soft: #fafafa;
  --color-canvas-deep: #0c0a09;
  --color-surface-card: #ffffff;
  --color-surface-strong: #f0efed;
  --color-surface-dark: #0c0a09;
  --color-surface-elevated: #1c1917;

  --color-ink: #0c0a09;
  --color-body: #4e4e4e;
  --color-body-strong: #292524;
  --color-muted: #777169;
  --color-muted-soft: #a8a29e;
  --color-on-primary: #ffffff;
  --color-on-dark: #ffffff;
  --color-on-dark-soft: #a8a29e;

  /* Accent Ink Pill */
  --color-primary: #292524;
  --color-primary-active: #0c0a09;

  /* Hairlines */
  --color-hairline: #e7e5e4;
  --color-hairline-soft: #f0efed;
  --color-hairline-strong: #d6d3d1;

  /* Semantic */
  --color-semantic-success: #16a34a;
  --color-semantic-error: #dc2626;

  /* Gradient Orbs */
  --color-gradient-mint: #a7e5d3;
  --color-gradient-peach: #f4c5a8;
  --color-gradient-lavender: #c8b8e0;
  --color-gradient-sky: #a8c8e8;
  --color-gradient-rose: #e8b8c4;

  /* Legacy M3 compatibility mappings */
  --color-md-surface: var(--color-canvas);
  --color-md-on-surface: var(--color-body);
  --color-md-primary: var(--color-primary);
  --color-md-on-primary: var(--color-on-primary);
  --color-md-outline-variant: var(--color-hairline);
  --color-md-outline: var(--color-hairline-strong);
  
  --color-md-error: var(--color-semantic-error);
  --color-md-on-error: #ffffff;
  --color-md-error-container: #fee2e2;
  --color-md-on-error-container: var(--color-semantic-error);
  
  --color-md-surface-container: var(--color-surface-card);

  /* ── Typeface ──────────────────────────────────── */
  --font-sans: var(--font-inter), -apple-system, sans-serif;
  --font-display: var(--font-garamond), "EB Garamond", "Times New Roman", serif;
  --font-mono: ui-monospace, SFMono-Regular, monospace;

  /* ── Spacing ───────────────────────────────────── */
  --spacing-xxs: 4px;
  --spacing-xs: 8px;
  --spacing-sm: 12px;
  --spacing-base: 16px;
  --spacing-md: 20px;
  --spacing-lg: 24px;
  --spacing-xl: 32px;
  --spacing-xxl: 48px;
  --spacing-section: 96px;

  /* ── Radii (ElevenLabs scale) ──────────────────── */
  --radius-xs: 4px;
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;
  --radius-xxl: 24px;
  --radius-pill: 9999px;
  --radius-full: 9999px;

  /* Override Native Radii */
  --native-radius-lg: var(--radius-xl);
  --native-radius-md: var(--radius-lg);
  --native-radius-sm: var(--radius-md);
  
  /* ── Elevation shadows ─────────────────────────── */
  --shadow-soft-drop: 0 4px 16px rgba(0,0,0,0.04);
}
"""

content = re.sub(r'@theme inline \{[^\}]*--color-md-primary:[^\}]*\}', new_theme.strip() + '\n', content, count=1, flags=re.DOTALL)

with open('pegasus/apps/admin-portal/app/globals.css', 'w') as f:
    f.write(content)
