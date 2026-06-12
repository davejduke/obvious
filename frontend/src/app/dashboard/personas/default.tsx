'use client';
import { Card, CardBody } from '@/components/ui/card';
import { personas } from '@/lib/mock-data';
import { usePersonaStore } from '@/store/persona';
import type { Persona } from '@shared/index';

export function DefaultDashboard() {
  const { setPersona } = usePersonaStore();
  return (
    <div className="p-6">
      <h2 className="text-xl font-semibold text-slate-900 mb-2">Select your persona</h2>
      <p className="text-slate-500 text-sm mb-6">Choose a persona to see your role-appropriate dashboard view.</p>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {personas.map(p => (
          <button key={p.id} onClick={() => setPersona(p.id as Persona)}
            className="text-left p-5 rounded-lg border-2 border-slate-200 hover:border-blue-400 hover:bg-blue-50 transition-colors group">
            <p className="font-semibold text-slate-900 group-hover:text-blue-700">{p.label}</p>
            <p className="text-sm text-slate-500 mt-1">{p.description}</p>
          </button>
        ))}
      </div>
    </div>
  );
}
