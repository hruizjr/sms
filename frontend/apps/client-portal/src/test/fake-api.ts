import type { Message, MessageDetail, Session, Transport } from '../lib/api/client';

/*
 * A fake portal API for tests: a transport that answers from a handler function and records every
 * request. Tests never touch the real network (the network guard in setup.ts blocks it).
 */

export type FakeRequest = {
  method: string;
  path: string;
  headers: Record<string, string>;
  body: unknown;
};

export type Handler = (request: FakeRequest) => Response | Promise<Response>;

export type FakeTransport = Transport & { calls: FakeRequest[] };

export function fakeTransport(handler: Handler): FakeTransport {
  const calls: FakeRequest[] = [];
  const transport = async (url: string, init: RequestInit): Promise<Response> => {
    const request: FakeRequest = {
      method: init.method ?? 'GET',
      path: url,
      headers: { ...(init.headers as Record<string, string> | undefined) },
      body: typeof init.body === 'string' ? JSON.parse(init.body) : undefined,
    };
    calls.push(request);
    return handler(request);
  };
  return Object.assign(transport, { calls });
}

export function json(
  status: number,
  body: unknown,
  headers: Record<string, string> = {},
): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  });
}

export function problem(
  status: number,
  type: string,
  extra: Record<string, unknown> = {},
): Response {
  return new Response(
    JSON.stringify({
      type: `https://api.example.invalid/problems/${type}`,
      title: type,
      status,
      detail: `${type} detail`,
      ...extra,
    }),
    { status, headers: { 'Content-Type': 'application/problem+json' } },
  );
}

export const CSRF = 'csrf-token-for-tests';

export function sessionFor(role: 'owner' | 'sender' | 'viewer' = 'owner'): Session {
  return {
    user: { email: `${role}@example.invalid`, role },
    organization: { name: 'Example Co' },
    permissions: role === 'viewer' ? ['messages:read'] : ['messages:read', 'messages:send'],
    csrf_token: CSRF,
    expires_at: '2026-11-27T20:00:00Z',
    idle_expires_at: '2026-11-27T08:30:00Z',
  };
}

export function messageFixture(overrides: Partial<Message> = {}): Message {
  return {
    id: 'msg_06H1WH346HSF13TPPJR44P43PW',
    to: '+639000000001',
    sender_id: 'EXAMPLECO',
    status: 'delivered',
    status_category: 'final',
    status_reason: null,
    encoding: 'gsm7',
    segment_count: 1,
    accepted_at: '2026-11-27T08:00:00Z',
    status_updated_at: '2026-11-27T08:00:05Z',
    ...overrides,
  };
}

export function detailFixture(overrides: Partial<MessageDetail> = {}): MessageDetail {
  return {
    ...messageFixture(),
    status_history: [
      {
        status: 'accepted',
        status_category: 'in_progress',
        status_reason: null,
        occurred_at: '2026-11-27T08:00:00Z',
      },
      {
        status: 'provider_accepted',
        status_category: 'in_progress',
        status_reason: null,
        occurred_at: '2026-11-27T08:00:01Z',
      },
      {
        status: 'delivered',
        status_category: 'final',
        status_reason: null,
        occurred_at: '2026-11-27T08:00:05Z',
      },
    ],
    ...overrides,
  };
}

/** A handler for a signed-in user that answers the given routes ("METHOD /path") and 404 otherwise. */
export function signedIn(
  role: 'owner' | 'sender' | 'viewer',
  routes: Record<string, Handler> = {},
): Handler {
  return (request) => {
    const key = `${request.method} ${request.path}`;
    const route = routes[key];
    if (route !== undefined) return route(request);
    if (key === 'GET /portal/v1/session') return json(200, sessionFor(role));
    return problem(404, 'not-found');
  };
}
