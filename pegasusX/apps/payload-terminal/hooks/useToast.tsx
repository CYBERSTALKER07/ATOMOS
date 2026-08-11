import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Animated, PanResponder, Text, View } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';

import Pressable from '../components/Pressable';
import { isIOS, type AppTheme } from '../theme';
import { clamp } from '../utils/math';

// ─── Types ────────────────────────────────────────────────────────────────────

export type UiToastTone = 'info' | 'success' | 'warning' | 'error';
export type UiToast = { id: number; title: string; message?: string; tone: UiToastTone };
export type ShowToast = (title: string, message?: string, tone?: UiToastTone, durationMs?: number) => void;

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useToast({ theme: T, isTabletLayout }: { theme: AppTheme; isTabletLayout: boolean }) {
  const toastMotionProfile = useMemo(
    () => isTabletLayout
      ? {
          startOffsetY: 18,
          hiddenOffsetY: 12,
          initialScale: 0.975,
          enterDuration: 250,
          exitDuration: 170,
          swipeDismissDuration: 190,
          swipeSnapDuration: 170,
          swipeStartThreshold: 8,
          swipeDismissDistance: 124,
          swipeDismissVelocity: 1.2,
          maxDragDistance: 260,
          swipeDismissTravel: 520,
          swipeSnapFriction: 8,
          swipeSnapTension: 110,
          dragOpacityDistance: 300,
          defaultDurationMs: 3200,
        }
      : {
          startOffsetY: 10,
          hiddenOffsetY: 7,
          initialScale: 0.97,
          enterDuration: 165,
          exitDuration: 115,
          swipeDismissDuration: 125,
          swipeSnapDuration: 110,
          swipeStartThreshold: 4,
          swipeDismissDistance: 76,
          swipeDismissVelocity: 0.88,
          maxDragDistance: 200,
          swipeDismissTravel: 340,
          swipeSnapFriction: 6,
          swipeSnapTension: 158,
          dragOpacityDistance: 220,
          defaultDurationMs: 2300,
        },
    [isTabletLayout]
  );

  // Lightweight in-app toast for non-blocking feedback.
  const [uiToast, setUiToast] = useState<UiToast | null>(null);
  const toastTranslateX = useRef(new Animated.Value(0)).current;
  const toastTranslateY = useRef(new Animated.Value(toastMotionProfile.startOffsetY)).current;
  const toastOpacity = useRef(new Animated.Value(0)).current;
  const toastScale = useRef(new Animated.Value(toastMotionProfile.initialScale)).current;
  const toastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const toastIdRef = useRef(0);
  const activeToastIdRef = useRef<number | null>(null);

  const clearToastTimer = useCallback(() => {
    if (toastTimerRef.current) {
      clearTimeout(toastTimerRef.current);
      toastTimerRef.current = null;
    }
  }, []);

  const clearToastImmediate = useCallback(() => {
    activeToastIdRef.current = null;
    setUiToast(null);
    toastTranslateX.setValue(0);
    toastTranslateY.setValue(toastMotionProfile.startOffsetY);
    toastOpacity.setValue(0);
    toastScale.setValue(toastMotionProfile.initialScale);
  }, [toastMotionProfile.initialScale, toastMotionProfile.startOffsetY, toastOpacity, toastScale, toastTranslateX, toastTranslateY]);

  const dismissToast = useCallback((immediate = false) => {
    clearToastTimer();
    if (immediate) {
      clearToastImmediate();
      return;
    }

    Animated.parallel([
      Animated.timing(toastOpacity, { toValue: 0, duration: toastMotionProfile.exitDuration, useNativeDriver: true }),
      Animated.timing(toastTranslateY, { toValue: toastMotionProfile.hiddenOffsetY, duration: toastMotionProfile.exitDuration, useNativeDriver: true }),
      Animated.timing(toastScale, { toValue: 0.985, duration: toastMotionProfile.exitDuration, useNativeDriver: true }),
    ]).start(() => clearToastImmediate());
  }, [clearToastImmediate, clearToastTimer, toastMotionProfile.exitDuration, toastMotionProfile.hiddenOffsetY, toastOpacity, toastScale, toastTranslateY]);

  const showToast = useCallback((title: string, message?: string, tone: UiToastTone = 'info', durationMs = toastMotionProfile.defaultDurationMs) => {
    clearToastTimer();

    const id = ++toastIdRef.current;
    activeToastIdRef.current = id;
    setUiToast({ id, title, message, tone });

    toastTranslateX.setValue(0);
    toastTranslateY.setValue(toastMotionProfile.startOffsetY);
    toastOpacity.setValue(0);
    toastScale.setValue(toastMotionProfile.initialScale);

    Animated.parallel([
      Animated.timing(toastOpacity, { toValue: 1, duration: toastMotionProfile.enterDuration, useNativeDriver: true }),
      Animated.timing(toastTranslateY, { toValue: 0, duration: toastMotionProfile.enterDuration, useNativeDriver: true }),
      Animated.timing(toastScale, { toValue: 1, duration: toastMotionProfile.enterDuration, useNativeDriver: true }),
    ]).start();

    toastTimerRef.current = setTimeout(() => {
      if (activeToastIdRef.current === id) {
        dismissToast();
      }
    }, durationMs);
  }, [clearToastTimer, dismissToast, toastMotionProfile.defaultDurationMs, toastMotionProfile.enterDuration, toastMotionProfile.initialScale, toastMotionProfile.startOffsetY, toastOpacity, toastScale, toastTranslateX, toastTranslateY]);

  const toastPanResponder = useMemo(
    () => PanResponder.create({
      onMoveShouldSetPanResponder: (_, gesture) => {
        if (!uiToast) return false;
        return Math.abs(gesture.dx) > toastMotionProfile.swipeStartThreshold && Math.abs(gesture.dx) > Math.abs(gesture.dy);
      },
      onPanResponderMove: (_, gesture) => {
        toastTranslateX.setValue(clamp(gesture.dx, -toastMotionProfile.maxDragDistance, toastMotionProfile.maxDragDistance));
        toastOpacity.setValue(Math.max(0.35, 1 - Math.abs(gesture.dx) / toastMotionProfile.dragOpacityDistance));
      },
      onPanResponderRelease: (_, gesture) => {
        const shouldDismiss =
          Math.abs(gesture.dx) > toastMotionProfile.swipeDismissDistance ||
          Math.abs(gesture.vx) > toastMotionProfile.swipeDismissVelocity;
        if (shouldDismiss) {
          clearToastTimer();
          Animated.parallel([
            Animated.timing(toastTranslateX, {
              toValue: gesture.dx >= 0 ? toastMotionProfile.swipeDismissTravel : -toastMotionProfile.swipeDismissTravel,
              duration: toastMotionProfile.swipeDismissDuration,
              useNativeDriver: true,
            }),
            Animated.timing(toastOpacity, { toValue: 0, duration: toastMotionProfile.swipeDismissDuration, useNativeDriver: true }),
          ]).start(() => dismissToast(true));
          return;
        }

        Animated.parallel([
          Animated.spring(toastTranslateX, {
            toValue: 0,
            friction: toastMotionProfile.swipeSnapFriction,
            tension: toastMotionProfile.swipeSnapTension,
            useNativeDriver: true,
          }),
          Animated.timing(toastOpacity, { toValue: 1, duration: toastMotionProfile.swipeSnapDuration, useNativeDriver: true }),
        ]).start();
      },
      onPanResponderTerminate: () => {
        Animated.parallel([
          Animated.spring(toastTranslateX, {
            toValue: 0,
            friction: toastMotionProfile.swipeSnapFriction,
            tension: toastMotionProfile.swipeSnapTension,
            useNativeDriver: true,
          }),
          Animated.timing(toastOpacity, { toValue: 1, duration: toastMotionProfile.swipeSnapDuration, useNativeDriver: true }),
        ]).start();
      },
    }),
    [
      clearToastTimer,
      dismissToast,
      toastMotionProfile.dragOpacityDistance,
      toastMotionProfile.maxDragDistance,
      toastMotionProfile.swipeDismissDistance,
      toastMotionProfile.swipeDismissDuration,
      toastMotionProfile.swipeDismissTravel,
      toastMotionProfile.swipeDismissVelocity,
      toastMotionProfile.swipeSnapDuration,
      toastMotionProfile.swipeSnapFriction,
      toastMotionProfile.swipeSnapTension,
      toastMotionProfile.swipeStartThreshold,
      toastOpacity,
      toastTranslateX,
      uiToast,
    ]
  );

  const renderUiToast = () => {
    if (!uiToast) return null;

    const toneStyles: Record<UiToastTone, {
      bg: string;
      border: string;
      title: string;
      message: string;
      icon: keyof typeof MaterialIcons.glyphMap;
    }> = {
      info: {
        bg: T.colors.cardBackground,
        border: T.colors.separator,
        title: T.colors.label,
        message: T.colors.secondaryLabel,
        icon: 'info-outline',
      },
      success: {
        bg: 'rgba(22, 163, 74, 0.12)',
        border: 'rgba(22, 163, 74, 0.35)',
        title: '#166534',
        message: '#15803D',
        icon: 'check-circle',
      },
      warning: {
        bg: 'rgba(245, 158, 11, 0.14)',
        border: 'rgba(245, 158, 11, 0.4)',
        title: '#92400E',
        message: '#B45309',
        icon: 'warning-amber',
      },
      error: {
        bg: 'rgba(239, 68, 68, 0.14)',
        border: 'rgba(239, 68, 68, 0.45)',
        title: '#991B1B',
        message: '#B91C1C',
        icon: 'error-outline',
      },
    };

    const tone = toneStyles[uiToast.tone];

    return (
      <View pointerEvents="box-none" style={{ position: 'absolute', left: 0, right: 0, bottom: 14, alignItems: 'center', zIndex: 1200, elevation: 1200 }}>
        <Animated.View
          {...toastPanResponder.panHandlers}
          style={{
            width: '86%',
            maxWidth: 560,
            minHeight: 62,
            borderRadius: isIOS ? 18 : 14,
            borderWidth: 1,
            borderColor: tone.border,
            backgroundColor: tone.bg,
            paddingHorizontal: 14,
            paddingVertical: 12,
            flexDirection: 'row',
            alignItems: 'flex-start',
            opacity: toastOpacity,
            transform: [
              { translateX: toastTranslateX },
              { translateY: toastTranslateY },
              { scale: toastScale },
            ],
            ...T.shadow.card,
          }}
        >
          <MaterialIcons name={tone.icon} size={18} color={tone.title} style={{ marginTop: 1, marginRight: 10 }} />
          <View style={{ flex: 1 }}>
            <Text style={{ color: tone.title, fontWeight: '700', fontSize: 13, letterSpacing: 0.2 }} numberOfLines={2}>
              {uiToast.title}
            </Text>
            {uiToast.message ? (
              <Text style={{ color: tone.message, fontSize: 12, marginTop: 3, lineHeight: 17 }} numberOfLines={2}>
                {uiToast.message}
              </Text>
            ) : null}
          </View>
          <Pressable onPress={() => dismissToast()} style={({ pressed }) => ({ marginLeft: 10, padding: 2, opacity: pressed ? 0.7 : 1 })}>
            <MaterialIcons name="close" size={16} color={tone.message} />
          </Pressable>
        </Animated.View>
      </View>
    );
  };

  useEffect(() => {
    return () => {
      clearToastTimer();
    };
  }, [clearToastTimer]);

  return { showToast, renderUiToast };
}
