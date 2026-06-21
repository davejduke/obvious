'use client';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { clsx } from 'clsx';
import {
  LayoutDashboard, Briefcase, ShieldCheck, FileSearch,
  Network, AlertTriangle, FileText, Settings, ChevronLeft,
  ChevronRight, Activity, CalendarDays
} from 'lucide-react';
import { useAppStore } from '@/store/app';

const navItems = [
  { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/engagement', label: 'Engagement', icon: Briefcase },
  { href: '/planning', label: 'Planning', icon: CalendarDays },
  { href: '/controls', label: 'Controls', icon: ShieldCheck },
  { href: '/evidence', label: 'Evidence', icon: FileSearch },
  { href: '/reasoning', label: 'Reasoning', icon: Network },
  { href: '/findings', label: 'Findings', icon: AlertTriangle },
  { href: '/reports', label: 'Reports', icon: FileText },
  { href: '/settings', label: 'Settings', icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();
  const { sidebarCollapsed, toggleSidebar } = useAppStore();

  return (
    <aside className={clsx(
      'flex flex-col bg-slate-900 text-white transition-all duration-200 flex-shrink-0',
      sidebarCollapsed ? 'w-16' : 'w-60'
    )}>
      {/* Logo */}
      <div className="flex items-center gap-3 px-4 py-4 border-b border-slate-700">
        <div className="flex-shrink-0 w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
          <Activity size={16} className="text-white" />
        </div>
        {!sidebarCollapsed && (
          <span className="font-bold text-sm tracking-wide">AIAUDITOR</span>
        )}
      </div>

      {/* Nav */}
      <nav className="flex-1 py-4">
        {navItems.map((item) => {
          const active = pathname === item.href || pathname.startsWith(item.href + '/');
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={clsx(
                'flex items-center gap-3 px-4 py-2.5 mx-2 rounded-md text-sm transition-colors',
                active
                  ? 'bg-blue-600 text-white'
                  : 'text-slate-400 hover:text-white hover:bg-slate-800'
              )}
              title={sidebarCollapsed ? item.label : undefined}
            >
              <Icon size={18} className="flex-shrink-0" />
              {!sidebarCollapsed && <span>{item.label}</span>}
            </Link>
          );
        })}
      </nav>

      {/* Collapse toggle */}
      <button
        onClick={toggleSidebar}
        className="flex items-center justify-center py-3 border-t border-slate-700 text-slate-400 hover:text-white transition-colors"
      >
        {sidebarCollapsed ? <ChevronRight size={16} /> : <ChevronLeft size={16} />}
      </button>
    </aside>
  );
}
