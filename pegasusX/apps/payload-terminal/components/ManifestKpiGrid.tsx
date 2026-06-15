import { type ReactNode } from 'react';
import { Text, View } from 'react-native';

import type { AppTheme } from '../theme';
import { isIOS } from '../theme';
import StatusBadge, { manifestStateTone } from './StatusBadge';

type ManifestKpiGridProps = {
  theme: AppTheme;
  manifestId: string;
  state: string;
  totalVolumeVu: number;
  maxVolumeVu: number;
  stopCount?: number;
  regionCode?: string;
  compact?: boolean;
};

function KpiTile({
  footer,
  label,
  theme,
  value,
}: {
  footer?: ReactNode;
  label: string;
  theme: AppTheme;
  value: string;
}) {
  return (
    <View
      style={{
        flex: 1,
        backgroundColor: theme.colors.fillTertiary,
        borderRadius: theme.radius.card,
        paddingHorizontal: 12,
        paddingVertical: 10,
      }}
    >
      <Text
        style={{
          color: theme.colors.sidebarSecondary,
          fontFamily: theme.typography.mono.fontFamily,
          fontSize: 9,
          fontWeight: '700',
          letterSpacing: 0.6,
          marginBottom: 4,
        }}
      >
        {isIOS ? label : label.toUpperCase()}
      </Text>
      <Text
        style={{
          color: theme.colors.sidebarLabel,
          fontFamily: theme.typography.mono.fontFamily,
          fontSize: 12,
          fontWeight: '700',
        }}
      >
        {value}
      </Text>
      {footer}
    </View>
  );
}

export default function ManifestKpiGrid({
  compact = false,
  manifestId,
  maxVolumeVu,
  regionCode,
  state,
  stopCount = 0,
  theme,
  totalVolumeVu,
}: ManifestKpiGridProps) {
  const cap = Math.max(maxVolumeVu, 0.001);
  const pct = Math.min((totalVolumeVu / cap) * 100, 100);
  const barColor = pct > 95 ? '#EF4444' : pct > 80 ? '#F59E0B' : theme.colors.accent;

  return (
    <View style={{ gap: compact ? 8 : 10 }}>
      <View style={{ flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' }}>
        <StatusBadge compact label={state || 'DRAFT'} theme={theme} tone={manifestStateTone(state)} />
        <Text
          style={{
            color: theme.colors.sidebarSecondary,
            fontFamily: theme.typography.mono.fontFamily,
            fontSize: 9,
            letterSpacing: 0.3,
          }}
        >
          {manifestId.slice(0, 8)}
        </Text>
      </View>

      <View style={{ flexDirection: 'row', gap: 8 }}>
        <KpiTile
          label="Payload volume"
          theme={theme}
          value={`${totalVolumeVu.toFixed(1)} / ${maxVolumeVu.toFixed(1)} VU`}
          footer={
            <View
              style={{
                backgroundColor: theme.colors.fillSecondary,
                borderRadius: 3,
                height: 6,
                marginTop: 8,
                overflow: 'hidden',
              }}
            >
              <View
                style={{
                  backgroundColor: barColor,
                  borderRadius: 3,
                  height: 6,
                  width: `${pct}%`,
                }}
              />
            </View>
          }
        />
      </View>

      {(stopCount > 0 || regionCode) ? (
        <View style={{ flexDirection: 'row', gap: 8 }}>
          {stopCount > 0 ? (
            <KpiTile
              label="Target stops"
              theme={theme}
              value={`${stopCount} ${isIOS ? 'units' : 'UNITS'}`}
            />
          ) : null}
          {regionCode ? (
            <KpiTile
              label="Deployment zone"
              theme={theme}
              value={regionCode.toUpperCase()}
            />
          ) : null}
        </View>
      ) : null}
    </View>
  );
}
