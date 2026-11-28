import type {
  ApiKey,
  AuditEvent,
  Message,
  MessageDetail,
  Session,
  Transport,
} from '../lib/api/client';

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
  headers: Record<string, string> = {},
): Response {
  return new Response(
    JSON.stringify({
      type: `https://api.example.invalid/problems/${type}`,
      title: type,
      status,
      detail: `${type} detail`,
      ...extra,
    }),
    { status, headers: { 'Content-Type': 'application/problem+json', ...headers } },
  );
}

export const CSRF = 'csrf-token-for-tests';

export function sessionFor(role: 'owner' | 'sender' | 'viewer' = 'owner'): Session {
  return {
    user: { email: `${role}@example.invalid`, role },
    organization: { name: 'Example Co' },
    permissions:
      role === 'viewer'
        ? ['messages:read']
        : role === 'sender'
          ? ['messages:read', 'messages:send']
          : ['messages:read', 'messages:send', 'api_keys:manage', 'audit_log:read'],
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

export function apiKeyFixture(overrides: Partial<ApiKey> = {}): ApiKey {
  const id = overrides.id ?? '06h1wh346hsf13tppjr44p43pw';
  return {
    id,
    name: 'Order notifications',
    prefix: `smsk_06h1wh346hsf13tppjr44p43aa_${id}`,
    scopes: ['messages:send', 'messages:read'],
    status: 'active',
    created_at: '2026-11-27T08:00:00Z',
    revoked_at: null,
    last_used_at: null,
    ...overrides,
  };
}

export function auditEventFixture(overrides: Partial<AuditEvent> = {}): AuditEvent {
  return {
    id: '0193f0a1-7c3e-7d2a-9b1e-2f6c8d4e5a01',
    occurred_at: '2026-11-28T01:30:00Z',
    action: 'portal.sign_in',
    outcome: 'success',
    actor: {
      type: 'portal_user',
      id: '0193f0a1-0000-7000-8000-000000000001',
      email: 'owner@example.invalid',
    },
    target: { type: 'portal_session', id: '0193f0a1-0000-7000-8000-0000000000aa', email: null },
    request_id: '9b2f4c1e7d3a5b60',
    source_ip: '203.0.113.10',
    user_agent: 'Mozilla/5.0 (Example)',
    details: {},
    ...overrides,
  };
}
