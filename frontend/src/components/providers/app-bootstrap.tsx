'use client';
import { useEffect, useState } from 'react';
import { hydrateOnboarding, useOnboardingStore } from '@/store/onboarding';
import { OnboardingWizard } from '@/components/onboarding/onboarding-wizard';
import { ShortcutHelpOverlay } from '@/components/shortcuts/shortcut-help-overlay';
import { useKeyboardShortcut } from '@/hooks/use-keyboard-shortcuts';

function ShortcutHelp() {
  const [open, setOpen] = useState(false);

  useKeyboardShortcut({
    key: '?',
    description: 'Show keyboard shortcuts',
    group: 'Help',
    handler: () => setOpen(v => !v),
  });

  return <ShortcutHelpOverlay open={open} onClose={() => setOpen(false)} />;
}

export function AppBootstrap({ children }: { children: React.ReactNode }) {
  const { completed } = useOnboardingStore();

  useEffect(() => {
    hydrateOnboarding();
  }, []);

  return (
    <>
      {!completed && <OnboardingWizard />}
      <ShortcutHelp />
      {children}
    </>
  );
}
