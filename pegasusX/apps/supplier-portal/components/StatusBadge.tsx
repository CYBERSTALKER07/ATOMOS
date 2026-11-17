import StatusChip from './StatusChip';

export default function StatusBadge({ state, label, className = '' }: { state: string; label?: string; className?: string }) {
  return <StatusChip status={state} label={label} className={className} />;
}
