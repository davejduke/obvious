'use client';
import { usePersonaStore } from '@/store/persona';
import { personas } from '@/lib/mock-data';
import { Bell, ChevronDown } from 'lucide-react';
import { useState } from 'react';
import { clsx } from 'clsx';
import type { Persona } from '@shared/index';

export function Header({ title }: { title?: string }) {
  const { currentPersona, setPersona } = usePersonaStore();
  const [open, setOpen] = useState(false);
  const current = personas.find(p => p.id === currentPersona);

  return (
    <header className="flex items-center justify-between px-6 py-3 bg-white border-b border-slate-200">
      <div className="flex items-center gap-4">
        {title && <h1 className="text-lg font-semibold text-slate-900">{title}</h1>}
      </div>
      <div className="flex items-center gap-4">
        {/* Notification bell */}
        <button className="relative p-2 text-slate-400 hover:text-slate-600 rounded-md hover:bg-slate-100">
          <Bell size={18} />
          <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
        </button>

        {/* Persona switcher */}
        <div className="relative">
          <button
            onClick={() => setOpen(!open)}
            className="flex items-center gap-2 px-3 py-1.5 text-sm bg-slate-100 hover:bg-slate-200 rounded-md transition-colors"
          >
            <div className="w-6 h-6 bg-blue-600 rounded-full flex items-center justify-center text-white text-xs font-bold">
              {current?.label.charAt(0) ?? 'U'}
            </div>
            <span className="font-medium text-slate-700 max-w-32 truncate">{current?.label ?? 'Unknown'}</span>
            <ChevronDown size={14} className="text-slate-400" />
          </button>
          {open && (
            <div className="absolute right-0 mt-1 w-56 bg-white rounded-lg border border-slate-200 shadow-lg z-50">
              <div className="p-2">
                <p className="px-2 py-1 text-xs font-semibold text-slate-400 uppercase tracking-wide">Switch Persona</p>
                {personas.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => { setPersona(p.id as Persona); setOpen(false); }}
                    className={clsx(
                      'w-full flex flex-col items-start px-2 py-2 rounded text-sm transition-colors',
                      currentPersona === p.id
                        ? 'bg-blue-50 text-blue-700'
                        : 'text-slate-700 hover:bg-slate-50'
                    )}
                  >
                    <span className="font-medium">{p.label}</span>
                    <span className="text-xs text-slate-400">{p.description}</span>
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
