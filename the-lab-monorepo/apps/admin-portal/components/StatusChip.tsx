'use client';

import { Chip } from '@heroui/react';

type ChipColor = 'default' | 'accent' | 'success' | 'warning' | 'danger';
type ChipVariant = 'primary' | 'secondary' | 'tertiary' | 'soft';

interface StatusConfig {
  color: ChipColor;
  variant: ChipVariant;
}

const STATUS_MAP: Record<string, StatusConfig> = {
  // Order lifecycle
  PENDING:                 { color: 'warning', variant: 'soft' },
  SCHEDULED:               { color: 'default', variant: 'soft' },
  LOADED:                  { color: 'accent',  variant: 'soft' },
  DISPATCHED:              { color: 'accent',  variant: 'soft' },
  IN_TRANSIT:              { color: 'accent',  variant: 'primary' },
  ARRIVED:                 { color: 'success', variant: 'soft' },
  ARRIVED_SHOP_CLOSED:     { color: 'warning', variant: 'primary' },
  AWAITING_PAYMENT:        { color: 'warning', variant: 'soft' },
  PENDING_CASH_COLLECTION: { color: 'warning', variant: 'soft' },
  COMPLETED:               { color: 'success', variant: 'primary' },
  CANCELLED:               { color: 'danger',  variant: 'soft' },
  CANCEL_REQUESTED:        { color: 'accent',  variant: 'soft' },
  NO_CAPACITY:             { color: 'danger',  variant: 'primary' },
  FAILED:                  { color: 'danger',  variant: 'primary' },
  QUARANTINE:              { color: 'danger',  variant: 'soft' },
  DELIVERED_ON_CREDIT:     { color: 'warning', variant: 'primary' },

  // Generic states
  ACTIVE:                  { color: 'success', variant: 'soft' },
  INACTIVE:                { color: 'default', variant: 'soft' },
  ENABLED:                 { color: 'success', variant: 'soft' },
  DISABLED:                { color: 'default', variant: 'soft' },
  VERIFIED:                { color: 'success', variant: 'soft' },
  UNVERIFIED:              { color: 'warning', variant: 'soft' },
  APPROVED:                { color: 'success', variant: 'primary' },
  REJECTED:                { color: 'danger',  variant: 'soft' },
  SUSPENDED:               { color: 'danger',  variant: 'soft' },

  // Fleet / driver
  AVAILABLE:               { color: 'success', variant: 'soft' },
  ON_ROUTE:                { color: 'accent',  variant: 'primary' },
  OFF_DUTY:                { color: 'default', variant: 'soft' },
  MAINTENANCE:             { color: 'warning', variant: 'soft' },

  // Financial
  PAID:                    { color: 'success', variant: 'soft' },
  UNPAID:                  { color: 'danger',  variant: 'soft' },
  PARTIAL:                 { color: 'warning', variant: 'soft' },
  REFUNDED:                { color: 'default', variant: 'soft' },
  MATCHED:                 { color: 'success', variant: 'soft' },
  UNMATCHED:               { color: 'danger',  variant: 'soft' },
  RECONCILED:              { color: 'success', variant: 'primary' },

  // Inventory / stock
  IN_STOCK:                { color: 'success', variant: 'soft' },
  LOW_STOCK:               { color: 'warning', variant: 'soft' },
  OUT_OF_STOCK:            { color: 'danger',  variant: 'soft' },

  // KYC
  PENDING_REVIEW:          { color: 'warning', variant: 'soft' },
  UNDER_REVIEW:            { color: 'accent',  variant: 'soft' },
};

const FALLBACK: StatusConfig = { color: 'default', variant: 'soft' };

interface StatusChipProps {
  status: string;
  label?: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export default function StatusChip({ status, label, size = 'sm', className }: StatusChipProps) {
  const key = status.toUpperCase().replace(/[\s-]+/g, '_');
  const config = STATUS_MAP[key] || FALLBACK;
  const displayLabel = label || status.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());

  return (
    <Chip color={config.color} variant={config.variant} size={size} className={className}>
      <Chip.Label>{displayLabel}</Chip.Label>
    </Chip>
  );
}
