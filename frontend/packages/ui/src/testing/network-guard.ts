// Blocks browser network APIs in tests (owner decision, Phase 3 Group 5). Installed by every frontend
// test setup file, so a test fails if rendered code tries to connect. Code that calls the portal API
// (Phase 4 Group 3) is tested with an injected fake transport, never with the real network.

export const BLOCKED_NETWORK_APIS = [
  'fetch',
  'XMLHttpRequest',
  'WebSocket',
  'EventSource',
  'navigator.sendBeacon',
] as const;

export class NetworkAccessBlockedError extends Error {
  constructor(api: string) {
    super(`${api} is blocked in tests: inject a fake transport instead of calling the network.`);
    this.name = 'NetworkAccessBlockedError';
  }
}

// A plain function also throws when called with `new`, which is how these APIs are used.
function blockedConstructor(api: string) {
  return function Blocked(): never {
    throw new NetworkAccessBlockedError(api);
  };
}

export function installNetworkGuard(): void {
  Object.defineProperty(globalThis, 'fetch', {
    configurable: true,
    writable: true,
    value: () => {
      throw new NetworkAccessBlockedError('fetch');
    },
  });
  for (const api of ['XMLHttpRequest', 'WebSocket', 'EventSource'] as const) {
    Object.defineProperty(globalThis, api, {
      configurable: true,
      writable: true,
      value: blockedConstructor(api),
    });
  }
  Object.defineProperty(globalThis.navigator, 'sendBeacon', {
    configurable: true,
    writable: true,
    value: () => {
      throw new NetworkAccessBlockedError('navigator.sendBeacon');
    },
  });
}
