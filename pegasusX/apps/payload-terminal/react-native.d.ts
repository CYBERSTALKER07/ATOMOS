import type React from 'react';

declare module '@expo/vector-icons' {
    export interface MaterialIconsProps {
        name: string;
        size?: number;
        color?: string;
        style?: Record<string, unknown> | Array<Record<string, unknown>>;
    }
    export const MaterialIcons: React.ComponentType<MaterialIconsProps>;
}
