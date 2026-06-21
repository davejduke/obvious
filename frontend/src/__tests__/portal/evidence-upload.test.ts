/**
 * Tests for evidence upload logic.
 *
 * Validates file type gating, size limits, and request queue state — all
 * tested as pure logic (no React rendering) to keep the suite fast.
 */

// Allowed MIME types — mirrors the definition in the detail page
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
const MAX_BYTES = MAX_FILE_SIZE_MB * 1024 * 1024;

function validateFile(type: string, sizeBytes: number): string | undefined {
  if (!ALLOWED_TYPES[type]) return `File type not allowed: ${type || 'unknown'}`;
  if (sizeBytes > MAX_BYTES) return `File exceeds ${MAX_FILE_SIZE_MB}MB limit`;
  return undefined;
}

describe('evidence upload — file type validation', () => {
  it('accepts CSV files', () => {
    expect(validateFile('text/csv', 1024)).toBeUndefined();
  });

  it('accepts JSON files', () => {
    expect(validateFile('application/json', 2048)).toBeUndefined();
  });

  it('accepts PDF files', () => {
    expect(validateFile('application/pdf', 1024 * 1024)).toBeUndefined();
  });

  it('accepts PNG images', () => {
    expect(validateFile('image/png', 500 * 1024)).toBeUndefined();
  });

  it('accepts XLSX spreadsheets', () => {
    expect(
      validateFile('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', 200 * 1024),
    ).toBeUndefined();
  });

  it('rejects disallowed MIME type (video/mp4)', () => {
    const err = validateFile('video/mp4', 1024);
    expect(err).toBeDefined();
    expect(err).toContain('File type not allowed');
  });

  it('rejects unknown MIME type (empty string)', () => {
    const err = validateFile('', 1024);
    expect(err).toBeDefined();
    expect(err).toContain('File type not allowed');
  });

  it('rejects executable files (.exe)', () => {
    const err = validateFile('application/x-msdownload', 1024);
    expect(err).toBeDefined();
  });
});

describe('evidence upload — file size validation', () => {
  it('accepts a file exactly at the 50MB limit', () => {
    expect(validateFile('application/pdf', MAX_BYTES)).toBeUndefined();
  });

  it('rejects a file 1 byte over the 50MB limit', () => {
    const err = validateFile('application/pdf', MAX_BYTES + 1);
    expect(err).toBeDefined();
    expect(err).toContain('exceeds');
  });

  it('accepts a small file well under the limit', () => {
    expect(validateFile('text/csv', 100)).toBeUndefined();
  });
});

describe('evidence request queue — priority sorting', () => {
  const {
    sortRequestsByPriority,
    mockEvidenceRequests,
    PORTAL_PRIORITY_ORDER,
  } = jest.requireActual<typeof import('@/lib/portal-mock-data')>(
    '@/lib/portal-mock-data',
  );

  it('sorts urgent before high before medium before low', () => {
    const sorted = sortRequestsByPriority(mockEvidenceRequests);
    for (let i = 0; i < sorted.length - 1; i++) {
      expect(PORTAL_PRIORITY_ORDER[sorted[i].priority]).toBeLessThanOrEqual(
        PORTAL_PRIORITY_ORDER[sorted[i + 1].priority],
      );
    }
  });

  it('PORTAL_PRIORITY_ORDER assigns 0 to urgent', () => {
    expect(PORTAL_PRIORITY_ORDER.urgent).toBe(0);
  });

  it('PORTAL_PRIORITY_ORDER assigns highest number to low', () => {
    const values = Object.values(PORTAL_PRIORITY_ORDER) as number[];
    expect(PORTAL_PRIORITY_ORDER.low).toBe(Math.max(...values));
  });

  it('does not mutate the original array', () => {
    const original = [...mockEvidenceRequests];
    sortRequestsByPriority(mockEvidenceRequests);
    expect(mockEvidenceRequests).toEqual(original);
  });
});
