/**
 * Keyboard shortcut registry tests
 */
import { globalShortcuts, registerShortcut } from '../hooks/use-keyboard-shortcuts';

beforeEach(() => {
  // Clear any globally-registered shortcuts between tests
  globalShortcuts.length = 0;
});

describe('Keyboard shortcut registry', () => {
  it('registers a shortcut and adds it to the global list', () => {
    const handler = jest.fn();
    registerShortcut({ key: 'k', meta: true, description: 'Open search', group: 'Navigation', handler });
    expect(globalShortcuts).toHaveLength(1);
    expect(globalShortcuts[0].key).toBe('k');
    expect(globalShortcuts[0].group).toBe('Navigation');
  });

  it('returns an unregister function that removes the shortcut', () => {
    const handler = jest.fn();
    const unregister = registerShortcut({ key: '?', description: 'Help', group: 'Help', handler });
    expect(globalShortcuts).toHaveLength(1);
    unregister();
    expect(globalShortcuts).toHaveLength(0);
  });

  it('registers multiple shortcuts from different groups', () => {
    registerShortcut({ key: 'k', meta: true, description: 'Search', group: 'Navigation', handler: jest.fn() });
    registerShortcut({ key: '?', description: 'Shortcuts', group: 'Help', handler: jest.fn() });
    registerShortcut({ key: 'Escape', description: 'Close', group: 'Navigation', handler: jest.fn() });
    expect(globalShortcuts).toHaveLength(3);
    const groups = new Set(globalShortcuts.map(s => s.group));
    expect(groups.has('Navigation')).toBe(true);
    expect(groups.has('Help')).toBe(true);
  });

  it('unregistering one shortcut does not affect others', () => {
    const un1 = registerShortcut({ key: 'k', meta: true, description: 'Search', group: 'Navigation', handler: jest.fn() });
    registerShortcut({ key: '?', description: 'Help', group: 'Help', handler: jest.fn() });
    un1();
    expect(globalShortcuts).toHaveLength(1);
    expect(globalShortcuts[0].key).toBe('?');
  });

  it('shortcut definitions contain expected fields', () => {
    const handler = jest.fn();
    registerShortcut({ key: 'k', meta: true, shift: false, description: 'Test shortcut', group: 'Test', handler });
    const s = globalShortcuts[0];
    expect(s).toHaveProperty('key');
    expect(s).toHaveProperty('description');
    expect(s).toHaveProperty('group');
    expect(s).toHaveProperty('handler');
  });
});
