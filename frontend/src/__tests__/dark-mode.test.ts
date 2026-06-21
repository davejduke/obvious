/**
 * Dark mode store tests
 */
import { useThemeStore, initTheme } from '../store/theme';

beforeEach(() => {
  localStorage.clear();
  useThemeStore.setState({ mode: 'system', resolvedDark: false });
  // Remove any theme classes
  document.documentElement.classList.remove('dark', 'light');
});

describe('Theme store', () => {
  it('defaults to system mode', () => {
    expect(useThemeStore.getState().mode).toBe('system');
  });

  it('setMode persists to localStorage', () => {
    useThemeStore.getState().setMode('dark');
    expect(useThemeStore.getState().mode).toBe('dark');
    expect(localStorage.getItem('aiauditor-theme')).toBe('dark');
  });

  it('setMode to light persists correctly', () => {
    useThemeStore.getState().setMode('light');
    expect(localStorage.getItem('aiauditor-theme')).toBe('light');
  });

  it('setMode to system persists correctly', () => {
    useThemeStore.getState().setMode('system');
    expect(localStorage.getItem('aiauditor-theme')).toBe('system');
  });

  it('initTheme reads stored preference', () => {
    localStorage.setItem('aiauditor-theme', 'dark');
    const result = initTheme();
    expect(result).toBe('dark');
  });

  it('initTheme defaults to system for unknown values', () => {
    localStorage.setItem('aiauditor-theme', 'invalid-value');
    const result = initTheme();
    expect(result).toBe('system');
  });

  it('initTheme defaults to system when nothing stored', () => {
    const result = initTheme();
    expect(result).toBe('system');
  });

  it('setResolvedDark tracks resolved state', () => {
    useThemeStore.getState().setResolvedDark(true);
    expect(useThemeStore.getState().resolvedDark).toBe(true);
    useThemeStore.getState().setResolvedDark(false);
    expect(useThemeStore.getState().resolvedDark).toBe(false);
  });

  it('all three modes are valid values', () => {
    const validModes = ['light', 'dark', 'system'] as const;
    for (const m of validModes) {
      useThemeStore.getState().setMode(m);
      expect(useThemeStore.getState().mode).toBe(m);
    }
  });
});
