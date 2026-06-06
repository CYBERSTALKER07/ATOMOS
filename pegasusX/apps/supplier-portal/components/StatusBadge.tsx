import StatusChip from './StatusChip';

export default function StatusBadge({ state, className = '' }: { state: string; className?: string }) {
  return <StatusChip status={state} className={className} />;
}
