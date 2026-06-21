'use client';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { clsx } from 'clsx';
import {
  LayoutDashboard,
  FileSearch,
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  Shield,
  ArrowLeft,
} from 'lucide-react';
import { useAppStore } from '@/store/app';
import { NotificationBell } from './notification-bell';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState } from 'react';

const portalNavItems = [
  { href: '/portal/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/portal/evidence-requests', label: 'Evidence Requests', icon: FileSearch },
  { href: '/portal/findings', label: 'Findings', icon: AlertTriangle },
];

function PortalSidebar() {
  const pathname = usePathname();
  const { sidebarCollapsed, toggleSidebar } = useAppStore();

  return (
    <aside
      className={clsx(
        'flex flex-col bg-indigo-900 text-white transition-all duration-200 flex-shrink-0',
        sidebarCollapsed ? 'w-16' : 'w-60',
      )}
    >
      {/* Logo */}
      <div className="flex items-center gap-3 px-4 py-4 border-b border-indigo-700">
        <div className="flex-shrink-0 w-8 h-8 bg-indigo-500 rounded-lg flex items-center justify-center">
          <Shield size={16} className="text-white" />
        </div>
        {!sidebarCollapsed && (
          <div>
            <span className="font-bold text-sm tracking-wide block">AIAUDITOR</span>
            <span className="text-[10px] text-indigo-300 uppercase tracking-widest">Auditee Portal</span>
          </div>
        )}
      </div>

      {/* Nav */}
      <nav className="flex-1 py-4">
        {portalNavItems.map((item) => {
          const active = pathname === item.href || pathname.startsWith(item.href + '/');
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={clsx(
                'flex items-center gap-3 px-4 py-2.5 mx-2 rounded-md text-sm transition-colors',
                active
                  ? 'bg-indigo-500 text-white'
                  : 'text-indigo-300 hover:text-white hover:bg-indigo-800',
              )}
              title={sidebarCollapsed ? item.label : undefined}
            >
              <Icon size={18} className="flex-shrink-0" />
              {!sidebarCollapsed && <span>{item.label}</span>}
            </Link>
          );
        })}

        {/* Back to main app */}
        <div className="mt-4 pt-4 border-t border-indigo-800 mx-2">
          <Link
            href="/dashboard"
            className="flex items-center gap-3 px-4 py-2 rounded-md text-xs text-indigo-400 hover:text-white hover:bg-indigo-800 transition-colors"
            title={sidebarCollapsed ? 'Back to Main App' : undefined}
          >
            <ArrowLeft size={14} className="flex-shrink-0" />
            {!sidebarCollapsed && <span>Back to Main App</span>}
          </Link>
        </div>
      </nav>

      {/* Collapse toggle */}
      <button
        onClick={toggleSidebar}
        className="flex items-center justify-center py-3 border-t border-indigo-800 text-indigo-400 hover:text-white transition-colors"
      >
        {sidebarCollapsed ? <ChevronRight size={16} /> : <ChevronLeft size={16} />}
      </button>
    </aside>
  );
}

interface PortalShellProps {
  children: React.ReactNode;
  title?: string;
}

export function PortalShell({ children, title }: PortalShellProps) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: { queries: { staleTime: 30_000, retry: 1 } },
  }));

  return (
    <QueryClientProvider client={queryClient}>
      <div className="flex h-screen overflow-hidden bg-slate-50">
        <PortalSidebar />
        <div className="flex flex-col flex-1 overflow-hidden">
          {/* Header */}
          <header className="h-14 flex items-center justify-between px-6 bg-white border-b border-slate-200 flex-shrink-0">
            <h1 className="text-base font-semibold text-slate-900">{title ?? 'Portal'}</h1>
            <div className="flex items-center gap-3">
              <NotificationBell />
              {/* Auditee identity stub */}
              <div className="flex items-center gap-2 pl-3 border-l border-slate-200">
                <div className="w-7 h-7 rounded-full bg-indigo-100 flex items-center justify-center">
                  <span className="text-xs font-semibold text-indigo-700">C</span>
                </div>
                <span className="text-xs text-slate-600">ciso@example.com</span>
              </div>
            </div>
          </header>
          <main className="flex-1 overflow-y-auto">{children}</main>
        </div>
      </div>
    </QueryClientProvider>
  );
}
