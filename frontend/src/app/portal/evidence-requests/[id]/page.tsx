'use client';
import { use, useState, useCallback } from 'react';
import { notFound } from 'next/navigation';
import { mockEvidenceRequests, getControlForFinding } from '@/lib/portal-mock-data';
import type { EvidenceRequest } from '@shared/index';
import { Card, CardHeader, CardBody } from '@/components/ui/card';
import { StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { usePortalStore } from '@/store/portal';
import {
  Upload,
  FileText,
  Image,
  File,
  X,
  CheckCircle2,
  AlertTriangle,
  Info,
  ChevronLeft,
  Loader2,
} from 'lucide-react';
import Link from 'next/link';
import { clsx } from 'clsx';

// Allowed MIME types and their display labels
const ALLOWED_TYPES: Record<string, string> = {
  'text/csv': 'CSV',
  'application/json': 'JSON',
  'application/pdf': 'PDF',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'XLSX',
  'application/vnd.ms-excel': 'XLS',
  'application/msword': 'DOC',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'DOCX',
  'image/png': 'PNG',
  'image/jpeg': 'JPEG',
  'image/gif': 'GIF',
  'image/webp': 'WEBP',
  'text/plain': 'TXT',
};

const MAX_FILE_SIZE_MB = 50;

interface PendingFile {
  id: string;
  file: File;
  error?: string;
}

function FileIcon({ type }: { type: string }) {
  if (type.startsWith('image/')) return <Image size={16} className="text-purple-500" />;
  if (type === 'application/pdf') return <FileText size={16} className="text-red-500" />;
  return <File size={16} className="text-slate-400" />;
}

function validateFile(file: File): string | undefined {
  if (!ALLOWED_TYPES[file.type]) {
    return `File type not allowed: ${file.type || 'unknown'}`;
  }
  if (file.size > MAX_FILE_SIZE_MB * 1024 * 1024) {
    return `File exceeds ${MAX_FILE_SIZE_MB}MB limit`;
  }
  return undefined;
}

const PRIORITY_BADGE: Record<EvidenceRequest['priority'], string> = {
  urgent: 'bg-red-100 text-red-800 border border-red-200',
  high: 'bg-orange-100 text-orange-800 border border-orange-200',
  medium: 'bg-yellow-100 text-yellow-800 border border-yellow-200',
  low: 'bg-slate-100 text-slate-600 border border-slate-200',
};

export default function EvidenceRequestDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const request = mockEvidenceRequests.find((r) => r.id === id);
  if (!request) notFound();

  const control = getControlForFinding(request.control_id);
  const { setUploadingRequestId } = usePortalStore();

  const [dragOver, setDragOver] = useState(false);
  const [pendingFiles, setPendingFiles] = useState<PendingFile[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(request.status === 'submitted' || request.status === 'accepted');
  const [note, setNote] = useState('');

  const addFiles = useCallback((files: FileList | File[]) => {
    const incoming = Array.from(files).map((f) => ({
      id: `${f.name}-${f.size}-${Date.now()}`,
      file: f,
      error: validateFile(f),
    }));
    setPendingFiles((prev) => [...prev, ...incoming]);
  }, []);

  const removeFile = (fileId: string) => {
    setPendingFiles((prev) => prev.filter((f) => f.id !== fileId));
  };

  async function handleSubmit() {
    const valid = pendingFiles.filter((f) => !f.error);
    if (valid.length === 0 && request!.evidence_ids.length === 0) return;
    setSubmitting(true);
    setUploadingRequestId(request!.id);
    // Simulate upload delay (production: POST /api/v1/portal/evidence-requests/:id/upload)
    await new Promise((resolve) => setTimeout(resolve, 1200));
    setSubmitting(false);
    setUploadingRequestId(null);
    setSubmitted(true);
    setPendingFiles([]);
  }

  const isReadOnly = submitted || request.status === 'accepted' || request.status === 'rejected';

  return (
    <div className="p-6 space-y-6 max-w-3xl">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-slate-500">
        <Link href="/portal/evidence-requests" className="hover:text-indigo-600 flex items-center gap-1">
          <ChevronLeft size={14} />
          Evidence Requests
        </Link>
        <span>/</span>
        <span className="text-slate-900 font-medium truncate">{request.title}</span>
      </div>

      {/* Request header */}
      <Card>
        <CardHeader>
          <div className="flex items-start gap-3">
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <span className={clsx('text-[10px] font-semibold px-1.5 py-0.5 rounded uppercase', PRIORITY_BADGE[request.priority])}>
                  {request.priority}
                </span>
                <StatusBadge status={request.status}>{request.status.replace('_', ' ')}</StatusBadge>
                {new Date(request.due_date) < new Date() && !submitted && (
                  <span className="text-[10px] font-bold text-red-600 bg-red-50 border border-red-200 px-1.5 py-0.5 rounded">
                    OVERDUE
                  </span>
                )}
              </div>
              <h2 className="text-lg font-semibold text-slate-900">{request.title}</h2>
              <p className="text-sm text-slate-500 mt-1">{request.description}</p>
            </div>
            <div className="text-right text-xs text-slate-400 flex-shrink-0">
              <p>Requested by</p>
              <p className="font-medium text-slate-700">{request.requested_by_email}</p>
              <p className="mt-1">Due: <span className="font-medium text-slate-700">{request.due_date}</span></p>
            </div>
          </div>
        </CardHeader>

        {request.instructions && (
          <CardBody className="border-t border-slate-100 bg-blue-50">
            <div className="flex items-start gap-2">
              <Info size={15} className="text-blue-500 mt-0.5 flex-shrink-0" />
              <div>
                <p className="text-xs font-semibold text-blue-800 mb-1">Auditor Instructions</p>
                <p className="text-sm text-blue-700">{request.instructions}</p>
              </div>
            </div>
          </CardBody>
        )}
      </Card>

      {/* Related control context */}
      {control && (
        <Card>
          <CardHeader>
            <h3 className="text-sm font-semibold text-slate-700 flex items-center gap-2">
              <Info size={14} className="text-indigo-500" />
              Related Control — Context
            </h3>
          </CardHeader>
          <CardBody>
            <div className="space-y-2">
              <div className="flex items-start gap-3">
                <span className="font-mono text-xs bg-indigo-50 text-indigo-700 px-2 py-0.5 rounded flex-shrink-0">
                  {control.control_id}
                </span>
                <span className="text-sm font-medium text-slate-900">{control.title}</span>
              </div>
              {control.description && (
                <p className="text-sm text-slate-500 pl-0">{control.description}</p>
              )}
              {control.objective && (
                <p className="text-xs text-slate-400 italic">Objective: {control.objective}</p>
              )}
            </div>
          </CardBody>
        </Card>
      )}

      {/* Submission success */}
      {submitted && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-5 flex items-center gap-4">
          <CheckCircle2 size={28} className="text-green-500 flex-shrink-0" />
          <div>
            <p className="font-semibold text-green-900">
              {request.status === 'accepted' ? 'Evidence accepted by auditor' : 'Evidence submitted for review'}
            </p>
            <p className="text-sm text-green-700 mt-0.5">
              {request.status === 'accepted'
                ? 'The audit team has reviewed and accepted your submission.'
                : 'The audit team will review your submission and follow up if needed.'}
            </p>
          </div>
        </div>
      )}

      {/* Upload zone */}
      {!isReadOnly && (
        <Card>
          <CardHeader>
            <h3 className="font-semibold text-slate-900 flex items-center gap-2">
              <Upload size={16} className="text-indigo-500" />
              Upload Evidence Files
            </h3>
            <p className="text-xs text-slate-400 mt-1">
              Accepted: CSV, JSON, PDF, XLSX, XLS, DOCX, PNG, JPEG, TXT — max {MAX_FILE_SIZE_MB}MB each
            </p>
          </CardHeader>
          <CardBody>
            {/* Drop zone */}
            <div
              onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
              onDragLeave={() => setDragOver(false)}
              onDrop={(e) => {
                e.preventDefault();
                setDragOver(false);
                if (e.dataTransfer.files) addFiles(e.dataTransfer.files);
              }}
              className={clsx(
                'border-2 border-dashed rounded-lg p-8 text-center transition-colors cursor-pointer',
                dragOver ? 'border-indigo-500 bg-indigo-50' : 'border-slate-200 hover:border-slate-300 bg-slate-50',
              )}
              onClick={() => document.getElementById('file-input')?.click()}
            >
              <Upload size={28} className="mx-auto mb-2 text-slate-400" />
              <p className="text-sm font-medium text-slate-700">Drop files here or click to browse</p>
              <p className="text-xs text-slate-400 mt-1">Multiple files supported</p>
              <input
                id="file-input"
                type="file"
                multiple
                accept={Object.keys(ALLOWED_TYPES).join(',')}
                className="hidden"
                onChange={(e) => e.target.files && addFiles(e.target.files)}
              />
            </div>

            {/* Pending files list */}
            {pendingFiles.length > 0 && (
              <ul className="mt-4 space-y-2">
                {pendingFiles.map((pf) => (
                  <li
                    key={pf.id}
                    className={clsx(
                      'flex items-center gap-3 px-4 py-2.5 rounded-lg border',
                      pf.error
                        ? 'border-red-200 bg-red-50'
                        : 'border-slate-200 bg-white',
                    )}
                  >
                    <FileIcon type={pf.file.type} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-slate-900 truncate">{pf.file.name}</p>
                      {pf.error ? (
                        <p className="text-xs text-red-600 flex items-center gap-1">
                          <AlertTriangle size={10} /> {pf.error}
                        </p>
                      ) : (
                        <p className="text-xs text-slate-400">
                          {ALLOWED_TYPES[pf.file.type] ?? 'Unknown'} — {(pf.file.size / 1024).toFixed(1)} KB
                        </p>
                      )}
                    </div>
                    <button
                      onClick={() => removeFile(pf.id)}
                      className="text-slate-400 hover:text-red-500 transition-colors"
                      aria-label="Remove file"
                    >
                      <X size={15} />
                    </button>
                  </li>
                ))}
              </ul>
            )}

            {/* Note */}
            <div className="mt-4">
              <label className="block text-sm font-medium text-slate-700 mb-1.5">
                Note to auditor <span className="text-slate-400 font-normal">(optional)</span>
              </label>
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                rows={3}
                placeholder="Add any context or caveats about the files you are submitting..."
                className="w-full px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none"
              />
            </div>

            {/* Submit */}
            <div className="flex items-center justify-end gap-3 mt-4 pt-4 border-t border-slate-100">
              <Link href="/portal/evidence-requests">
                <Button variant="secondary" size="md">Cancel</Button>
              </Link>
              <Button
                variant="primary"
                size="md"
                onClick={handleSubmit}
                disabled={
                  submitting ||
                  (pendingFiles.filter((f) => !f.error).length === 0 &&
                    request.evidence_ids.length === 0)
                }
              >
                {submitting ? (
                  <>
                    <Loader2 size={14} className="animate-spin" /> Uploading...
                  </>
                ) : (
                  <>
                    <Upload size={14} /> Submit Evidence
                  </>
                )}
              </Button>
            </div>
          </CardBody>
        </Card>
      )}

      {/* Already-submitted files */}
      {request.evidence_ids.length > 0 && (
        <Card>
          <CardHeader>
            <h3 className="text-sm font-semibold text-slate-700">Previously Submitted Files</h3>
          </CardHeader>
          <CardBody>
            <ul className="space-y-2">
              {request.evidence_ids.map((evId) => (
                <li key={evId} className="flex items-center gap-3 px-4 py-2.5 bg-slate-50 rounded-lg border border-slate-200">
                  <FileText size={15} className="text-slate-400" />
                  <span className="text-sm text-slate-700 font-mono">{evId}</span>
                  <CheckCircle2 size={13} className="text-green-500 ml-auto" />
                </li>
              ))}
            </ul>
          </CardBody>
        </Card>
      )}
    </div>
  );
}
