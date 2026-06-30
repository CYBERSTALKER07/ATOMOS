import { Easing, interpolate, useCurrentFrame } from 'remotion';
import { PEGASUS_VIDEO } from '../style/tokens';

type StrokeDrawProps = {
  d: string;
  drawStartFrame: number;
  drawDurationFrames: number;
  strokeWidth?: number;
  opacity?: number;
};

export const StrokeDraw = ({
  d,
  drawStartFrame,
  drawDurationFrames,
  strokeWidth = PEGASUS_VIDEO.strokeWidth,
  opacity = 1,
}: StrokeDrawProps) => {
  const frame = useCurrentFrame();

  const progress = interpolate(
    frame,
    [drawStartFrame, drawStartFrame + drawDurationFrames],
    [0, 1],
    {
      extrapolateLeft: 'clamp',
      extrapolateRight: 'clamp',
      easing: Easing.bezier(0.42, 0, 0.58, 1),
    }
  );

  return (
    <path
      d={d}
      fill="none"
      stroke={PEGASUS_VIDEO.stroke}
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      opacity={opacity}
      pathLength={1}
      strokeDasharray={1}
      strokeDashoffset={1 - progress}
    />
  );
};
