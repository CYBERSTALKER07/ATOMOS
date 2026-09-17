'use client';

import * as React from 'react';
import { AnimateIcon, type AnimateIconProps } from './icon';

export {
  AnimateIcon,
  IconWrapper,
  useAnimateIconContext,
  getVariants,
  type IconProps,
  type IconWrapperProps,
  type AnimateIconProps,
  type AnimateIconContextValue,
} from './icon';

export { ArrowLeft, type ArrowLeftProps } from './arrow-left';
export { ArrowRight, type ArrowRightProps } from './arrow-right';
export { ArrowUp, type ArrowUpProps } from './arrow-up';
export { ArrowDown, type ArrowDownProps } from './arrow-down';
export { Check, type CheckProps } from './check';
export { CheckCheck, type CheckCheckProps } from './check-check';
export { CircleCheck, type CircleCheckProps } from './circle-check';
export { ChevronDown, type ChevronDownProps } from './chevron-down';
export { ChevronRight, type ChevronRightProps } from './chevron-right';
export { Clock, type ClockProps } from './clock';
export { Copy, type CopyProps } from './copy';
export { Layers, type LayersProps } from './layers';
export { Lock, type LockProps } from './lock';
export { Menu, type MenuProps } from './menu';
export { Moon, type MoonProps } from './moon';
export { RefreshCw, type RefreshCwProps } from './refresh-cw';
export { Search, type SearchProps } from './search';
export { Settings, type SettingsProps } from './settings';
export { Sparkles, type SparklesProps } from './sparkles';
export { Star, type StarProps } from './star';
export { Sun, type SunProps } from './sun';
export { Terminal, type TerminalProps } from './terminal';
export { Bell, type BellProps } from './bell';
export { User, type UserProps } from './user';
export { Blocks, type BlocksProps } from './blocks';
export { Bot, type BotProps } from './bot';
export { Activity, type ActivityProps } from './activity';

export type AnimatedIconProps = Omit<AnimateIconProps, 'children' | 'asChild'> & {
  icon: React.ComponentType<any>;
  size?: number | string;
  className?: string;
  [key: string]: any;
};

/**
 * Universal Animated Lucide Icon wrapper
 * Wraps any static icon with Animate UI motion triggers (hover, tap, view, loop)
 */
export function AnimatedIcon({
  icon: IconComponent,
  animateOnHover = true,
  animateOnTap = true,
  className,
  size = 20,
  ...props
}: AnimatedIconProps) {
  return (
    <AnimateIcon animateOnHover={animateOnHover} animateOnTap={animateOnTap} asChild {...props}>
      <IconComponent className={className} size={size} />
    </AnimateIcon>
  );
}
