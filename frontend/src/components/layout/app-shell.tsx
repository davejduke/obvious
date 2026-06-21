'use client';
import { Sidebar } from './sidebar';
import { Header } from './header';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState } from 'react';

interface AppShellProps {
  children: React.ReactNode;
  title?: string;
}

export function AppShell({ children, title }: AppShellProps) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: { queries: { staleTime: 30_000, retry: 1 } },
  }));

  return (
    <QueryClientProvider client={queryClient}>
      <div className="flex h-screen overflow-hidden" style={{ backgroundColor: 'var(--bg-app)' }}>
        <Sidebar />
        <div className="flex flex-col flex-1 overflow-hidden">
          <Header title={title} />
          <main className="flex-1 overflow-y-auto">
            {children}
          </main>
        </div>
      </div>
    </QueryClientProvider>
  );
}
