'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { personas } from '@/lib/mock-data';
import { usePersonaStore } from '@/store/persona';
import { useState } from 'react';
import { User, Building2, Shield, Bell, Key, ChevronRight, Check } from 'lucide-react';
import { clsx } from 'clsx';
import type { Persona } from '@shared/index';

const sections = [
  { id: 'profile', label: 'Profile', icon: User },
  { id: 'org', label: 'Organisation', icon: Building2 },
  { id: 'rbac', label: 'Access & Permissions', icon: Shield },
  { id: 'notifications', label: 'Notifications', icon: Bell },
  { id: 'api', label: 'API Keys', icon: Key },
];

const permissionMatrix: Record<Persona, Record<string, boolean>> = {
  internal_auditor:  { view_findings: true,  edit_findings: true,  view_reports: true,  generate_reports: true,  manage_users: false, view_settings: true,  manage_org: false },
  cae:               { view_findings: true,  edit_findings: true,  view_reports: true,  generate_reports: true,  manage_users: true,  view_settings: true,  manage_org: true  },
  audit_committee:   { view_findings: true,  edit_findings: false, view_reports: true,  generate_reports: false, manage_users: false, view_settings: false, manage_org: false },
  auditee_ciso:      { view_findings: true,  edit_findings: false, view_reports: false, generate_reports: false, manage_users: false, view_settings: true,  manage_org: false },
  cosourced_provider:{ view_findings: true,  edit_findings: true,  view_reports: false, generate_reports: false, manage_users: false, view_settings: false, manage_org: false },
  ptwg_member:       { view_findings: false, edit_findings: false, view_reports: true,  generate_reports: false, manage_users: false, view_settings: false, manage_org: false },
  beta_tester:       { view_findings: true,  edit_findings: false, view_reports: true,  generate_reports: false, manage_users: false, view_settings: true,  manage_org: false },
};

const permissionLabels: Record<string, string> = {
  view_findings: 'View Findings',
  edit_findings: 'Edit Findings',
  view_reports: 'View Reports',
  generate_reports: 'Generate Reports',
  manage_users: 'Manage Users',
  view_settings: 'View Settings',
  manage_org: 'Manage Organisation',
};

