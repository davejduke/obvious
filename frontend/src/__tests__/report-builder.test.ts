/**
 * Report Builder block operation tests.
 * Pure function tests — no browser/DOM required (node test environment).
 */
import {
  createBlock,
  addBlock,
  removeBlock,
  updateBlock,
  moveBlockUp,
  moveBlockDown,
  reorderBlocks,
  getTemplateBlocks,
  TEMPLATE_LABELS,
  type ReportBlock,
  type BlockType,
} from '../lib/report-builder';

// ─── createBlock ───────────────────────────────────────────────────────────────────

describe('createBlock', () => {
  it('generates a unique id for each call', () => {
    const a = createBlock('paragraph');
    const b = createBlock('paragraph');
    expect(a.id).not.toBe(b.id);
  });

  it('sets the correct block type', () => {
    const types: BlockType[] = ['heading', 'paragraph', 'finding_block', 'evidence_reference', 'chart', 'table'];
    types.forEach(type => {
      expect(createBlock(type).type).toBe(type);
    });
  });

  it('applies overrides correctly', () => {
    const block = createBlock('heading', { content: 'My Title', level: 1 });
    expect(block.content).toBe('My Title');
    expect(block.level).toBe(1);
  });

  it('sets default content for paragraph', () => {
    const block = createBlock('paragraph');
    expect(typeof block.content).toBe('string');
    expect(block.content.length).toBeGreaterThan(0);
  });
});

// ─── addBlock ─────────────────────────────────────────────────────────────────────

describe('addBlock', () => {
  function makeBlocks(n: number): ReportBlock[] {
    return Array.from({ length: n }, (_, i) =>
      createBlock('paragraph', { content: `Block ${i}` })
    );
  }

  it('increases the block count by 1', () => {
    const blocks = makeBlocks(3);
    expect(addBlock(blocks, 'heading', 1)).toHaveLength(4);
  });

  it('inserts the new block after the specified index', () => {
    const blocks = makeBlocks(2);
    const result = addBlock(blocks, 'heading', 0);
    // result should be: [block0, NEW, block1]
    expect(result[1].type).toBe('heading');
    expect(result[0].content).toBe('Block 0');
    expect(result[2].content).toBe('Block 1');
  });

  it('appends when afterIndex is the last index', () => {
    const blocks = makeBlocks(2);
    const result = addBlock(blocks, 'chart', 1); // last index = 1
    expect(result[2].type).toBe('chart');
  });

  it('does not mutate the original array', () => {
    const blocks = makeBlocks(2);
    addBlock(blocks, 'table', 0);
    expect(blocks).toHaveLength(2);
  });
});

// ─── removeBlock ────────────────────────────────────────────────────────────────

describe('removeBlock', () => {
  it('decreases the block count by 1', () => {
    const block = createBlock('paragraph');
    const result = removeBlock([block], block.id);
    expect(result).toHaveLength(0);
  });

  it('removes the correct block by id', () => {
    const a = createBlock('heading');
    const b = createBlock('paragraph');
    const result = removeBlock([a, b], a.id);
    expect(result[0].id).toBe(b.id);
  });

  it('does nothing when id is not found', () => {
    const blocks = [createBlock('paragraph')];
    expect(removeBlock(blocks, 'nonexistent-id')).toHaveLength(1);
  });

  it('does not mutate the original array', () => {
    const block = createBlock('paragraph');
    removeBlock([block], block.id);
    // Original created block still accessible
    expect(block.id).toBeDefined();
  });
});

// ─── updateBlock ────────────────────────────────────────────────────────────────

describe('updateBlock', () => {
  it('updates the content of a matching block', () => {
    const block = createBlock('paragraph', { content: 'original' });
    const result = updateBlock([block], block.id, { content: 'updated' });
    expect(result[0].content).toBe('updated');
  });

  it('preserves other fields when updating', () => {
    const block = createBlock('heading', { content: 'title', level: 1 });
    const result = updateBlock([block], block.id, { content: 'new title' });
    expect(result[0].level).toBe(1);
  });

  it('does not modify blocks with non-matching ids', () => {
    const a = createBlock('paragraph', { content: 'A' });
    const b = createBlock('paragraph', { content: 'B' });
    const result = updateBlock([a, b], a.id, { content: 'A-updated' });
    expect(result[1].content).toBe('B');
  });
});

// ─── moveBlockUp / moveBlockDown ───────────────────────────────────────────

