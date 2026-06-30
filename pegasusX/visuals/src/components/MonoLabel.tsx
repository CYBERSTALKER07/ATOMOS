import type { CSSProperties } from 'react';

type MonoLabelProps = {
  children: string;
  style?: CSSProperties;
};

export const MonoLabel = ({ children, style }: MonoLabelProps) => (
  <span
    style={{
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
      fontSize: 14,
      letterSpacing: '0.2em',
      textTransform: 'uppercase',
      color: 'rgba(255, 255, 255, 0.6)',
      ...style,
    }}
  >
    {children}
  </span>
);
