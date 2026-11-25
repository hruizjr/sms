// Blocks browser network APIs in tests. No frontend code may call a server until a separately
// approved, session-authenticated portal API contract exists (owner decision, Phase 3 Group 5).
// Installed by every frontend test setup file, so a test fails if rendered code tries to connect.

export const BLOCKED_NETWORK_APIS = [
  'fetch',
  'XMLHttpRequest',
  'WebSocket',
  'EventSource',
  'navigator.sendBeacon',
] as const;

export class NetworkAccessBlockedError extends Error {
  constructor(api: string) {
    super(
      `${api} is blocked: the frontend has no approved API contract yet, so no network calls are allowed.`,
    );
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
