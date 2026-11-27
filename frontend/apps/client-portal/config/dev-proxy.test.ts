import { describe, expect, it } from 'vitest';
import viteConfig from '../vite.config.ts';
import { DEV_API_PROXY, DEV_API_TARGET } from './dev-proxy';

describe('development proxy', () => {
  it('forwards only /healthz, /readyz and the portal API to the local API', () => {
    expect(Object.keys(DEV_API_PROXY).sort()).toEqual(['/healthz', '/portal', '/readyz']);
    for (const options of Object.values(DEV_API_PROXY)) {
      expect(options.target).toBe('http://127.0.0.1:8080');
    }
    expect(DEV_API_TARGET).toBe('http://127.0.0.1:8080');
  });

  it('is used by the development server only, not by preview', () => {
    expect(viteConfig.server?.proxy).toBe(DEV_API_PROXY);
    expect(viteConfig.preview?.proxy).toEqual({});
  });

  it('does not proxy any public API path', () => {
    const paths = Object.keys(viteConfig.server?.proxy ?? {});
    expect(paths.some((path) => path.startsWith('/v1'))).toBe(false);
  });
});
