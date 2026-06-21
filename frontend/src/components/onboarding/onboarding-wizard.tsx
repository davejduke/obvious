'use client';
import { useState } from 'react';
import { useOnboardingStore } from '@/store/onboarding';
import { clsx } from 'clsx';
import {
  Building2, Users, Briefcase, BookOpen, LayoutDashboard,
  CheckCircle2, ChevronRight, ChevronLeft, X, Plus, Trash2
} from 'lucide-react';
import { Button } from '@/components/ui/button';

const STEPS = [
  { id: 0, label: 'Organisation', icon: Building2, title: 'Set up your organisation' },
  { id: 1, label: 'Team',         icon: Users,      title: 'Invite your team' },
  { id: 2, label: 'Engagement',  icon: Briefcase,  title: 'Create first engagement' },
  { id: 3, label: 'Framework',   icon: BookOpen,   title: 'Select audit framework' },
  { id: 4, label: 'Tour',        icon: LayoutDashboard, title: 'Explore your dashboard' },
] as const;

const FRAMEWORKS = [
  { id: 'nis2',   label: 'NIS 2 Article 21',      description: 'EU cybersecurity directive' },
  { id: 'iso27001', label: 'ISO 27001:2022',       description: 'Information security management' },
  { id: 'soc2',   label: 'SOC 2 Type II',          description: 'Trust services criteria' },
  { id: 'nist',   label: 'NIST CSF 2.0',           description: 'Cybersecurity framework' },
];

const INDUSTRIES = ['Financial Services', 'Healthcare', 'Energy & Utilities', 'Technology', 'Manufacturing', 'Government', 'Retail', 'Other'];

/* ---------- Step components ---------- */

function StepOrgProfile() {
  const { data, updateData } = useOnboardingStore();
  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-[var(--text-primary)] mb-1">Organisation name *</label>
        <input
          type="text"
          value={data.orgName}
          onChange={e => updateData({ orgName: e.target.value })}
          placeholder="Acme Corporation"
          className={clsx(
            'w-full px-3 py-2 rounded-lg border text-sm',
            'bg-[var(--bg-surface)] border-[var(--border-default)]',
            'text-[var(--text-primary)] placeholder:text-[var(--text-muted)]',
            'focus:outline-none focus:ring-2 focus:ring-[var(--ring-focus)]'
          )}
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-[var(--text-primary)] mb-1">Industry</label>
        <select
          value={data.industry}
          onChange={e => updateData({ industry: e.target.value })}
          className={clsx(
            'w-full px-3 py-2 rounded-lg border text-sm',
            'bg-[var(--bg-surface)] border-[var(--border-default)]',
            'text-[var(--text-primary)]',
            'focus:outline-none focus:ring-2 focus:ring-[var(--ring-focus)]'
          )}
        >
          <option value="">Select industry…</option>
          {INDUSTRIES.map(i => <option key={i} value={i}>{i}</option>)}
        </select>
      </div>
      <div>
        <label className="block text-sm font-medium text-[var(--text-primary)] mb-1">Country</label>
        <input
          type="text"
          value={data.country}
          onChange={e => updateData({ country: e.target.value })}
          placeholder="e.g. Germany"
          className={clsx(
            'w-full px-3 py-2 rounded-lg border text-sm',
            'bg-[var(--bg-surface)] border-[var(--border-default)]',
            'text-[var(--text-primary)] placeholder:text-[var(--text-muted)]',
            'focus:outline-none focus:ring-2 focus:ring-[var(--ring-focus)]'
          )}
        />
      </div>
    </div>
  );
}

function StepTeamInvites() {
  const { data, updateData } = useOnboardingStore();
  const [draft, setDraft] = useState('');

  function addEmail() {
    const email = draft.trim();
    if (!email || data.invites.includes(email)) return;
    updateData({ invites: [...data.invites, email] });
    setDraft('');
  }

  function removeEmail(email: string) {
    updateData({ invites: data.invites.filter(e => e !== email) });
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-[var(--text-secondary)]">
        Invite colleagues by email. They will receive an invitation to join your workspace.
      </p>
      <div className="flex gap-2">
        <input
          type="email"
          value={draft}
          onChange={e => setDraft(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); addEmail(); } }}
          placeholder="colleague@company.com"
          className={clsx(
            'flex-1 px-3 py-2 rounded-lg border text-sm',
            'bg-[var(--bg-surface)] border-[var(--border-default)]',
            'text-[var(--text-primary)] placeholder:text-[var(--text-muted)]',
            'focus:outline-none focus:ring-2 focus:ring-[var(--ring-focus)]'
          )}
        />
        <button
          onClick={addEmail}
          className={clsx(
            'flex items-center gap-1 px-3 py-2 rounded-lg text-sm font-medium',
            'bg-[var(--brand-600)] text-white hover:opacity-90 transition-opacity'
          )}
        >
          <Plus size={14} /> Add
        </button>
      </div>
      {data.invites.length > 0 && (
        <ul className="space-y-1">
          {data.invites.map(email => (
            <li key={email} className="flex items-center justify-between px-3 py-2 rounded-lg bg-[var(--bg-muted)] text-sm">
              <span className="text-[var(--text-primary)]">{email}</span>
              <button onClick={() => removeEmail(email)} className="text-[var(--text-muted)] hover:text-red-500 transition-colors">
                <Trash2 size={14} />
              </button>
            </li>
          ))}
        </ul>
      )}
      <p className="text-xs text-[var(--text-muted)]">You can also skip this and invite team members later from Settings.</p>
    </div>
  );
}

