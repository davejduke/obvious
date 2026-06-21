import { create } from 'zustand';

export type ThemeMode = 'light' | 'dark' | 'system';

interface ThemeState {
  mode: ThemeMode;
  setMode: (mode: ThemeMode) => void;
  resolvedDark: boolean;
  setResolvedDark: (dark: boolean) => void;
}

function loadStoredMode(): ThemeMode {
  if (typeof window === 'undefined') return 'system';
  const stored = localStorage.getItem('aiauditor-theme');
  if (stored === 'light' || stored === 'dark' || stored === 'system') return stored;
  return 'system';
}

export const useThemeStore = create<ThemeState>((set) => ({
  mode: 'system',
  resolvedDark: false,
  setMode: (mode) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('aiauditor-theme', mode);
    }
    set({ mode });
  },
  setResolvedDark: (dark) => set({ resolvedDark: dark }),
}));

export function initTheme(): ThemeMode {
  return loadStoredMode();
}
