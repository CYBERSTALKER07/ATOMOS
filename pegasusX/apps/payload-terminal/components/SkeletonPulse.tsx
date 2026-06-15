import { useEffect, useRef } from 'react';
import { Animated, StyleSheet, View, type ViewStyle } from 'react-native';

import type { AppTheme } from '../theme';

type SkeletonPulseProps = {
  theme: AppTheme;
  height?: number;
  width?: number | `${number}%`;
  borderRadius?: number;
  style?: ViewStyle;
};

export function SkeletonPulse({
  borderRadius = 8,
  height = 12,
  style,
  theme,
  width = '100%',
}: SkeletonPulseProps) {
  const opacity = useRef(new Animated.Value(0.45)).current;

  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(opacity, { toValue: 0.9, duration: 700, useNativeDriver: true }),
        Animated.timing(opacity, { toValue: 0.45, duration: 700, useNativeDriver: true }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, [opacity]);

  return (
    <Animated.View
      style={[
        {
          backgroundColor: theme.colors.fillSecondary,
          borderRadius,
          height,
          opacity,
          width,
        },
        style,
      ]}
    />
  );
}

export function SkeletonList({ count = 3, theme }: { count?: number; theme: AppTheme }) {
  return (
    <View style={styles.list}>
      {Array.from({ length: count }).map((_, index) => (
        <View key={index} style={styles.row}>
          <SkeletonPulse theme={theme} height={40} width={40} borderRadius={10} />
          <View style={styles.copy}>
            <SkeletonPulse theme={theme} height={12} width="72%" />
            <SkeletonPulse theme={theme} height={10} width="48%" style={{ marginTop: 8 }} />
          </View>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  copy: {
    flex: 1,
    marginLeft: 12,
  },
  list: {
    gap: 14,
    paddingHorizontal: 20,
    paddingVertical: 16,
    width: '100%',
  },
  row: {
    alignItems: 'center',
    flexDirection: 'row',
  },
});
