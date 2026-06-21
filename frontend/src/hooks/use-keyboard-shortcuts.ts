'use client';
import { useEffect, useCallback } from 'react';

export interface ShortcutDef {
  key: string;          // e.g. 'k', '?', 'Escape', 'ArrowDown'
  meta?: boolean;       // Cmd (Mac) / Ctrl (Win)
  ctrl?: boolean;       // Ctrl only
  shift?: boolean;
  description: string;
  group: string;
  handler: () => void;
}

/** Global registry — populated at runtime; exported so the overlay can read it */
export const globalShortcuts: ShortcutDef[] = [];

export function registerShortcut(def: ShortcutDef): () => void {
  globalShortcuts.push(def);
  return () => {
    const idx = globalShortcuts.indexOf(def);
    if (idx !== -1) globalShortcuts.splice(idx, 1);
  };
}

/**
 * useKeyboardShortcut — register a single shortcut tied to a component's lifetime.
 */
export function useKeyboardShortcut(def: Omit<ShortcutDef, 'handler'> & { handler: () => void }) {
  const stableHandler = useCallback(def.handler, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const unregister = registerShortcut({ ...def, handler: stableHandler });

    function onKeyDown(e: KeyboardEvent) {
      const metaOrCtrl = e.metaKey || e.ctrlKey;

      if (def.meta && !metaOrCtrl) return;
      if (!def.meta && metaOrCtrl && def.key !== 'Escape') return;
      if (def.ctrl && !e.ctrlKey) return;
      if (def.shift && !e.shiftKey) return;

      if (e.key.toLowerCase() !== def.key.toLowerCase()) return;

      // Don't fire shortcuts while typing in inputs/textareas unless Escape
      const tag = (e.target as HTMLElement)?.tagName;
      if (def.key !== 'Escape' && (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT')) return;

      e.preventDefault();
      stableHandler();
    }

    window.addEventListener('keydown', onKeyDown);
    return () => {
      window.removeEventListener('keydown', onKeyDown);
      unregister();
    };
  }, [def.key, def.meta, def.ctrl, def.shift, stableHandler]); // eslint-disable-line react-hooks/exhaustive-deps
}
