import { Text, View } from 'react-native';

import type { AppTheme } from '../theme';
import { isIOS } from '../theme';

export type StatusBadgeTone = 'default' | 'accent' | 'success' | 'warning' | 'danger' | 'muted';

type StatusBadgeProps = {
  label: string;
  theme: AppTheme;
  tone?: StatusBadgeTone;
  compact?: boolean;
};

function resolveColors(theme: AppTheme, tone: StatusBadgeTone) {
  switch (tone) {
    case 'accent':
      return { bg: theme.colors.accent, fg: '#FFFFFF' };
    case 'success':
      return { bg: 'rgba(22, 163, 74, 0.18)', fg: '#16A34A' };
    case 'warning':
      return { bg: 'rgba(245, 158, 11, 0.18)', fg: '#B45309' };
    case 'danger':
      return { bg: 'rgba(239, 68, 68, 0.18)', fg: theme.colors.destructive };
    case 'muted':
      return { bg: theme.colors.fillTertiary, fg: theme.colors.secondaryLabel };
    default:
      return { bg: theme.colors.fillSecondary, fg: theme.colors.secondaryLabel };
  }
}

export function manifestStateTone(state: string): StatusBadgeTone {
  switch (state.toUpperCase()) {
    case 'LOADING':
      return 'accent';
    case 'SEALED':
      return 'success';
    case 'DRAFT':
      return 'muted';
    default:
      return 'default';
  }
}

export function exceptionReasonTone(reason: string): StatusBadgeTone {
  switch (reason.toUpperCase()) {
    case 'DAMAGED':
      return 'danger';
    case 'OVERFLOW':
      return 'warning';
    default:
      return 'muted';
  }
}

export default function StatusBadge({ compact = false, label, theme, tone = 'default' }: StatusBadgeProps) {
  const colors = resolveColors(theme, tone);
  const displayLabel = isIOS ? label : label.toUpperCase();

  return (
    <View
      style={{
        alignSelf: 'flex-start',
        backgroundColor: colors.bg,
        borderRadius: compact ? 4 : 6,
        paddingHorizontal: compact ? 6 : 8,
        paddingVertical: compact ? 2 : 4,
      }}
    >
      <Text
        style={{
          color: colors.fg,
          fontSize: compact ? 9 : 10,
          fontWeight: '700',
          letterSpacing: isIOS ? 0.2 : 0.5,
        }}
      >
        {displayLabel}
      </Text>
    </View>
  );
}
