'use client';
import { useEffect, useState } from 'react';
import { X, Command } from 'lucide-react';
import { globalShortcuts } from '@/hooks/use-keyboard-shortcuts';
import { clsx } from 'clsx';

interface ShortcutHelpOverlayProps {
  open: boolean;
  onClose: () => void;
}

function KeyBadge({ keys }: { keys: string[] }) {
  return (
    <span className="inline-flex items-center gap-0.5">
      {keys.map((k, i) => (
        <kbd
          key={i}
          className={clsx(
            'inline-flex items-center justify-center h-6 min-w-6 px-1.5 rounded text-xs font-mono font-semibold',
            'bg-[var(--bg-muted)] border border-[var(--border-strong)] text-[var(--text-secondary)]'
          )}
        >
          {k}
        </kbd>
      ))}
    </span>
  );
}

export function ShortcutHelpOverlay({ open, onClose }: ShortcutHelpOverlayProps) {
  const [groups, setGroups] = useState<Record<string, typeof globalShortcuts>>({});

  useEffect(() => {
    if (!open) return;
    const map: Record<string, typeof globalShortcuts> = {};
    for (const s of globalShortcuts) {
      if (!map[s.group]) map[s.group] = [];
      map[s.group].push(s);
    }
    setGroups(map);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;

  function formatKey(s: typeof globalShortcuts[0]): string[] {
    const parts: string[] = [];
    if (s.meta) parts.push('\u2318');
    if (s.shift) parts.push('\u21e7');
    const label = s.key === 'Escape' ? 'Esc' : s.key === '?' ? '?' : s.key.toUpperCase();
    parts.push(label);
    return parts;
  }

  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/50 backdrop-blur-sm"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="relative w-full max-w-lg mx-4 rounded-xl shadow-2xl bg-[var(--bg-surface)] border border-[var(--border-default)]">
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-default)]">
          <div className="flex items-center gap-2">
            <Command size={18} className="text-[var(--brand-600)]" />
            <h2 className="text-sm font-semibold text-[var(--text-primary)]">Keyboard Shortcuts</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-[var(--bg-hover)] text-[var(--text-muted)] transition-colors"
          >
            <X size={16} />
          </button>
        </div>

        <div className="px-6 py-4 max-h-[70vh] overflow-y-auto space-y-5">
          {Object.entries(groups).map(([group, shortcuts]) => (
            <div key={group}>
              <p className="text-xs font-semibold uppercase tracking-wider text-[var(--text-muted)] mb-2">
                {group}
              </p>
              <div className="space-y-1">
                {shortcuts.map((s, i) => (
                  <div key={i} className="flex items-center justify-between py-1.5">
                    <span className="text-sm text-[var(--text-secondary)]">{s.description}</span>
                    <KeyBadge keys={formatKey(s)} />
                  </div>
                ))}
              </div>
            </div>
          ))}
          {Object.keys(groups).length === 0 && (
            <p className="text-sm text-[var(--text-muted)] text-center py-4">No shortcuts registered.</p>
          )}
        </div>

        <div className="px-6 py-3 border-t border-[var(--border-default)] text-center">
          <span className="text-xs text-[var(--text-muted)]">
            Press <KeyBadge keys={['?']} /> to toggle this overlay
          </span>
        </div>
      </div>
    </div>
  );
}
