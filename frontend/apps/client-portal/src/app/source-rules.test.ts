import { findStylingViolations } from '@sms/ui/testing';
import { describe, expect, it } from 'vitest';

const appSources = import.meta.glob<string>(['../**/*.{ts,tsx}', '!../**/*.test.{ts,tsx}'], {
  query: '?raw',
  import: 'default',
  eager: true,
});

describe('client portal source rules', () => {
  it('uses semantic token classes only', () => {
    expect(Object.keys(appSources).length).toBeGreaterThan(0);
    expect(findStylingViolations(appSources)).toEqual([]);
  });
});
