import { create } from 'zustand';

interface AppState {
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
  activeEngagementId: string | null;
  setActiveEngagement: (id: string | null) => void;
}

export const useAppStore = create<AppState>((set) => ({
  sidebarCollapsed: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  activeEngagementId: 'eng-001',
  setActiveEngagement: (id) => set({ activeEngagementId: id }),
}));