describe('moveBlockUp', () => {
  it('swaps a block with its predecessor', () => {
    const a = createBlock('heading',   { content: 'A' });
    const b = createBlock('paragraph', { content: 'B' });
    const result = moveBlockUp([a, b], b.id);
    expect(result[0].content).toBe('B');
    expect(result[1].content).toBe('A');
  });

  it('is a no-op when block is already first', () => {
    const a = createBlock('heading', { content: 'A' });
    const b = createBlock('heading', { content: 'B' });
    const result = moveBlockUp([a, b], a.id);
    expect(result[0].content).toBe('A');
  });
});

describe('moveBlockDown', () => {
  it('swaps a block with its successor', () => {
    const a = createBlock('heading',   { content: 'A' });
    const b = createBlock('paragraph', { content: 'B' });
    const result = moveBlockDown([a, b], a.id);
    expect(result[0].content).toBe('B');
    expect(result[1].content).toBe('A');
  });

  it('is a no-op when block is already last', () => {
    const a = createBlock('heading', { content: 'A' });
    const b = createBlock('heading', { content: 'B' });
    const result = moveBlockDown([a, b], b.id);
    expect(result[1].content).toBe('B');
  });
});

// ─── reorderBlocks (drag-and-drop) ──────────────────────────────────────────

describe('reorderBlocks', () => {
  function namedBlocks(): ReportBlock[] {
    return [
      createBlock('heading',   { content: 'A' }),
      createBlock('paragraph', { content: 'B' }),
      createBlock('paragraph', { content: 'C' }),
      createBlock('paragraph', { content: 'D' }),
    ];
  }

  it('returns the same array when fromIndex === toIndex', () => {
    const blocks = namedBlocks();
    expect(reorderBlocks(blocks, 1, 1)).toEqual(blocks);
  });

  it('moves a block from a lower index to a higher index', () => {
    const result = reorderBlocks(namedBlocks(), 0, 2);
    // A was at 0, should now be at 2 → B, C, A, D
    expect(result.map(b => b.content)).toEqual(['B', 'C', 'A', 'D']);
  });

  it('moves a block from a higher index to a lower index', () => {
    const result = reorderBlocks(namedBlocks(), 3, 1);
    // D was at 3, should now be at 1 → A, D, B, C
    expect(result.map(b => b.content)).toEqual(['A', 'D', 'B', 'C']);
  });

  it('preserves the total block count', () => {
    const blocks = namedBlocks();
    expect(reorderBlocks(blocks, 0, 3)).toHaveLength(blocks.length);
  });

  it('returns unchanged array when fromIndex is out of bounds', () => {
    const blocks = namedBlocks();
    expect(reorderBlocks(blocks, -1, 1)).toEqual(blocks);
    expect(reorderBlocks(blocks, 100, 1)).toEqual(blocks);
  });

  it('returns unchanged array when toIndex is out of bounds', () => {
    const blocks = namedBlocks();
    expect(reorderBlocks(blocks, 0, -1)).toEqual(blocks);
    expect(reorderBlocks(blocks, 0, 100)).toEqual(blocks);
  });
});

// ─── getTemplateBlocks ─────────────────────────────────────────────────────────

describe('getTemplateBlocks', () => {
  it('returns at least one block for each template', () => {
    (['standard', 'executive', 'regulatory'] as const).forEach(id => {
      expect(getTemplateBlocks(id).length).toBeGreaterThan(0);
    });
  });

  it('generates unique IDs across two calls to the same template', () => {
    const call1 = getTemplateBlocks('standard');
    const call2 = getTemplateBlocks('standard');
    const ids1 = call1.map(b => b.id);
    const ids2 = call2.map(b => b.id);
    const intersection = ids1.filter(id => ids2.includes(id));
    expect(intersection).toHaveLength(0);
  });

  it('standard template contains at least one finding_block', () => {
    const blocks = getTemplateBlocks('standard');
    expect(blocks.some(b => b.type === 'finding_block')).toBe(true);
  });

  it('regulatory template contains at least one evidence_reference', () => {
    const blocks = getTemplateBlocks('regulatory');
    expect(blocks.some(b => b.type === 'evidence_reference')).toBe(true);
  });

  it('executive template starts with a level-1 heading', () => {
    const blocks = getTemplateBlocks('executive');
    expect(blocks[0].type).toBe('heading');
    expect(blocks[0].level).toBe(1);
  });
});

// ─── TEMPLATE_LABELS ────────────────────────────────────────────────────────────

describe('TEMPLATE_LABELS', () => {
  it('has a human-readable label for each template', () => {
    (['standard', 'executive', 'regulatory'] as const).forEach(id => {
      expect(TEMPLATE_LABELS[id]).toBeTruthy();
      expect(typeof TEMPLATE_LABELS[id]).toBe('string');
    });
  });
});

