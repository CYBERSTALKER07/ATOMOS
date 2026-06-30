import type { ReactNode } from 'react';
import { AbsoluteFill } from 'remotion';
import { PEGASUS_VIDEO } from '../style/tokens';

type LineCanvasProps = {
  children: ReactNode;
};

export const LineCanvas = ({ children }: LineCanvasProps) => (
  <AbsoluteFill
    style={{
      backgroundColor: PEGASUS_VIDEO.background,
      justifyContent: 'center',
      alignItems: 'center',
    }}
  >
    <svg
      viewBox="0 0 1920 1080"
      width={PEGASUS_VIDEO.width}
      height={PEGASUS_VIDEO.height}
      style={{ display: 'block' }}
    >
      {children}
    </svg>
  </AbsoluteFill>
);
