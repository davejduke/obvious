'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { mockControls } from '@/lib/mock-data';
import { mockNIS2Score } from '@/lib/mock-data';
import { useState } from 'react';
import { Search, ChevronRight, ShieldCheck } from 'lucide-react';
import { clsx } from 'clsx';
import type { NIS2Article } from '@shared/index';

const articles: NIS2Article[] = ['21a','21b','21c','21d','21e','21f','21g','21h','21i','21j'];

const articleDescriptions: Record<NIS2Article, string> = {
  '21a': 'Risk analysis and information system security policies',
  '21b': 'Incident handling — detection, analysis, containment',
  '21c': 'Business continuity — backup management, disaster recovery',
  '21d': 'Supply chain security — relationships with direct suppliers',
  '21e': 'Security in network and information systems acquisition',
  '21f': 'Policies and procedures for assessing effectiveness',
  '21g': 'Basic cyber hygiene practices and cybersecurity training',
  '21h': 'Policies and procedures regarding cryptography',
  '21i': 'Human resources security, access control policies, asset management',
  '21j': 'Multi-factor authentication and secure communications',
};

function scoreColor(score: number) {
  if (score >= 80) return 'bg-green-500';
  if (score >= 60) return 'bg-yellow-500';
  return 'bg-red-500';
}

export default function ControlsPage() {
  const [selectedArticle, setSelectedArticle] = useState<NIS2Article | null>(null);
  const [search, setSearch] = useState('');

  const filteredControls = mockControls.filter(c => {
    if (selectedArticle && c.article_ref !== selectedArticle) return false;
    if (search && !c.title.toLowerCase().includes(search.toLowerCase()) && !c.control_id.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  return (
    <AppShell title="Control Browser">
      <div className="flex h-full">
        {/* Article tree sidebar */}
        <div className="w-72 flex-shrink-0 border-r border-slate-200 bg-white overflow-y-auto">
          <div className="px-4 py-3 border-b border-slate-100">
            <h3 className="font-semibold text-slate-900 text-sm">NIS 2 Article 21</h3>
            <p className="text-xs text-slate-400 mt-0.5">Control framework navigation</p>
          </div>
          <ul className="py-2">
            <li>
              <button
                onClick={() => setSelectedArticle(null)}
                className={clsx(
                  'w-full text-left px-4 py-2 text-sm hover:bg-slate-50 transition-colors flex items-center justify-between',
                  !selectedArticle && 'bg-blue-50 text-blue-700 font-medium'
                )}
              >
                All Controls
                <ChevronRight size={14} />
              </button>
            </li>
            {articles.map(art => {
              const data = mockNIS2Score.by_article[art];
              return (
                <li key={art}>
                  <button
                    onClick={() => setSelectedArticle(art)}
                    className={clsx(
                      'w-full text-left px-4 py-3 hover:bg-slate-50 transition-colors',
                      selectedArticle === art && 'bg-blue-50 border-r-2 border-blue-600'
                    )}
                  >
                    <div className="flex items-center justify-between">
                      <span className={clsx('text-xs font-bold', selectedArticle === art ? 'text-blue-700' : 'text-slate-900')}>
                        Art. {art.toUpperCase()}
                      </span>
                      <span className={clsx('text-xs font-semibold', data.score >= 75 ? 'text-green-600' : 'text-red-600')}>
                        {data.score}%
                      </span>
                    </div>
                    <p className="text-xs text-slate-500 mt-0.5 line-clamp-2">{articleDescriptions[art]}</p>
                    <div className="mt-2 h-1 bg-slate-100 rounded-full overflow-hidden">
                      <div className={clsx('h-full rounded-full', scoreColor(data.score))}
                        style={{ width: `${data.score}%` }} />
                    </div>
                  </button>
                </li>
              );
            })}
          </ul>
        </div>

        {/* Controls list */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          <div className="flex items-center gap-3">
            <div className="relative flex-1">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                placeholder="Search controls..."
                value={search}
                onChange={e => setSearch(e.target.value)}
                className="w-full pl-9 pr-4 py-2 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            {selectedArticle && (
              <span className="px-3 py-1.5 bg-blue-100 text-blue-700 text-xs font-semibold rounded-md">
                Article {selectedArticle.toUpperCase()}
              </span>
            )}
          </div>

          {filteredControls.length === 0 ? (
            <div className="text-center py-16 text-slate-400">
              <ShieldCheck size={40} className="mx-auto mb-3 opacity-30" />
              <p>No controls match your filters</p>
            </div>
          ) : (
            <div className="space-y-3">
              {filteredControls.map(ctrl => (
                <Card key={ctrl.id} className="hover:shadow-md transition-shadow">
                  <CardBody>
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="font-mono text-xs text-blue-700 bg-blue-50 px-2 py-0.5 rounded">{ctrl.control_id}</span>
                          {ctrl.article_ref && (
                            <span className="text-xs text-slate-500 bg-slate-100 px-2 py-0.5 rounded">Art. {ctrl.article_ref.toUpperCase()}</span>
                          )}
                          {ctrl.domain && (
                            <span className="text-xs text-purple-600 bg-purple-50 px-2 py-0.5 rounded">{ctrl.domain}</span>
                          )}
                        </div>
                        <h4 className="font-semibold text-slate-900">{ctrl.title}</h4>
                        {ctrl.description && <p className="text-sm text-slate-500 mt-1">{ctrl.description}</p>}
                        {ctrl.objective && (
                          <p className="text-xs text-slate-400 mt-2 italic">Objective: {ctrl.objective}</p>
                        )}
                      </div>
                      <div className="text-right flex-shrink-0">
                        <p className="text-xs text-slate-400">Risk Weight</p>
                        <p className="text-lg font-bold text-slate-900">{Math.round(ctrl.risk_weight * 100)}%</p>
                        <div className="flex gap-1 mt-1 flex-wrap justify-end">
                          {ctrl.tags.map(t => (
                            <span key={t} className="text-xs text-slate-400 bg-slate-50 border border-slate-200 px-1.5 py-0.5 rounded">{t}</span>
                          ))}
                        </div>
                      </div>
                    </div>
                  </CardBody>
                </Card>
              ))}
            </div>
          )}

          {/* Show all controls hint */}
          {filteredControls.length > 0 && (
            <p className="text-xs text-slate-400 text-center py-2">
              Showing {filteredControls.length} controls · Full NIS 2 framework has 65+ controls
            </p>
          )}
        </div>
      </div>
    </AppShell>
  );
}
