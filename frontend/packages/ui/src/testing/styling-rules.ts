// Source checks for the design-token rules (AGENTS.md §8.3): components use semantic token classes,
// never Tailwind arbitrary values (such as bg-[#123456]) or Tailwind's default palette names (such
// as bg-blue-500; the palette is switched off in theme.css, so those classes would silently do
// nothing). Inline style attributes are rejected by ESLint.

const ARBITRARY_VALUE = /\b[a-z][\w-]*-\[[^\]\s]+\]/g;
const PALETTE_COLOUR =
  /\b(?:bg|text|border(?:-[trblxy])?|outline|ring|ring-offset|divide|fill|stroke|from|via|to|decoration|accent|caret|placeholder|shadow)-(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose|black|white)(?:-\d{2,3})?\b/g;

export type StylingViolation = { file: string; match: string; rule: string };

export function findStylingViolations(sources: Record<string, string>): StylingViolation[] {
  const violations: StylingViolation[] = [];
  for (const [file, text] of Object.entries(sources)) {
    for (const match of text.matchAll(ARBITRARY_VALUE)) {
      violations.push({ file, match: match[0], rule: 'Tailwind arbitrary value' });
    }
    for (const match of text.matchAll(PALETTE_COLOUR)) {
      violations.push({ file, match: match[0], rule: 'Tailwind default palette colour' });
    }
  }
  return violations;
}