function StepFirstEngagement() {
  const { data, updateData } = useOnboardingStore();
  return (
    <div className="space-y-4">
      <p className="text-sm text-[var(--text-secondary)]">
        An engagement tracks an audit or continuous monitoring cycle. You can create more from the Engagement screen.
      </p>
      <div>
        <label className="block text-sm font-medium text-[var(--text-primary)] mb-1">Engagement name *</label>
        <input
          type="text"
          value={data.engagementName}
          onChange={e => updateData({ engagementName: e.target.value })}
          placeholder="e.g. NIS 2 Audit 2025"
          className={clsx(
            'w-full px-3 py-2 rounded-lg border text-sm',
            'bg-[var(--bg-surface)] border-[var(--border-default)]',
            'text-[var(--text-primary)] placeholder:text-[var(--text-muted)]',
            'focus:outline-none focus:ring-2 focus:ring-[var(--ring-focus)]'
          )}
        />
      </div>
    </div>
  );
}

function StepFrameworkSelection() {
  const { data, updateData } = useOnboardingStore();

  function toggle(id: string) {
    const next = data.frameworks.includes(id)
      ? data.frameworks.filter(f => f !== id)
      : [...data.frameworks, id];
    updateData({ frameworks: next });
  }

  return (
    <div className="space-y-3">
      <p className="text-sm text-[var(--text-secondary)]">
        Select the audit frameworks your organisation needs to comply with.
      </p>
      {FRAMEWORKS.map(fw => {
        const selected = data.frameworks.includes(fw.id);
        return (
          <button
            key={fw.id}
            onClick={() => toggle(fw.id)}
            className={clsx(
              'w-full flex items-start gap-3 p-3 rounded-lg border text-left transition-colors',
              selected
                ? 'border-[var(--brand-600)] bg-[var(--brand-50)]'
                : 'border-[var(--border-default)] hover:border-[var(--border-strong)] bg-[var(--bg-surface)]'
            )}
          >
            <div className={clsx(
              'flex-shrink-0 w-5 h-5 rounded-full border-2 mt-0.5 flex items-center justify-center transition-colors',
              selected ? 'border-[var(--brand-600)] bg-[var(--brand-600)]' : 'border-[var(--border-strong)]'
            )}>
              {selected && <CheckCircle2 size={12} className="text-white" />}
            </div>
            <div>
              <p className="text-sm font-medium text-[var(--text-primary)]">{fw.label}</p>
              <p className="text-xs text-[var(--text-muted)]">{fw.description}</p>
            </div>
          </button>
        );
      })}
    </div>
  );
}

function StepDashboardTour() {
  const tourItems = [
    { label: 'Dashboard',   desc: 'Persona-driven metrics and risk overview' },
    { label: 'Engagement',  desc: 'Track audit lifecycle from planning to reporting' },
    { label: 'Controls',    desc: 'NIS 2 Article 21 control framework mapping' },
    { label: 'Evidence',    desc: 'Ingest, classify, and score audit evidence' },
    { label: 'Reasoning',   desc: 'Deterministic AI confidence factors overlay' },
    { label: 'Findings',    desc: 'Track issues from discovery to remediation' },
    { label: 'Reports',     desc: 'Generate WYSIWYG audit reports' },
  ];

  return (
    <div className="space-y-3">
      <p className="text-sm text-[var(--text-secondary)]">
        Here is a quick overview of the main sections in your AIAUDITOR workspace:
      </p>
      <ul className="space-y-2">
        {tourItems.map(item => (
          <li key={item.label} className="flex items-start gap-2">
            <CheckCircle2 size={16} className="flex-shrink-0 mt-0.5 text-[var(--color-low)]" />
            <div>
              <span className="text-sm font-medium text-[var(--text-primary)]">{item.label}</span>
              <span className="text-sm text-[var(--text-muted)]"> — {item.desc}</span>
            </div>
          </li>
        ))}
      </ul>
      <p className="text-sm text-[var(--brand-600)] font-medium mt-2">
        You are all set. Click &ldquo;Finish setup&rdquo; to start auditing.
      </p>
    </div>
  );
}

