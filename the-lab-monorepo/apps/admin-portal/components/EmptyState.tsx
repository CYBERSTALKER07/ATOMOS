import Icon from './Icon';
import { Button } from '@heroui/react';

export default function EmptyState({
  icon = 'orders',
  headline,
  body,
  action,
  onAction,
}: {
  icon?: string;
  headline: string;
  body?: string;
  action?: string;
  onAction?: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4">
      <div
        className="w-14 h-14 flex items-center justify-center mb-4 rounded-full bg-surface text-muted"
      >
        <Icon name={icon} size={28} />
      </div>
      <p className="md-typescale-title-medium mb-1 text-foreground">
        {headline}
      </p>
      {body && (
        <p className="md-typescale-body-small text-center max-w-sm text-muted">
          {body}
        </p>
      )}
      {action && onAction && (
        <Button variant="secondary" onPress={onAction} className="mt-4">
          {action}
        </Button>
      )}
    </div>
  );
}
