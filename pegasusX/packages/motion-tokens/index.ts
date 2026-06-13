/**
 * Shared motion design tokens for pegasusX web surfaces (Next.js + Tauri).
 * Aligns with MDC Motion 1.14 duration/easing slots used by Android MotionTokens.
 */

export const duration = {
  short1: 0.05,
  short2: 0.1,
  short3: 0.15,
  short4: 0.2,
  medium1: 0.25,
  medium2: 0.3,
  medium3: 0.35,
  medium4: 0.4,
  long1: 0.45,
  long2: 0.5,
  long3: 0.55,
  long4: 0.6,
} as const;

export type CubicBezierEasing = [number, number, number, number];

export type MotionTransition = {
  duration?: number;
  ease?: CubicBezierEasing;
};

export const easing: Record<string, CubicBezierEasing> = {
  standard: [0.2, 0, 0, 1],
  standardDecelerate: [0, 0, 0, 1],
  standardAccelerate: [0.3, 0, 1, 1],
  emphasizedDecelerate: [0.05, 0.7, 0.1, 1],
  emphasizedAccelerate: [0.3, 0, 0.8, 0.15],
  linear: [0, 0, 1, 1],
};

export const spring = {
  fast: { type: "spring" as const, stiffness: 3800, damping: 30 },
  default: { type: "spring" as const, stiffness: 1600, damping: 28 },
  slow: { type: "spring" as const, stiffness: 800, damping: 26 },
};

export const motionVariants = {
  pageEnter: {
    initial: { opacity: 0, y: 4 },
    animate: { opacity: 1, y: 0 },
    exit: { opacity: 0, y: -4 },
    transition: { duration: duration.short4, ease: easing.standard },
  },
  modalEnter: {
    initial: { opacity: 0, scale: 0.97 },
    animate: { opacity: 1, scale: 1 },
    exit: { opacity: 0, scale: 0.97 },
    transition: { duration: duration.medium2, ease: easing.emphasizedDecelerate },
  },
  sheetEnter: {
    initial: { opacity: 0, y: 24 },
    animate: { opacity: 1, y: 0 },
    exit: { opacity: 0, y: 24 },
    transition: { duration: duration.medium4, ease: easing.emphasizedDecelerate },
  },
  listItem: (index: number) => ({
    initial: { opacity: 0, y: 8 },
    animate: { opacity: 1, y: 0 },
    transition: {
      duration: duration.short4,
      ease: easing.emphasizedDecelerate,
      delay: Math.min(index * 0.04, 0.4),
    },
  }),
};

export function reducedMotionTransition(
  prefersReduced: boolean,
  transition: MotionTransition,
): MotionTransition {
  if (prefersReduced) {
    return { duration: 0.01 };
  }
  return transition;
}
