import { useEffect, useMemo, useRef } from 'react';
import { Animated, StyleSheet, Text, View } from 'react-native';
import Svg, { Circle, Line, Path, Rect } from 'react-native-svg';

import type { AppTheme } from '../theme';

export type PayloadStateVariant = 'sync' | 'truck' | 'manifest' | 'dispatch' | 'notifications';

type PayloadStatePanelProps = {
  theme: AppTheme;
  variant: PayloadStateVariant;
  title: string;
  message?: string;
  detail?: string;
  compact?: boolean;
  tone?: 'default' | 'warning' | 'success';
};

function resolveAccent(theme: AppTheme, tone: NonNullable<PayloadStatePanelProps['tone']>) {
  if (tone === 'warning') {
    return theme.colors.destructive;
  }
  if (tone === 'success') {
    return theme.colors.success;
  }
  return theme.colors.accent;
}

function StateIllustration({
  accent,
  compact,
  secondary,
  surface,
  tertiary,
  variant,
}: {
  accent: string;
  compact: boolean;
  secondary: string;
  surface: string;
  tertiary: string;
  variant: PayloadStateVariant;
}) {
  const strokeWidth = compact ? 3 : 4;

  switch (variant) {
    case 'sync':
      return (
        <>
          <Circle cx="60" cy="60" r="38" stroke={tertiary} strokeWidth="2.5" strokeDasharray="6 7" fill="none" />
          <Path d="M60 20A40 40 0 0 1 96 44" stroke={accent} strokeWidth={strokeWidth} strokeLinecap="round" fill="none" />
          <Path d="M94 38l4 8-9 1" stroke={accent} strokeWidth={strokeWidth - 1} strokeLinecap="round" strokeLinejoin="round" fill="none" />
          <Rect x="36" y="30" width="48" height="60" rx="16" fill={surface} stroke={secondary} strokeWidth="2.5" />
          <Rect x="46" y="38" width="28" height="32" rx="8" fill={accent} opacity="0.16" />
          <Circle cx="60" cy="80" r="5" fill={accent} />
        </>
      );
    case 'truck':
      return (
        <>
          <Rect x="28" y="46" width="40" height="18" rx="5" fill={accent} opacity="0.18" />
          <Rect x="20" y="56" width="50" height="24" rx="8" fill={surface} stroke={secondary} strokeWidth="2.5" />
          <Path d="M70 60h14l12 12v8H70z" fill={surface} stroke={secondary} strokeWidth="2.5" strokeLinejoin="round" />
          <Rect x="76" y="64" width="10" height="8" rx="2" fill={accent} opacity="0.16" />
          <Circle cx="38" cy="84" r="8" fill={accent} />
          <Circle cx="82" cy="84" r="8" fill={accent} />
          <Circle cx="38" cy="84" r="3" fill={surface} />
          <Circle cx="82" cy="84" r="3" fill={surface} />
        </>
      );
    case 'manifest':
      return (
        <>
          <Rect x="34" y="22" width="52" height="76" rx="16" fill={surface} stroke={secondary} strokeWidth="2.5" />
          <Rect x="48" y="16" width="24" height="12" rx="6" fill={accent} opacity="0.2" />
          <Rect x="44" y="40" width="10" height="10" rx="3" fill={accent} opacity="0.18" />
          <Line x1="60" y1="45" x2="78" y2="45" stroke={secondary} strokeWidth="3" strokeLinecap="round" />
          <Rect x="44" y="58" width="10" height="10" rx="3" fill={accent} opacity="0.18" />
          <Line x1="60" y1="63" x2="78" y2="63" stroke={secondary} strokeWidth="3" strokeLinecap="round" />
          <Rect x="44" y="76" width="10" height="10" rx="3" fill={accent} opacity="0.18" />
          <Line x1="60" y1="81" x2="78" y2="81" stroke={secondary} strokeWidth="3" strokeLinecap="round" />
        </>
      );
    case 'dispatch':
      return (
        <>
          <Circle cx="32" cy="78" r="12" fill={surface} stroke={secondary} strokeWidth="2.5" />
          <Circle cx="88" cy="42" r="12" fill={surface} stroke={secondary} strokeWidth="2.5" />
          <Path d="M42 72C54 62 66 54 77 48" stroke={accent} strokeWidth={strokeWidth} strokeLinecap="round" fill="none" />
          <Path d="M70 38h18v18" stroke={accent} strokeWidth={strokeWidth - 1} strokeLinecap="round" strokeLinejoin="round" fill="none" />
          <Path d="M54 24l10 10-10 10" stroke={tertiary} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" fill="none" />
          <Path d="M48 34h18" stroke={tertiary} strokeWidth="2.5" strokeLinecap="round" fill="none" />
        </>
      );
    case 'notifications':
      return (
        <>
          <Path d="M60 24c-13 0-24 10-24 24v12l-8 12h64l-8-12V48c0-14-11-24-24-24Z" fill={surface} stroke={secondary} strokeWidth="2.5" strokeLinejoin="round" />
          <Circle cx="60" cy="86" r="7" fill={accent} opacity="0.22" />
          <Path d="M60 92c6 0 10-4 10-10H50c0 6 4 10 10 10Z" fill={accent} />
          <Circle cx="84" cy="30" r="8" fill={accent} />
          <Rect x="80" y="26" width="8" height="8" rx="4" fill={surface} />
        </>
      );
  }
}

