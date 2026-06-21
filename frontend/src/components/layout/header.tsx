'use client';
import { usePersonaStore } from '@/store/persona';
import { personas } from '@/lib/mock-data';
import { Bell, ChevronDown, Sun, Moon, Monitor } from 'lucide-react';
import { useState } from 'react';
import { clsx } from 'clsx';
import type { Persona } from '@shared/index';
import { GlobalSearch } from '@/components/search/global-search';
import { useThemeStore, type ThemeMode } from '@/store/theme';
import { useKeyboardShortcut } from '@/hooks/use-keyboard-shortcuts';

const THEME_ICONS: Record<ThemeMode, React.ElementType> = {
  light: Sun,
  dark:  Moon,
  system: Monitor,
};

const THEME_CYCLE: ThemeMode[] = ['light', 'dark', 'system'];

export function Header({ title }: { title?: string }) {
  const { currentPersona, setPersona } = usePersonaStore();
  const [personaOpen, setPersonaOpen] = useState(false);
  const current = personas.find(p => p.id === currentPersona);

  const { mode, setMode } = useThemeStore();
  const ThemeIcon = THEME_ICONS[mode];

  function cycleTheme() {
    const idx = THEME_CYCLE.indexOf(mode);
    setMode(THEME_CYCLE[(idx + 1) % THEME_CYCLE.length]);
  }

  // Keyboard shortcut: Escape closes open panels
  useKeyboardShortcut({
    key: 'Escape',
    description: 'Close open panels / modals',
    group: 'Navigation',
    handler: () => setPersonaOpen(false),
  });

  return (
    <header className="flex items-center justify-between px-6 py-3 bg-[var(--bg-header)] border-b border-[var(--border-header)]">
      <div className="flex items-center gap-4">
        {title && <h1 className="text-lg font-semibold text-[var(--text-primary)]">{title}</h1>}
      </div>
      <div className="flex items-center gap-3">
        {/* Global search (Cmd+K) */}
        <GlobalSearch />

        {/* Dark mode toggle */}
        <button
          onClick={cycleTheme}
          title={`Theme: ${mode}`}
          data-testid="dark-mode-toggle"
          className="p-2 rounded-md text-[var(--text-muted)] hover:text-[var(--text-secondary)] hover:bg-[var(--bg-hover)] transition-colors"
        >
          <ThemeIcon size={16} />
        </button>

        {/* Notification bell */}
        <button className="relative p-2 rounded-md text-[var(--text-muted)] hover:text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]">
          <Bell size={16} />
          <span className="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-red-500 rounded-full" />
        </button>

        {/* Persona switcher */}
        <div className="relative">
          <button
            onClick={() => setPersonaOpen(!personaOpen)}
            className="flex items-center gap-2 px-3 py-1.5 text-sm bg-[var(--bg-muted)] hover:bg-[var(--bg-hover)] rounded-md transition-colors"
          >
            <div className="w-6 h-6 bg-blue-600 rounded-full flex items-center justify-center text-white text-xs font-bold">
              {current?.label.charAt(0) ?? 'U'}
            </div>
            <span className="font-medium text-[var(--text-secondary)] max-w-32 truncate">{current?.label ?? 'Unknown'}</span>
            <ChevronDown size={14} className="text-[var(--text-muted)]" />
          </button>
          {personaOpen && (
            <div className="absolute right-0 mt-1 w-56 rounded-lg border shadow-lg z-50 bg-[var(--bg-surface)] border-[var(--border-default)]">
              <div className="p-2">
                <p className="px-2 py-1 text-xs font-semibold text-[var(--text-muted)] uppercase tracking-wide">Switch Persona</p>
                {personas.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => { setPersona(p.id as Persona); setPersonaOpen(false); }}
                    className={clsx(
                      'w-full flex flex-col items-start px-2 py-2 rounded text-sm transition-colors',
                      currentPersona === p.id
                        ? 'bg-[var(--brand-50)] text-[var(--brand-600)]'
                        : 'text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]'
                    )}
                  >
                    <span className="font-medium">{p.label}</span>
                    <span className="text-xs text-[var(--text-muted)]">{p.description}</span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
