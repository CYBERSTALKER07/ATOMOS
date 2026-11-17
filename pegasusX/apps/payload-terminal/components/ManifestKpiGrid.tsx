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
  compact = false,
  footer,
  label,
  theme,
  value,
}: {
  compact?: boolean;
  footer?: ReactNode;
  label: string;
  theme: AppTheme;
  value: string;
}) {
  return (
    <View
      style={{
        flex: 1,
        backgroundColor: compact ? theme.colors.fillTertiary : theme.colors.cardBackground,
        borderRadius: theme.radius.card,
        borderWidth: compact ? 0 : (isIOS ? 0.33 : 1),
        borderColor: theme.colors.separator,
        paddingHorizontal: compact ? 12 : 20,
        paddingVertical: compact ? 10 : 16,
        ...(compact ? {} : theme.shadow.card),
      }}
    >
      <Text
        style={{
          color: compact ? theme.colors.sidebarSecondary : theme.colors.tertiaryLabel,
          fontFamily: theme.typography.mono.fontFamily,
          fontSize: compact ? 9 : 10,
          fontWeight: '900',
          letterSpacing: 0.8,
          marginBottom: compact ? 4 : 6,
        }}
      >
        {label.toUpperCase()}
      </Text>
      <Text
        style={{
          color: compact ? theme.colors.sidebarLabel : theme.colors.label,
          fontFamily: theme.typography.mono.fontFamily,
          fontSize: compact ? 12 : 18,
          fontWeight: '900',
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

      <View style={{ flexDirection: 'row', gap: compact ? 8 : 12 }}>
        <KpiTile
          compact={compact}
          label="Payload volume"
          theme={theme}
          value={`${totalVolumeVu.toFixed(1)} / ${maxVolumeVu.toFixed(1)} VU`}
          footer={
            <View
              style={{
                backgroundColor: theme.colors.fillSecondary,
                borderRadius: 3,
                height: 6,
                marginTop: compact ? 6 : 10,
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
        <View style={{ flexDirection: 'row', gap: compact ? 8 : 12 }}>
          {stopCount > 0 ? (
            <KpiTile
              compact={compact}
              label="Target stops"
              theme={theme}
              value={`${stopCount} ${isIOS ? 'units' : 'UNITS'}`}
            />
          ) : null}
          {regionCode ? (
            <KpiTile
              compact={compact}
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
