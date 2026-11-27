import { describe, expect, it } from 'vitest';
import themeCss from './theme.css?raw';
import { contrastRatio } from './testing/contrast';

function colourTokens(css: string): Map<string, string> {
  const tokens = new Map<string, string>();
  for (const match of css.matchAll(/--color-([a-z-]+):\s*([^;]+);/g)) {
    const [, name, value] = match;
    if (name !== undefined && value !== undefined) {
      tokens.set(name, value.trim());
    }
  }
  return tokens;
}

const tokens = colourTokens(themeCss);

function token(name: string): string {
  const value = tokens.get(name);
  if (value === undefined) {
    throw new Error(`colour token --color-${name} is not defined in theme.css`);
  }
  return value;
}

// Text and the background it is drawn on: WCAG 2.2 SC 1.4.3, 4.5:1 for normal-size text.
const textPairs: [foreground: string, background: string][] = [
  ['text', 'surface'],
  ['text', 'surface-muted'],
  ['text-muted', 'surface'],
  ['text-muted', 'surface-muted'],
  ['primary-foreground', 'primary'],
  ['primary-foreground', 'primary-hover'],
  ['danger-foreground', 'danger'],
  ['danger-foreground', 'danger-hover'],
  ['success', 'surface'],
  ['success', 'surface-muted'],
  ['warning', 'surface'],
  ['warning', 'surface-muted'],
  ['info', 'surface'],
  ['info', 'surface-muted'],
  ['danger', 'surface'],
  ['danger', 'surface-muted'],
  ['text', 'surface-subtle'],
  ['text-muted', 'surface-subtle'],
  ['primary', 'primary-subtle'],
  ['text', 'primary-subtle'],
  ['success', 'success-subtle'],
  ['warning', 'warning-subtle'],
  ['info', 'info-subtle'],
  ['danger', 'danger-subtle'],
  ['text', 'danger-subtle'],
  ['text', 'warning-subtle'],
  ['text', 'info-subtle'],
  ['text', 'success-subtle'],
  ['code-foreground', 'code'],
];

// Focus indicators and component boundaries against the backgrounds they sit on:
// WCAG 2.2 SC 1.4.11 (non-text contrast) and SC 2.4.7 (focus visible), 3:1.
const nonTextPairs: [foreground: string, background: string][] = [
  ['focus', 'surface'],
  ['focus', 'surface-muted'],
  ['border-strong', 'surface'],
  ['border-strong', 'surface-muted'],
  ['primary', 'surface'],
  ['primary', 'surface-muted'],
  ['danger', 'surface'],
  ['danger', 'surface-muted'],
  ['focus', 'surface-subtle'],
  ['border-strong', 'surface-subtle'],
  ['focus', 'primary-subtle'],
];

describe('design tokens', () => {
  it("switch off Tailwind's default colour palette", () => {
    expect(themeCss).toMatch(/--color-\*:\s*initial;/);
  });

  it('define every colour as #rrggbb', () => {
    expect(tokens.size).toBeGreaterThan(0);
    for (const [name, value] of tokens) {
      expect(value, `--color-${name}`).toMatch(/^#[0-9a-f]{6}$/i);
    }
  });

  it.each(textPairs)('text %s on %s meets 4.5:1', (foreground, background) => {
    expect(contrastRatio(token(foreground), token(background))).toBeGreaterThanOrEqual(4.5);
  });

  it.each(nonTextPairs)('non-text %s on %s meets 3:1', (foreground, background) => {
    expect(contrastRatio(token(foreground), token(background))).toBeGreaterThanOrEqual(3);
  });

  it('draws the keyboard focus indicator with the focus token, offset onto the background', () => {
    expect(themeCss).toMatch(
      /:focus-visible\s*\{\s*outline:\s*2px solid var\(--color-focus\);\s*outline-offset:\s*2px;/,
    );
  });
});

describe('contrastRatio', () => {
  it('matches the WCAG reference values', () => {
    expect(contrastRatio('#000000', '#ffffff')).toBeCloseTo(21, 5);
    expect(contrastRatio('#ffffff', '#ffffff')).toBeCloseTo(1, 5);
    expect(contrastRatio('#767676', '#ffffff')).toBeCloseTo(4.54, 2);
  });
});
