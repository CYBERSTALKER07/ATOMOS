import { Platform, useColorScheme, type ColorSchemeName } from 'react-native';

export const isIOS = Platform.OS === 'ios';

// ─── iOS Light ──────────────────────────────────────────────────────────────
const iOSLight = {
  colors: {
    background: '#F2F2F7',
    cardBackground: '#FFFFFF',
    accent: '#6750A4',
    label: '#000000',
    secondaryLabel: '#3C3C43',
    tertiaryLabel: '#8E8E93',
    separator: 'rgba(60,60,67,0.12)',
    fillPrimary: 'rgba(120,120,128,0.2)',
    fillSecondary: 'rgba(120,120,128,0.16)',
    fillTertiary: 'rgba(120,120,128,0.12)',
    destructive: '#FF3B30',
    success: '#34C759',
    sidebarBackground: '#1C1C1E',
    sidebarLabel: '#FFFFFF',
    sidebarSecondary: 'rgba(235,235,245,0.3)',
    sidebarSeparator: 'rgba(84,84,88,0.65)',
    sidebarActive: '#FFFFFF',
    sidebarActiveText: '#000000',
  },
  radius: {
    card: 13,
    button: 12,
    checkbox: 6,
  },
  shadow: {
    card: {
      shadowColor: '#000',
      shadowOffset: { width: 0, height: 1 },
      shadowOpacity: 0.08,
      shadowRadius: 8,
    },
  },
  typography: {
    mono: { fontFamily: 'Menlo' },
    title: { fontSize: 20 },
    body: { fontSize: 15 },
    caption: { fontSize: 12 },
  },
};

// ─── Android Light ──────────────────────────────────────────────────────────
const androidLight = {
  colors: {
    background: '#FEF7FF',
    cardBackground: '#F7F2FA',
    accent: '#6750A4',
    label: '#1D1B20',
    secondaryLabel: '#49454F',
    tertiaryLabel: '#79747E',
    separator: '#CAC4D0',
    fillPrimary: '#E6E0E9',
    fillSecondary: '#ECE6F0',
    fillTertiary: '#F3EDF7',
    destructive: '#B3261E',
    success: '#006D3B',
    sidebarBackground: '#1D1B20',
    sidebarLabel: '#E6E0E9',
    sidebarSecondary: 'rgba(230,225,229,0.38)',
    sidebarSeparator: 'rgba(147,143,153,0.28)',
    sidebarActive: '#EADDFF',
    sidebarActiveText: '#1D1B20',
  },
  radius: {
    none: 0,
    extraSmall: 4,
    small: 8,
    medium: 12,
    large: 16,
    extraLarge: 28,
    full: 100,
    card: 12,
    button: 100,
    checkbox: 4,
  },
  shadow: {
    card: {
      elevation: 2,
    },
  },
  typography: {
    mono: { fontFamily: 'monospace' },
    title: { fontSize: 20 },
    body: { fontSize: 15 },
    caption: { fontSize: 12 },
  },
};

// ─── iOS Dark ───────────────────────────────────────────────────────────────
const iOSDark = {
  ...iOSLight,
  colors: {
    background: '#000000',
    cardBackground: '#1C1C1E',
    accent: '#D0BCFF',
    label: '#FFFFFF',
    secondaryLabel: 'rgba(235,235,245,0.6)',
    tertiaryLabel: 'rgba(235,235,245,0.3)',
    separator: 'rgba(84,84,88,0.65)',
    fillPrimary: 'rgba(120,120,128,0.36)',
    fillSecondary: 'rgba(120,120,128,0.32)',
    fillTertiary: 'rgba(120,120,128,0.24)',
    destructive: '#FF6961',
    success: '#30D158',
    sidebarBackground: '#000000',
    sidebarLabel: '#FFFFFF',
    sidebarSecondary: 'rgba(235,235,245,0.3)',
    sidebarSeparator: 'rgba(84,84,88,0.65)',
    sidebarActive: 'rgba(255,255,255,0.12)',
    sidebarActiveText: '#FFFFFF',
  },
};

// ─── Android Dark ───────────────────────────────────────────────────────────
const androidDark = {
  ...androidLight,
  colors: {
    background: '#141218',
    cardBackground: '#1D1B20',
    accent: '#D0BCFF',
    label: '#E6E0E9',
    secondaryLabel: '#CAC4D0',
    tertiaryLabel: '#938F99',
    separator: '#49454F',
    fillPrimary: '#36343B',
    fillSecondary: '#2B2930',
    fillTertiary: '#211F26',
    destructive: '#F2B8B5',
    success: '#86D993',
    sidebarBackground: '#0F0D13',
    sidebarLabel: '#E6E0E9',
    sidebarSecondary: 'rgba(230,224,233,0.38)',
    sidebarSeparator: 'rgba(147,143,153,0.28)',
    sidebarActive: '#4F378B',
    sidebarActiveText: '#EADDFF',
  },
};

export function getTheme(colorScheme: ColorSchemeName) {
  const isDark = colorScheme === 'dark';
  if (isIOS) return isDark ? iOSDark : iOSLight;
  return isDark ? androidDark : androidLight;
}

export type AppTheme = ReturnType<typeof getTheme>;

export function useT() {
  const colorScheme = useColorScheme();
  return getTheme(colorScheme);
}

export const T = getTheme('light');
