declare module 'react-window' {
  import * as React from 'react';

  export interface ListChildComponentProps<T = unknown> {
    index: number;
    style: React.CSSProperties;
    data: T;
    isScrolling?: boolean;
  }

  export interface FixedSizeListProps<T = unknown> {
    height: number;
    itemCount: number;
    itemSize: number;
    width: number | string;
    itemData?: T;
    children: React.ComponentType<ListChildComponentProps<T>>;
    className?: string;
    overscanCount?: number;
    initialScrollOffset?: number;
    style?: React.CSSProperties;
  }

  export const FixedSizeList: <T = unknown>(props: FixedSizeListProps<T>) => React.ReactElement | null;
}
