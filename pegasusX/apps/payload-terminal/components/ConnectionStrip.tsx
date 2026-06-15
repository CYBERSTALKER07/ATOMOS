import { Text, View } from 'react-native';

import type { AppTheme } from '../theme';
import { isIOS } from '../theme';

type ConnectionStripProps = {
  theme: AppTheme;
  isOnline: boolean;
  queuedCount?: number;
  compact?: boolean;
};

export default function ConnectionStrip({
  compact = false,
  isOnline,
  queuedCount = 0,
  theme,
}: ConnectionStripProps) {
  const statusLabel = isOnline
    ? (isIOS ? 'Live sync' : 'LIVE SYNC')
    : (isIOS ? 'Offline' : 'OFFLINE');
  const queueHint = queuedCount > 0 ? ` · ${queuedCount} queued` : '';

  return (
    <View style={{ flexDirection: 'row', alignItems: 'center', gap: compact ? 4 : 6, marginTop: compact ? 0 : 4 }}>
      <View
        style={{
          width: compact ? 7 : 8,
          height: compact ? 7 : 8,
          borderRadius: compact ? 3.5 : 4,
          backgroundColor: isOnline ? '#22C55E' : '#EF4444',
        }}
      />
      <Text
        style={{
          color: theme.colors.sidebarSecondary,
          fontFamily: theme.typography.mono.fontFamily,
          fontSize: compact ? 9 : 10,
        }}
      >
        {statusLabel}
        {queueHint}
      </Text>
    </View>
  );
}
