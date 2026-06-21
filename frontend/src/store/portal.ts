// Portal store — auditee session state and RBAC
import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Persona, PortalNotification } from '@shared/index';
import {
  mockPortalNotifications,
} from '@/lib/portal-mock-data';

// Personas that are permitted to access the /portal/ namespace
export const PORTAL_ALLOWED_PERSONAS: Persona[] = [
  'auditee_ciso',
  // Future: 'auditee_manager', 'auditee_owner'
];

export function isPortalAuthorized(persona: Persona): boolean {
  return PORTAL_ALLOWED_PERSONAS.includes(persona);
}

interface PortalState {
  // Notification state
  notifications: PortalNotification[];
  unreadCount: number;
  markNotificationRead: (id: string) => void;
  markAllRead: () => void;
  addNotification: (notif: PortalNotification) => void;
  // Upload tracking (stub — replaces backend state in prod)
  uploadingRequestId: string | null;
  setUploadingRequestId: (id: string | null) => void;
}

export const usePortalStore = create<PortalState>()(
  persist(
    (set) => ({
      notifications: mockPortalNotifications,
      unreadCount: mockPortalNotifications.filter((n) => !n.is_read).length,

      markNotificationRead: (id) =>
        set((s) => {
          const updated = s.notifications.map((n) =>
            n.id === id ? { ...n, is_read: true } : n,
          );
          return {
            notifications: updated,
            unreadCount: updated.filter((n) => !n.is_read).length,
          };
        }),

      markAllRead: () =>
        set((s) => ({
          notifications: s.notifications.map((n) => ({ ...n, is_read: true })),
          unreadCount: 0,
        })),

      addNotification: (notif) =>
        set((s) => ({
          notifications: [notif, ...s.notifications],
          unreadCount: s.unreadCount + (notif.is_read ? 0 : 1),
        })),

      uploadingRequestId: null,
      setUploadingRequestId: (id) => set({ uploadingRequestId: id }),
    }),
    { name: 'aiauditor-portal' },
  ),
);
