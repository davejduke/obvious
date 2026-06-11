// Design system tokens - AIAUDITOR UI/UX Spec Section 2

export const colors = {
  risk: {
    critical: '#DC2626',
    high: '#EA580C',
    medium: '#CA8A04',
    low: '#16A34A',
    info: '#2563EB',
  },
  neutral: {
    50: '#F8FAFC',
    100: '#F1F5F9',
    200: '#E2E8F0',
    300: '#CBD5E1',
    400: '#94A3B8',
    500: '#64748B',
    600: '#475569',
    700: '#334155',
    800: '#1E293B',
    900: '#0F172A',
    950: '#020617',
  },
  brand: {
    primary: '#2563EB',
    primaryDark: '#1D4ED8',
    secondary: '#7C3AED',
    accent: '#0EA5E9',
  },
} as const;

export const severityBgClass: Record<string, string> = {
  critical: 'bg-red-100 text-red-800 border border-red-200',
  high: 'bg-orange-100 text-orange-800 border border-orange-200',
  medium: 'bg-yellow-100 text-yellow-800 border border-yellow-200',
  low: 'bg-green-100 text-green-800 border border-green-200',
  informational: 'bg-blue-100 text-blue-800 border border-blue-200',
};

export const severityDotClass: Record<string, string> = {
  critical: 'bg-red-600',
  high: 'bg-orange-500',
  medium: 'bg-yellow-500',
  low: 'bg-green-600',
  informational: 'bg-blue-500',
};

