import { Text, View } from 'react-native';

import type { AppTheme } from '../theme';
import { isIOS } from '../theme';

type WorkflowSectionHeaderProps = {
  title: string;
  theme: AppTheme;
  subtitle?: string;
  onDark?: boolean;
};

export default function WorkflowSectionHeader({
  onDark = false,
  subtitle,
  theme,
  title,
}: WorkflowSectionHeaderProps) {
  const labelColor = onDark ? theme.colors.sidebarLabel : theme.colors.label;
  const secondaryColor = onDark ? theme.colors.sidebarSecondary : theme.colors.tertiaryLabel;
  const displayTitle = isIOS ? title : title.toUpperCase();

  return (
    <View style={{ marginBottom: subtitle ? 6 : 8 }}>
      <Text
        style={{
          color: labelColor,
          fontSize: 11,
          fontWeight: '700',
          letterSpacing: isIOS ? 0.3 : 0.8,
        }}
      >
        {displayTitle}
      </Text>
      {subtitle ? (
        <Text
          style={{
            color: secondaryColor,
            fontSize: 10,
            marginTop: 2,
            letterSpacing: 0.2,
          }}
        >
          {subtitle}
        </Text>
      ) : null}
    </View>
  );
}
