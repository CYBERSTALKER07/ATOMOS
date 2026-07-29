import Icon from '@/components/Icon';

interface LoadingBayControlsProps {
  onRefresh: () => void;
  onDispatch: () => void;
  dispatching: boolean;
}

export default function LoadingBayControls({ onRefresh, onDispatch, dispatching }: LoadingBayControlsProps) {
  return (
    <div className="flex items-center gap-3">
      <button
        type="button"
        onClick={onRefresh}
        className="portal-btn portal-btn--ghost inline-flex h-10 items-center gap-2"
      >
        <Icon name="refresh" size={16} /> Refresh
      </button>
      <button
        type="button"
        onClick={onDispatch}
        disabled={dispatching}
        className="portal-btn portal-btn--primary inline-flex h-10 items-center gap-2 disabled:opacity-50"
      >
        {dispatching ? 'Dispatching...' : 'Batch dispatch'}
      </button>
    </div>
  );
}
