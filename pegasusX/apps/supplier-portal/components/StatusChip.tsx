'use client';

const STATUS_MAP: Record<string, { tone: string }> = {
  PENDING:                 { tone: 'warning' },
  SCHEDULED:               { tone: 'neutral' },
  LOADED:                  { tone: 'accent' },
  DISPATCHED:              { tone: 'accent' },
  IN_TRANSIT:              { tone: 'accent-strong' },
  ARRIVED:                 { tone: 'success' },
  ARRIVED_SHOP_CLOSED:     { tone: 'warning-strong' },
  AWAITING_PAYMENT:        { tone: 'warning' },
  PENDING_CASH_COLLECTION: { tone: 'warning' },
  FISCALIZING:             { tone: 'warning' },
  FISCAL_FAILED:           { tone: 'danger' },
  COMPLETED:               { tone: 'success-strong' },
  CANCELLED:               { tone: 'danger' },
  CANCEL_REQUESTED:        { tone: 'accent' },
  NO_CAPACITY:             { tone: 'danger-strong' },
  FAILED:                  { tone: 'danger-strong' },
  QUARANTINE:              { tone: 'danger' },
  DELIVERED_ON_CREDIT:     { tone: 'warning-strong' },

  ACTIVE:                  { tone: 'success' },
  INACTIVE:                { tone: 'neutral' },
  ENABLED:                 { tone: 'success' },
  DISABLED:                { tone: 'neutral' },
  VERIFIED:                { tone: 'success' },
  UNVERIFIED:              { tone: 'warning' },
  APPROVED:                { tone: 'success-strong' },
  EXECUTED:                { tone: 'success-strong' },
  APPLIED:                 { tone: 'success-strong' },
  CONVERTED:               { tone: 'success-strong' },
  REJECTED:                { tone: 'danger' },
  SUSPENDED:               { tone: 'danger' },
  SUPERSEDED:              { tone: 'neutral' },
  LOW:                     { tone: 'success' },
  MEDIUM:                  { tone: 'warning' },
  HIGH:                    { tone: 'warning-strong' },
  BLOCK:                   { tone: 'danger-strong' },
  CRITICAL:                { tone: 'danger-strong' },
  GEMINI:                  { tone: 'accent' },
  CLASSIC:                 { tone: 'neutral' },

  AVAILABLE:               { tone: 'success' },
  ON_ROUTE:                { tone: 'accent-strong' },
  OFF_DUTY:                { tone: 'neutral' },
  MAINTENANCE:             { tone: 'warning' },

  PAID:                    { tone: 'success' },
  UNPAID:                  { tone: 'danger' },
  PARTIAL:                 { tone: 'warning' },
  REFUNDED:                { tone: 'neutral' },
  MATCHED:                 { tone: 'success' },
  UNMATCHED:               { tone: 'danger' },
  RECONCILED:              { tone: 'success-strong' },

  IN_STOCK:                { tone: 'success' },
  LOW_STOCK:               { tone: 'warning' },
  OUT_OF_STOCK:            { tone: 'danger' },

  PENDING_REVIEW:          { tone: 'warning' },
  UNDER_REVIEW:            { tone: 'accent' },

  OPEN:                    { tone: 'success' },
  SOFT_CLOSED:             { tone: 'warning' },
  CLOSED:                  { tone: 'neutral' },
  REOPENED:                { tone: 'warning-strong' },
  PENDING_APPROVAL:        { tone: 'warning' },
  APPROVAL_REQUIRED:       { tone: 'warning' },
  DRAFT:                   { tone: 'neutral' },
  SUBMITTED:               { tone: 'accent' },
  ACCEPTED:                { tone: 'success-strong' },
  EXPIRED:                 { tone: 'warning' },
  REVOKED:                 { tone: 'danger' },
  PASS:                    { tone: 'success-strong' },
  FAIL:                    { tone: 'danger-strong' },
  WARN:                    { tone: 'warning' },
  OUTBOUND:                { tone: 'accent' },
  INBOUND:                 { tone: 'success' },
};

const TONE_STYLE: Record<string, React.CSSProperties> = {
  'success':         { background: 'var(--desk-success-soft)', color: 'var(--desk-success)', borderColor: 'color-mix(in srgb, var(--desk-success) 30%, transparent)' },
  'success-strong':  { background: 'var(--desk-success)', color: '#ffffff', borderColor: 'var(--desk-success)' },
  'warning':         { background: 'var(--desk-warning-soft)', color: 'var(--desk-warning)', borderColor: 'color-mix(in srgb, var(--desk-warning) 30%, transparent)' },
  'warning-strong':  { background: 'var(--desk-warning)', color: '#ffffff', borderColor: 'var(--desk-warning)' },
  'danger':          { background: 'var(--desk-danger-soft)', color: 'var(--desk-danger)', borderColor: 'color-mix(in srgb, var(--desk-danger) 30%, transparent)' },
  'danger-strong':   { background: 'var(--desk-danger)', color: '#ffffff', borderColor: 'var(--desk-danger)' },
  'accent':          { background: 'var(--desk-accent-soft)', color: 'var(--desk-accent-strong)', borderColor: 'rgba(var(--desk-accent-rgb), 0.28)' },
  'accent-strong':   { background: 'var(--desk-accent)', color: 'var(--desk-accent-on)', borderColor: 'var(--desk-accent)' },
  'neutral':         { background: 'var(--desk-canvas)', color: 'var(--desk-text-secondary)', borderColor: 'var(--desk-border)' },
};

const SIZE_STYLE: Record<string, React.CSSProperties> = {
  sm: { fontSize: 11, padding: '2px 8px' },
  md: { fontSize: 12, padding: '3px 10px' },
  lg: { fontSize: 13, padding: '4px 12px' },
};

interface StatusChipProps {
  status: string;
  label?: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export default function StatusChip({ status, label, size = 'sm', className }: StatusChipProps) {
  const normalized = (status || 'UNKNOWN').trim();
  const key = normalized.toUpperCase().replace(/[\s-]+/g, '_');
  const config = STATUS_MAP[key] || { tone: 'neutral' };
  const tone = TONE_STYLE[config.tone] || TONE_STYLE.neutral;
  const sizeStyle = SIZE_STYLE[size];
  const displayLabel = label || normalized.replace(/[_-]+/g, ' ').replace(/\b\w/g, c => c.toUpperCase());

  return (
    <span
      className={`inline-flex items-center font-semibold tracking-tight ${className || ''}`}
      style={{
        ...tone,
        ...sizeStyle,
        borderRadius: 999,
        borderWidth: 1,
        borderStyle: 'solid',
        lineHeight: 1.2,
        whiteSpace: 'nowrap',
      }}
    >
      {displayLabel}
    </span>
  );
}