export default function PayloadStatePanel({
  compact = false,
  detail,
  message,
  theme,
  title,
  tone = 'default',
  variant,
}: PayloadStatePanelProps) {
  const float = useRef(new Animated.Value(0)).current;
  const pulse = useRef(new Animated.Value(0)).current;
  const rotate = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    const floatLoop = Animated.loop(
      Animated.sequence([
        Animated.timing(float, { toValue: 1, duration: compact ? 1500 : 1800, useNativeDriver: true }),
        Animated.timing(float, { toValue: 0, duration: compact ? 1500 : 1800, useNativeDriver: true }),
      ])
    );

    const pulseLoop = Animated.loop(
      Animated.sequence([
        Animated.timing(pulse, { toValue: 1, duration: compact ? 1000 : 1200, useNativeDriver: true }),
        Animated.timing(pulse, { toValue: 0, duration: compact ? 1000 : 1200, useNativeDriver: true }),
      ])
    );

    const rotateLoop = Animated.loop(
      Animated.timing(rotate, { toValue: 1, duration: variant === 'sync' ? 3200 : 2600, useNativeDriver: true })
    );

    floatLoop.start();
    pulseLoop.start();
    if (variant === 'sync' || variant === 'dispatch') {
      rotateLoop.start();
    }

    return () => {
      floatLoop.stop();
      pulseLoop.stop();
      rotateLoop.stop();
      float.stopAnimation();
      pulse.stopAnimation();
      rotate.stopAnimation();
    };
  }, [compact, float, pulse, rotate, variant]);

  const accent = useMemo(() => resolveAccent(theme, tone), [theme, tone]);
  const translateY = float.interpolate({ inputRange: [0, 1], outputRange: [0, compact ? -4 : -6] });
  const scale = pulse.interpolate({ inputRange: [0, 1], outputRange: [0.985, 1.03] });
  const haloScale = pulse.interpolate({ inputRange: [0, 1], outputRange: [0.92, 1.12] });
  const haloOpacity = pulse.interpolate({ inputRange: [0, 1], outputRange: [0.28, 0.08] });
  const rotation = rotate.interpolate({ inputRange: [0, 1], outputRange: ['0deg', '360deg'] });
  const illustrationSize = compact ? 112 : 152;

  const illustrationTransforms = variant === 'sync' || variant === 'dispatch'
    ? ([{ translateY }, { scale }, { rotate: rotation }] as const)
    : ([{ translateY }, { scale }] as const);

  return (
    <View style={[styles.container, compact ? styles.containerCompact : null]}>
      <View style={[styles.illustrationWrap, { width: illustrationSize, height: illustrationSize }]}> 
        <Animated.View
          style={[
            styles.halo,
            {
              backgroundColor: theme.colors.fillSecondary,
              width: illustrationSize - 18,
              height: illustrationSize - 18,
              borderRadius: (illustrationSize - 18) / 2,
              opacity: haloOpacity,
              transform: [{ scale: haloScale }],
            },
          ]}
        />
        <Animated.View style={{ transform: illustrationTransforms }}>
          <Svg width={illustrationSize} height={illustrationSize} viewBox="0 0 120 120">
            <StateIllustration
              accent={accent}
              compact={compact}
              secondary={theme.colors.secondaryLabel}
              surface={theme.colors.cardBackground}
              tertiary={theme.colors.tertiaryLabel}
              variant={variant}
            />
          </Svg>
        </Animated.View>
      </View>
      <Text style={[styles.title, { color: theme.colors.label }, compact ? styles.titleCompact : null]}>{title}</Text>
      {message ? (
        <Text style={[styles.message, { color: theme.colors.secondaryLabel }, compact ? styles.messageCompact : null]}>{message}</Text>
      ) : null}
      {detail ? (
        <Text
          style={[
            styles.detail,
            { color: theme.colors.tertiaryLabel, fontFamily: theme.typography.mono.fontFamily },
            compact ? styles.detailCompact : null,
          ]}
        >
          {detail}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    justifyContent: 'center',
    maxWidth: 360,
  },
  containerCompact: {
    maxWidth: 260,
  },
  halo: {
    position: 'absolute',
  },
  illustrationWrap: {
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 20,
  },
  title: {
    fontSize: 22,
    fontWeight: '700',
    letterSpacing: 0.2,
    textAlign: 'center',
  },
  titleCompact: {
    fontSize: 16,
  },
  message: {
    marginTop: 10,
    fontSize: 13,
    lineHeight: 19,
    letterSpacing: 0.18,
    textAlign: 'center',
  },
  messageCompact: {
    fontSize: 12,
    lineHeight: 17,
  },
  detail: {
    marginTop: 12,
    fontSize: 11,
    letterSpacing: 0.45,
    textAlign: 'center',
  },
  detailCompact: {
    marginTop: 10,
  },
});