// Checks the production build after `vite build` (run by `pnpm build`):
// - the Content Security Policy <meta> element comes before every script and stylesheet and blocks
//   network calls (connect-src 'none');
// - index.html has no inline <script> or <style> (the policy would block them);
// - no source maps are published.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const dist = join(import.meta.dirname, '..', 'dist');
const html = readFileSync(join(dist, 'index.html'), 'utf8');
const failures = [];

const cspTag = /<meta http-equiv="Content-Security-Policy" content="([^"]*)"/.exec(html);
const csp = (cspTag?.[1] ?? '').replaceAll('&#39;', "'");
if (cspTag === null) {
  failures.push('index.html has no Content Security Policy <meta> element');
} else {
  const firstResource = html.search(/<(script|link|style)\b/);
  if (firstResource !== -1 && firstResource < cspTag.index) {
    failures.push('the Content Security Policy comes after a script, stylesheet or style element');
  }
  if (!csp.split(';').some((directive) => directive.trim() === "connect-src 'none'")) {
    failures.push("the Content Security Policy does not contain connect-src 'none'");
  }
}
for (const match of html.matchAll(/<script\b([^>]*)>/g)) {
  if (!/\bsrc=/.test(match[1] ?? '')) {
    failures.push(`inline script in index.html: ${match[0]}`);
  }
}
if (/<style\b/.test(html)) {
  failures.push('inline <style> in index.html');
}
const maps = readdirSync(dist, { recursive: true }).filter((file) => String(file).endsWith('.map'));
if (maps.length > 0) {
  failures.push(`source maps in the build: ${maps.join(', ')}`);
}

if (failures.length > 0) {
  console.error(`Build verification failed:\n- ${failures.join('\n- ')}`);
  process.exitCode = 1;
} else {
  console.log(
    'Build verification passed: CSP before all resources, no inline scripts or styles, no source maps.',
  );
}