const STEP_CONTENTS = [StepOrgProfile, StepTeamInvites, StepFirstEngagement, StepFrameworkSelection, StepDashboardTour];

/* ---------- Main wizard ---------- */

export function OnboardingWizard() {
  const { currentStep, nextStep, prevStep, complete, skip } = useOnboardingStore();
  const { data } = useOnboardingStore();

  const isLastStep = currentStep === 4;

  function canAdvance(): boolean {
    if (currentStep === 0) return data.orgName.trim().length > 0;
    if (currentStep === 2) return data.engagementName.trim().length > 0;
    return true;
  }

  const StepContent = STEP_CONTENTS[currentStep];
  const stepMeta = STEPS[currentStep];

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm"
      data-testid="onboarding-wizard"
    >
      <div className="relative w-full max-w-lg mx-4 rounded-2xl shadow-2xl overflow-hidden bg-[var(--bg-surface)] border border-[var(--border-default)]">
        {/* Skip */}
        <button
          onClick={skip}
          className="absolute top-4 right-4 p-1.5 rounded-lg text-[var(--text-muted)] hover:bg-[var(--bg-hover)] transition-colors z-10"
          title="Skip setup"
        >
          <X size={16} />
        </button>

        {/* Step indicator */}
        <div className="px-6 pt-6 pb-4">
          <div className="flex items-center gap-1.5 mb-5">
            {STEPS.map((s) => (
              <div
                key={s.id}
                className={clsx(
                  'h-1.5 flex-1 rounded-full transition-all duration-300',
                  s.id < currentStep
                    ? 'bg-[var(--brand-600)]'
                    : s.id === currentStep
                      ? 'bg-[var(--brand-600)] opacity-60'
                      : 'bg-[var(--border-default)]'
                )}
              />
            ))}
          </div>

          {/* Step header */}
          <div className="flex items-center gap-3 mb-1">
            <div className="w-8 h-8 rounded-lg bg-[var(--brand-50)] flex items-center justify-center">
              <stepMeta.icon size={16} className="text-[var(--brand-600)]" />
            </div>
            <div>
              <p className="text-xs text-[var(--text-muted)]">Step {currentStep + 1} of 5</p>
              <h2 className="text-base font-semibold text-[var(--text-primary)]">{stepMeta.title}</h2>
            </div>
          </div>
        </div>

        {/* Step content */}
        <div className="px-6 pb-4 min-h-[240px]">
          <StepContent />
        </div>

        {/* Navigation */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-[var(--border-default)] bg-[var(--bg-muted)]">
          <button
            onClick={prevStep}
            disabled={currentStep === 0}
            className={clsx(
              'flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm transition-colors',
              currentStep === 0
                ? 'opacity-0 pointer-events-none'
                : 'text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]'
            )}
          >
            <ChevronLeft size={16} /> Back
          </button>

          <div className="flex items-center gap-3">
            <span className="text-xs text-[var(--text-muted)]">
              {STEPS.map((_, i) => (
                <span
                  key={i}
                  className={clsx('inline-block w-1.5 h-1.5 rounded-full mx-0.5', i === currentStep ? 'bg-[var(--brand-600)]' : 'bg-[var(--border-strong)]')}
                />
              ))}
            </span>
          </div>

          {isLastStep ? (
            <button
              onClick={complete}
              className="flex items-center gap-1 px-4 py-1.5 rounded-lg text-sm font-medium bg-[var(--brand-600)] text-white hover:opacity-90 transition-opacity"
              data-testid="onboarding-finish"
            >
              Finish setup <CheckCircle2 size={14} />
            </button>
          ) : (
            <button
              onClick={nextStep}
              disabled={!canAdvance()}
              className={clsx(
                'flex items-center gap-1 px-4 py-1.5 rounded-lg text-sm font-medium transition-all',
                canAdvance()
                  ? 'bg-[var(--brand-600)] text-white hover:opacity-90'
                  : 'bg-[var(--border-default)] text-[var(--text-muted)] cursor-not-allowed'
              )}
              data-testid="onboarding-next"
            >
              Continue <ChevronRight size={14} />
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
