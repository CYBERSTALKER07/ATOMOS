import { Easing, interpolate, useCurrentFrame, useVideoConfig } from 'remotion';
import { LineCanvas } from '../components/LineCanvas';
import { StrokeDraw } from '../components/StrokeDraw';
import { PEGASUS_VIDEO, secondsToFrames } from '../style/tokens';

const STEPS = [
  'PLACED',
  'VETTED',
  'LOADED',
  'SEALED',
  'IN TRANSIT',
  'ARRIVED',
  'PAID',
  'COMPLETED',
] as const;

const STEP_ICONS: Record<(typeof STEPS)[number], string> = {
  PLACED: 'M 200 520 h 40 v -30 h 25 l 15 15 v 15 h -40 z',
  VETTED: 'M 420 500 h 50 l 15 20 l 30 -50 h 50',
  LOADED: 'M 640 530 h 80 v -25 h 50 l 20 -15 h -70 z M 650 505 h 15 v 15',
  SEALED: 'M 860 510 h 40 v 30 h -40 z M 870 500 v -15 h 20 v 15',
  'IN TRANSIT': 'M 1080 520 h 120 M 1100 520 l 15 -10 M 1100 520 l 15 10',
  ARRIVED: 'M 1320 520 m -30 0 a 30 30 0 1 0 60 0 a 30 30 0 1 0 -60 0',
  PAID: 'M 1540 510 h 50 v 35 h -50 z M 1550 520 h 30',
  COMPLETED: 'M 1760 520 l 20 20 l 40 -45',
};

export const OrderLifecycle = () => {
  const frame = useCurrentFrame();
  const { fps, durationInFrames } = useVideoConfig();
  const stepDuration = secondsToFrames(1.2, fps);
  const activeStep = Math.min(
    STEPS.length - 1,
    Math.floor(frame / stepDuration)
  );

  const holdStart = durationInFrames - secondsToFrames(PEGASUS_VIDEO.holdSeconds, fps);
  const endOpacity = interpolate(frame, [holdStart, durationInFrames], [0.6, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
    easing: Easing.out(Easing.quad),
  });

  return (
    <LineCanvas>
      <StrokeDraw
        d="M 120 560 H 1800"
        drawStartFrame={0}
        drawDurationFrames={secondsToFrames(1, fps)}
        opacity={0.35}
      />

      {STEPS.map((step, index) => {
        const x = 200 + index * 220;
        const isActive = index === activeStep;
        const isPast = index < activeStep;

        const nodeOpacity = interpolate(
          frame,
          [index * stepDuration, index * stepDuration + 12],
          [0.25, isActive ? 1 : isPast ? 0.7 : 0.35],
          { extrapolateLeft: 'clamp', extrapolateRight: 'clamp' }
        );

        return (
          <g key={step} opacity={nodeOpacity}>
            <StrokeDraw
              d={`M ${x} 560 m -8 0 a 8 8 0 1 0 16 0 a 8 8 0 1 0 -16 0`}
              drawStartFrame={index * stepDuration}
              drawDurationFrames={18}
            />
            <StrokeDraw
              d={STEP_ICONS[step]}
              drawStartFrame={index * stepDuration + 6}
              drawDurationFrames={24}
            />
            <text
              x={x}
              y={620}
              textAnchor="middle"
              fill={PEGASUS_VIDEO.label}
              fontFamily="ui-monospace, monospace"
              fontSize={12}
              letterSpacing="3"
            >
              {step}
            </text>
          </g>
        );
      })}

      <text
        x={960}
        y={780}
        textAnchor="middle"
        fill={PEGASUS_VIDEO.stroke}
        opacity={endOpacity}
        fontFamily="ui-monospace, monospace"
        fontSize={18}
        letterSpacing="6"
      >
        ORDER LIFECYCLE
      </text>
    </LineCanvas>
  );
};
