import type { Plugin } from 'vite';

/**
 * Content Security Policy added to the production build's index.html (owner requirement, Phase 3
 * Group 5; widened in Phase 4 Group 3, owner decision G3-6, for the approved session-authenticated
 * portal API).
 *
 * - connect-src 'self' lets the page call only its own origin, where the portal API is served
 *   (/portal/v1); every other host is refused for fetch, XMLHttpRequest, WebSocket, EventSource and
 *   navigator.sendBeacon.
 * - Only same-origin scripts, styles, images and fonts load; no inline scripts or styles.
 * - form-action 'none' and base-uri 'none' stop form posts and <base> redirection.
 *
 * Delivered as a <meta> element because hosting is not decided. frame-ancestors and reporting
 * only work in an HTTP header; the hosting decision should send this policy (plus frame-ancestors)
 * as a header. The Vite development server is not covered: its hot reload needs a WebSocket and
 * inline scripts. ESLint rules and the test network guard cover source code there.
 */
export const CONTENT_SECURITY_POLICY = [
  "default-src 'none'",
  "script-src 'self'",
  "style-src 'self'",
  "img-src 'self'",
  "font-src 'self'",
  "connect-src 'self'",
  "base-uri 'none'",
  "form-action 'none'",
  "object-src 'none'",
].join('; ');

const CHARSET_META = '<meta charset="UTF-8" />';

/** Inserts the policy directly after <meta charset>, before any script or stylesheet. */
export function addContentSecurityPolicy(html: string): string {
  if (!html.includes(CHARSET_META)) {
    throw new Error(`index.html must contain ${CHARSET_META} to place the Content Security Policy`);
  }
  return html.replace(
    CHARSET_META,
    `${CHARSET_META}\n    <meta http-equiv="Content-Security-Policy" content="${CONTENT_SECURITY_POLICY}" />`,
  );
}

export function contentSecurityPolicy(): Plugin {
  return {
    name: 'client-portal:content-security-policy',
    apply: 'build',
    transformIndexHtml: { order: 'post', handler: addContentSecurityPolicy },
  };
}
