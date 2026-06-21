import type { Metadata, Viewport } from 'next';
import './globals.css';
import { ThemeProvider } from '@/components/providers/theme-provider';
import { AppBootstrap } from '@/components/providers/app-bootstrap';

export const metadata: Metadata = {
  title: 'AIAUDITOR',
  description: 'Autonomous AI-powered cybersecurity audit platform — NIS 2 Article 21 compliance',
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <ThemeProvider>
          <AppBootstrap>
            {children}
          </AppBootstrap>
        </ThemeProvider>
      </body>
    </html>
  );
}
