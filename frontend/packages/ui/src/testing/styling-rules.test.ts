import { describe, expect, it } from 'vitest';
import { findStylingViolations } from './styling-rules';

const uiSources = import.meta.glob<string>(
  ['../**/*.{ts,tsx}', '!../**/*.test.{ts,tsx}', '!../testing/**'],
  {
    query: '?raw',
    import: 'default',
    eager: true,
  },
);

describe('styling rules', () => {
  it('the UI package uses semantic token classes only', () => {
    expect(Object.keys(uiSources).length).toBeGreaterThan(0);
    expect(findStylingViolations(uiSources)).toEqual([]);
  });

  it('detects arbitrary values and default palette colours', () => {
    const violations = findStylingViolations({
      'example.tsx': '<p className="bg-[#0f172a] text-blue-500 border-white p-4 text-sm" />',
    });
    expect(violations.map((v) => v.match)).toEqual([
      'bg-[#0f172a]',
      'text-blue-500',
      'border-white',
    ]);
  });

  it('accepts semantic token classes', () => {
    expect(
      findStylingViolations({
        'ok.tsx': '<p className="bg-surface text-text-muted border-border" />',
      }),
    ).toEqual([]);
  });
});
