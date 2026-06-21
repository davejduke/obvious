'use client';
import { useState } from 'react';
import { Bell, FileText, AlertTriangle, Clock, CheckCircle2, X, MailOpen } from 'lucide-react';
import { clsx } from 'clsx';
import { usePortalStore } from '@/store/portal';
import type { PortalNotification } from '@shared/index';
import Link from 'next/link';

function NotifIcon({ type }: { type: PortalNotification['type'] }) {
  if (type === 'evidence_request') return <FileText size={14} className="text-blue-600" />;
  if (type === 'finding_assignment') return <AlertTriangle size={14} className="text-orange-500" />;
  if (type === 'deadline_reminder') return <Clock size={14} className="text-red-500" />;
  if (type === 'response_acknowledged') return <CheckCircle2 size={14} className="text-green-600" />;
  return <Bell size={14} className="text-slate-400" />;
}

function notifHref(notif: PortalNotification): string {
  if (!notif.reference_id) return '#';
  if (notif.reference_type === 'evidence_request')
    return `/portal/evidence-requests/${notif.reference_id}`;
  if (notif.reference_type === 'finding')
    return `/portal/findings/${notif.reference_id}`;
  return '#';
}

export function NotificationBell() {
  const { notifications, unreadCount, markNotificationRead, markAllRead } = usePortalStore();
  const [open, setOpen] = useState(false);

  return (
    <div className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className={clsx(
          'relative p-2 rounded-lg transition-colors',
          open ? 'bg-slate-100' : 'hover:bg-slate-100',
        )}
        aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ''}`}
      >
        <Bell size={18} className="text-slate-600" />
        {unreadCount > 0 && (
          <span className="absolute top-1 right-1 min-w-[16px] h-4 bg-red-500 text-white text-[10px] font-bold rounded-full flex items-center justify-center px-0.5">
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-30"
            onClick={() => setOpen(false)}
            aria-hidden="true"
          />
          {/* Panel */}
          <div className="absolute right-0 top-10 z-40 w-80 bg-white rounded-xl shadow-xl border border-slate-200 overflow-hidden">
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100">
              <h3 className="text-sm font-semibold text-slate-900">Notifications</h3>
              <div className="flex items-center gap-2">
                {unreadCount > 0 && (
                  <button
                    onClick={markAllRead}
                    className="text-xs text-blue-600 hover:text-blue-700 flex items-center gap-1"
                  >
                    <MailOpen size={12} /> Mark all read
                  </button>
                )}
                <button onClick={() => setOpen(false)} className="text-slate-400 hover:text-slate-600">
                  <X size={14} />
                </button>
              </div>
            </div>

            <ul className="max-h-80 overflow-y-auto divide-y divide-slate-50">
              {notifications.length === 0 && (
                <li className="px-4 py-8 text-center text-sm text-slate-400">No notifications</li>
              )}
              {notifications.map((notif) => (
                <li key={notif.id}>
                  <Link
                    href={notifHref(notif)}
                    onClick={() => {
                      markNotificationRead(notif.id);
                      setOpen(false);
                    }}
                    className={clsx(
                      'flex items-start gap-3 px-4 py-3 hover:bg-slate-50 transition-colors',
                      !notif.is_read && 'bg-blue-50 hover:bg-blue-50',
                    )}
                  >
                    <div className="flex-shrink-0 mt-0.5 w-6 h-6 rounded-full bg-white border border-slate-200 flex items-center justify-center">
                      <NotifIcon type={notif.type} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className={clsx('text-xs font-medium leading-snug', notif.is_read ? 'text-slate-700' : 'text-slate-900')}>
                        {notif.title}
                      </p>
                      <p className="text-xs text-slate-500 mt-0.5 line-clamp-2">{notif.body}</p>
                      <p className="text-[10px] text-slate-400 mt-1">
                        {new Date(notif.created_at).toLocaleDateString()}
                      </p>
                    </div>
                    {!notif.is_read && (
                      <div className="flex-shrink-0 w-2 h-2 bg-blue-500 rounded-full mt-1" />
                    )}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </>
      )}
    </div>
  );
}
