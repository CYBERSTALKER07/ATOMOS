import { type ComponentProps } from 'react';
import { Pressable as RNPressable } from 'react-native';

const defaultPressFeedback = { opacity: 0.82, transform: [{ scale: 0.97 }] } as const;

export default function Pressable(props: ComponentProps<typeof RNPressable>) {
  const { style, ...rest } = props;
  return (
    <RNPressable
      {...rest}
      style={(state) => {
        if (typeof style === 'function') return style(state);
        return [style, state.pressed ? defaultPressFeedback : null];
      }}
    />
  );
}
