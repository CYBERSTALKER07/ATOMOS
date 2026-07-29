export const STATE_COLORS: Record<string, string> = {
  DRAFT: 'var(--color-md-outline)',
  SUBMITTED: 'var(--color-md-info)',
  ACKNOWLEDGED: 'var(--color-md-primary)',
  IN_PRODUCTION: 'var(--color-md-warning)',
  READY: 'var(--color-md-success)',
  FULFILLED: 'var(--color-md-on-surface-variant)',
  CANCELLED: 'var(--color-md-error)',
};

export const PRIORITY_COLORS: Record<string, string> = {
  CRITICAL: 'var(--color-md-error)',
  URGENT: 'var(--color-md-warning)',
  NORMAL: 'var(--color-md-on-surface-variant)',
};

export const ACTIONS: Record<string, { label: string; action: string; color: string }[]> = {
  SUBMITTED: [
    { label: 'Acknowledge', action: 'ACKNOWLEDGE', color: 'var(--color-md-primary)' },
    { label: 'Cancel', action: 'CANCEL', color: 'var(--color-md-error)' },
  ],
  ACKNOWLEDGED: [
    { label: 'Start Production', action: 'START_PRODUCTION', color: 'var(--color-md-warning)' },
    { label: 'Cancel', action: 'CANCEL', color: 'var(--color-md-error)' },
  ],
  IN_PRODUCTION: [
    { label: 'Mark Ready', action: 'MARK_READY', color: 'var(--color-md-success)' },
  ],
  READY: [
    { label: 'Fulfill', action: 'FULFILL', color: 'var(--color-md-success)' },
  ],
};