export default function SettingsPage() {
  const [activeSection, setActiveSection] = useState('profile');
  const { currentPersona, setPersona } = usePersonaStore();

  return (
    <AppShell title="Settings">
      <div className="flex h-full">
        {/* Settings nav */}
        <div className="w-60 flex-shrink-0 border-r border-slate-200 bg-white">
          <div className="py-4">
            {sections.map(sec => {
              const Icon = sec.icon;
              return (
                <button
                  key={sec.id}
                  onClick={() => setActiveSection(sec.id)}
                  className={clsx(
                    'w-full flex items-center gap-3 px-4 py-2.5 text-sm transition-colors',
                    activeSection === sec.id
                      ? 'bg-blue-50 text-blue-700 font-medium border-r-2 border-blue-600'
                      : 'text-slate-600 hover:bg-slate-50'
                  )}
                >
                  <Icon size={16} className="flex-shrink-0" />
                  {sec.label}
                  {activeSection !== sec.id && <ChevronRight size={14} className="ml-auto text-slate-300" />}
                </button>
              );
            })}
          </div>
        </div>

        {/* Settings content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {activeSection === 'profile' && (
            <>
              <h2 className="text-lg font-semibold text-slate-900">Profile Settings</h2>
              <Card>
                <CardHeader><h3 className="font-semibold text-slate-900 text-sm">Personal Information</h3></CardHeader>
                <CardBody className="space-y-4">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    {[
                      { label: 'Display Name', placeholder: 'Alex Auditor', defaultValue: 'Alex Auditor' },
                      { label: 'Email', placeholder: 'alex@example.com', defaultValue: 'alex@example.com' },
                      { label: 'Job Title', placeholder: 'Senior Internal Auditor', defaultValue: 'Senior Internal Auditor' },
                      { label: 'Organisation', placeholder: 'Acme Corp', defaultValue: 'Acme Corp' },
                    ].map(f => (
                      <div key={f.label}>
                        <label className="block text-xs font-medium text-slate-700 mb-1">{f.label}</label>
                        <input defaultValue={f.defaultValue} placeholder={f.placeholder}
                          className="w-full px-3 py-2 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" />
                      </div>
                    ))}
                  </div>
                  <Button>Save Changes</Button>
                </CardBody>
              </Card>

              <Card>
                <CardHeader><h3 className="font-semibold text-slate-900 text-sm">Current Persona</h3></CardHeader>
                <CardBody>
                  <p className="text-xs text-slate-500 mb-3">Your persona determines your default dashboard view and available features.</p>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    {personas.map(p => (
                      <button key={p.id} onClick={() => setPersona(p.id as Persona)}
                        className={clsx(
                          'flex items-center gap-3 p-3 rounded-lg border-2 transition-colors text-left',
                          currentPersona === p.id ? 'border-blue-500 bg-blue-50' : 'border-slate-200 hover:border-slate-300'
                        )}>
                        <div className={clsx('w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold flex-shrink-0',
                          currentPersona === p.id ? 'bg-blue-600 text-white' : 'bg-slate-200 text-slate-600')}>
                          {p.label.charAt(0)}
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-slate-900">{p.label}</p>
                          <p className="text-xs text-slate-400 truncate">{p.description}</p>
                        </div>
                        {currentPersona === p.id && <Check size={16} className="text-blue-600 flex-shrink-0" />}
                      </button>
                    ))}
                  </div>
                </CardBody>
              </Card>
            </>
          )}

          {activeSection === 'rbac' && (
            <>
              <h2 className="text-lg font-semibold text-slate-900">Access & Permissions</h2>
              <p className="text-sm text-slate-500">7-persona permission matrix from UI/UX spec Section 8.</p>
              <Card>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead className="bg-slate-50">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">Permission</th>
                        {personas.map(p => (
                          <th key={p.id} className="px-3 py-3 text-center text-xs font-semibold text-slate-500">
                            <div className="w-20 mx-auto">{p.label}</div>
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {Object.entries(permissionLabels).map(([perm, label]) => (
                        <tr key={perm} className="hover:bg-slate-50">
                          <td className="px-4 py-3 font-medium text-slate-700">{label}</td>
                          {personas.map(p => {
                            const allowed = permissionMatrix[p.id as Persona]?.[perm] ?? false;
                            return (
                              <td key={p.id} className="px-3 py-3 text-center">
                                {allowed ? (
                                  <span className="inline-flex items-center justify-center w-5 h-5 bg-green-100 rounded-full">
                                    <Check size={12} className="text-green-600" />
                                  </span>
                                ) : (
                                  <span className="inline-flex items-center justify-center w-5 h-5 bg-slate-100 rounded-full">
                                    <span className="text-slate-300 text-xs">—</span>
                                  </span>
                                )}
                              </td>
                            );
                          })}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </Card>
            </>
          )}

          {activeSection === 'org' && (
            <>
              <h2 className="text-lg font-semibold text-slate-900">Organisation Settings</h2>
              <Card>
                <CardHeader><h3 className="font-semibold text-slate-900 text-sm">Organisation Details</h3></CardHeader>
                <CardBody className="space-y-4">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    {[
                      { label: 'Organisation Name', value: 'Acme Corp' },
                      { label: 'Slug', value: 'acme-corp' },
                      { label: 'Country', value: 'Germany' },
                      { label: 'Industry', value: 'Financial Services' },
                      { label: 'Tier', value: 'Enterprise' },
                      { label: 'NIS 2 Jurisdiction', value: 'EU — Essential Entity' },
                    ].map(f => (
                      <div key={f.label}>
                        <label className="block text-xs font-medium text-slate-700 mb-1">{f.label}</label>
                        <input defaultValue={f.value}
                          className="w-full px-3 py-2 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" />
                      </div>
                    ))}
                  </div>
                  <Button>Save Changes</Button>
                </CardBody>
              </Card>
            </>
          )}

          {(activeSection === 'notifications' || activeSection === 'api') && (
            <div className="text-center py-16 text-slate-400">
              <p className="text-lg font-medium mb-2">{sections.find(s => s.id === activeSection)?.label}</p>
              <p className="text-sm">Configuration coming in Phase 2.</p>
            </div>
          )}
        </div>
      </div>
    </AppShell>
  );
}
