import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Persona } from '@shared/index';

interface PersonaState {
  currentPersona: Persona;
  setPersona: (persona: Persona) => void;
}

export const usePersonaStore = create<PersonaState>()(
  persist(
    (set) => ({
      currentPersona: 'internal_auditor',
      setPersona: (persona) => set({ currentPersona: persona }),
    }),
    { name: 'aiauditor-persona' }
  )
);
