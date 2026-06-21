import { create } from 'zustand';

export type OnboardingStep = 0 | 1 | 2 | 3 | 4;

export interface OnboardingData {
  orgName: string;
  industry: string;
  country: string;
  invites: string[];       // comma-separated emails entered per step
  engagementName: string;
  frameworks: string[];    // selected framework ids
}

interface OnboardingState {
  completed: boolean;
  skipped: boolean;
  currentStep: OnboardingStep;
  data: OnboardingData;
  setStep: (step: OnboardingStep) => void;
  nextStep: () => void;
  prevStep: () => void;
  updateData: (patch: Partial<OnboardingData>) => void;
  complete: () => void;
  skip: () => void;
  reset: () => void;
}

const DEFAULT_DATA: OnboardingData = {
  orgName: '',
  industry: '',
  country: '',
  invites: [],
  engagementName: '',
  frameworks: [],
};

function loadCompleted(): boolean {
  if (typeof window === 'undefined') return false;
  return localStorage.getItem('aiauditor-onboarding-done') === 'true';
}

export const useOnboardingStore = create<OnboardingState>((set, get) => ({
  completed: false,  // hydrated on mount
  skipped: false,
  currentStep: 0,
  data: { ...DEFAULT_DATA },

  setStep: (step) => set({ currentStep: step }),
  nextStep: () => {
    const { currentStep } = get();
    if (currentStep < 4) set({ currentStep: (currentStep + 1) as OnboardingStep });
  },
  prevStep: () => {
    const { currentStep } = get();
    if (currentStep > 0) set({ currentStep: (currentStep - 1) as OnboardingStep });
  },
  updateData: (patch) => set((s) => ({ data: { ...s.data, ...patch } })),
  complete: () => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('aiauditor-onboarding-done', 'true');
    }
    set({ completed: true });
  },
  skip: () => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('aiauditor-onboarding-done', 'true');
    }
    set({ skipped: true, completed: true });
  },
  reset: () => {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('aiauditor-onboarding-done');
    }
    set({ completed: false, skipped: false, currentStep: 0, data: { ...DEFAULT_DATA } });
  },
}));

export function hydrateOnboarding() {
  const done = loadCompleted();
  useOnboardingStore.setState({ completed: done });
}
