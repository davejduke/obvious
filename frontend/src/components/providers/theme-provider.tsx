'use client';
import { useEffect } from 'react';
import { useThemeStore, initTheme } from '@/store/theme';

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const { mode, setMode, setResolvedDark } = useThemeStore();

  // On mount: load persisted preference
  useEffect(() => {
    const stored = initTheme();
    setMode(stored);
  }, [setMode]);

  // Apply dark/light class to <html> whenever mode or system preference changes
  useEffect(() => {
    const root = document.documentElement;

    function apply(dark: boolean) {
      if (dark) {
        root.classList.add('dark');
        root.classList.remove('light');
      } else {
        root.classList.add('light');
        root.classList.remove('dark');
      }
      setResolvedDark(dark);
    }

    if (mode === 'dark') {
      apply(true);
      return;
    }
    if (mode === 'light') {
      apply(false);
      return;
    }

    // system: follow prefers-color-scheme
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    apply(mq.matches);
    const handler = (e: MediaQueryListEvent) => apply(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, [mode, setResolvedDark]);

  return <>{children}</>;
}
