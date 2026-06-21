/**
 * Onboarding wizard tests
 */
import { renderHook, act } from '@testing-library/react';
import { useOnboardingStore, hydrateOnboarding } from '../store/onboarding';

// Reset store and localStorage before each test
beforeEach(() => {
  localStorage.clear();
  useOnboardingStore.getState().reset();
});

describe('Onboarding store', () => {
  it('starts at step 0 with empty data', () => {
    const { result } = renderHook(() => useOnboardingStore());
    expect(result.current.currentStep).toBe(0);
    expect(result.current.data.orgName).toBe('');
    expect(result.current.completed).toBe(false);
  });

  it('advances through all 5 steps', () => {
    const { result } = renderHook(() => useOnboardingStore());
    act(() => result.current.nextStep()); // 0 → 1
    expect(result.current.currentStep).toBe(1);
    act(() => result.current.nextStep()); // 1 → 2
    act(() => result.current.nextStep()); // 2 → 3
    act(() => result.current.nextStep()); // 3 → 4
    expect(result.current.currentStep).toBe(4);
  });

  it('does not advance past step 4', () => {
    const { result } = renderHook(() => useOnboardingStore());
    for (let i = 0; i < 10; i++) act(() => result.current.nextStep());
    expect(result.current.currentStep).toBe(4);
  });

  it('goes back via prevStep', () => {
    const { result } = renderHook(() => useOnboardingStore());
    act(() => result.current.nextStep());
    act(() => result.current.nextStep());
    expect(result.current.currentStep).toBe(2);
    act(() => result.current.prevStep());
    expect(result.current.currentStep).toBe(1);
  });

  it('does not go below step 0', () => {
    const { result } = renderHook(() => useOnboardingStore());
    act(() => result.current.prevStep());
    expect(result.current.currentStep).toBe(0);
  });

  it('updates org name via updateData', () => {
    const { result } = renderHook(() => useOnboardingStore());
    act(() => result.current.updateData({ orgName: 'Acme Corp' }));
    expect(result.current.data.orgName).toBe('Acme Corp');
  });

  it('marks completed and persists to localStorage', () => {
    const { result } = renderHook(() => useOnboardingStore());
    act(() => result.current.complete());
    expect(result.current.completed).toBe(true);
    expect(localStorage.getItem('aiauditor-onboarding-done')).toBe('true');
  });

  it('skips and marks completed', () => {
    const { result } = renderHook(() => useOnboardingStore());
    act(() => result.current.skip());
    expect(result.current.completed).toBe(true);
    expect(result.current.skipped).toBe(true);
  });

  it('hydrateOnboarding reads localStorage', () => {
    localStorage.setItem('aiauditor-onboarding-done', 'true');
    hydrateOnboarding();
    expect(useOnboardingStore.getState().completed).toBe(true);
  });

  it('reset clears completed state and localStorage', () => {
    localStorage.setItem('aiauditor-onboarding-done', 'true');
    const { result } = renderHook(() => useOnboardingStore());
    act(() => result.current.reset());
    expect(result.current.completed).toBe(false);
    expect(localStorage.getItem('aiauditor-onboarding-done')).toBeNull();
  });

  it('invites list grows and shrinks', () => {
    const { result } = renderHook(() => useOnboardingStore());
    act(() => result.current.updateData({ invites: ['a@b.com'] }));
    expect(result.current.data.invites).toHaveLength(1);
    act(() => result.current.updateData({ invites: ['a@b.com', 'c@d.com'] }));
    expect(result.current.data.invites).toHaveLength(2);
  });

  it('frameworks selection toggles correctly', () => {
    const { result } = renderHook(() => useOnboardingStore());
    act(() => result.current.updateData({ frameworks: ['nis2', 'iso27001'] }));
    expect(result.current.data.frameworks).toContain('nis2');
    expect(result.current.data.frameworks).toContain('iso27001');
    // Deselect
    act(() => result.current.updateData({ frameworks: ['iso27001'] }));
    expect(result.current.data.frameworks).not.toContain('nis2');
  });
});
