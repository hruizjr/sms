import { describe, expect, it } from 'vitest';
import indexHtml from '../index.html?raw';
import {
  addContentSecurityPolicy,
  CONTENT_SECURITY_POLICY,
  contentSecurityPolicy,
} from './content-security-policy';

function directives(policy: string): Map<string, string> {
  return new Map(
    policy.split(';').map((part) => {
      const [name = '', ...values] = part.trim().split(/\s+/);
      return [name, values.join(' ')];
    }),
  );
}

describe('content security policy', () => {
  const policy = directives(CONTENT_SECURITY_POLICY);

  it('blocks every browser network call from the page', () => {
    expect(policy.get('connect-src')).toBe("'none'");
  });

  it('denies by default and allows only same-origin scripts and styles, nothing inline', () => {
    expect(policy.get('default-src')).toBe("'none'");
    expect(policy.get('script-src')).toBe("'self'");
    expect(policy.get('style-src')).toBe("'self'");
    expect(CONTENT_SECURITY_POLICY).not.toContain('unsafe-inline');
    expect(CONTENT_SECURITY_POLICY).not.toContain('unsafe-eval');
  });

  it('blocks form posts, <base> changes and plugins', () => {
    expect(policy.get('form-action')).toBe("'none'");
    expect(policy.get('base-uri')).toBe("'none'");
    expect(policy.get('object-src')).toBe("'none'");
  });

  it('is placed directly after <meta charset>, before any script or stylesheet', () => {
    const html = addContentSecurityPolicy(indexHtml);
    const csp = html.indexOf('<meta http-equiv="Content-Security-Policy"');
    expect(csp).toBeGreaterThan(html.indexOf('<meta charset="UTF-8" />'));
    expect(csp).toBeLessThan(html.indexOf('<script'));
    expect(html).toContain(`content="${CONTENT_SECURITY_POLICY}"`);
  });

  it('refuses an index.html without <meta charset>', () => {
    expect(() => addContentSecurityPolicy('<html><head></head></html>')).toThrow(/meta charset/);
  });

  it('applies to the production build only', () => {
    expect(contentSecurityPolicy().apply).toBe('build');
  });
});
