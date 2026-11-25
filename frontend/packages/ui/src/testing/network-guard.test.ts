/* eslint-disable no-restricted-globals, no-restricted-properties -- this test calls the blocked APIs to prove the guard stops them */
import { describe, expect, it } from 'vitest';
import { NetworkAccessBlockedError } from './network-guard';

describe('network guard (installed by the test setup)', () => {
  it('blocks fetch', () => {
    expect(() => fetch('https://example.invalid/')).toThrow(NetworkAccessBlockedError);
  });

  it.each(['XMLHttpRequest', 'WebSocket', 'EventSource'] as const)('blocks %s', (api) => {
    const Constructor = globalThis[api] as unknown as new (url?: string) => unknown;
    expect(() => new Constructor('https://example.invalid/')).toThrow(NetworkAccessBlockedError);
  });

  it('blocks navigator.sendBeacon', () => {
    expect(() => navigator.sendBeacon('https://example.invalid/')).toThrow(
      NetworkAccessBlockedError,
    );
  });
});
