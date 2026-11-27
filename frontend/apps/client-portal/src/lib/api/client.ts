import type { components } from './schema.gen';

/*
 * The single typed client of the session-authenticated portal API (AGENTS.md §8.5; owner decision
 * G3-6). Types come from schema.gen.ts, generated from contracts/openapi/portal-api-v1.yaml.
 *
 * Security:
 * - The session lives in an HttpOnly cookie set by the server; this code never sees or stores it,
 *   and nothing is written to browser storage.
 * - Every state-changing request carries the session's CSRF token in X-CSRF-Token.
 * - The tenant is never sent: the server derives it from the session.
 * - Requests go to the page's own origin only (relative URLs; the CSP allows connect-src 'self').
 */

type Schemas = components['schemas'];
export type Session = Schemas['Session'];
export type Message = Schemas['Message'];
export type MessageDetail = Schemas['MessageDetail'];
export type MessageList = Schemas['MessageList'];
export type MessageEstimate = Schemas['MessageEstimate'];
export type MessageCreateRequest = Schemas['MessageCreateRequest'];
export type MessageStatus = Schemas['MessageStatus'];
export type StatusCategory = Schemas['StatusCategory'];
export type StatusChange = Schemas['StatusChange'];
export type Permission = Schemas['Permission'];
export type Role = Schemas['Role'];
export type Problem = Schemas['Problem'];
export type FieldError = Schemas['FieldError'];

/** Sends one HTTP request; the browser's fetch in production, a fake in tests. */
export type Transport = (url: string, init: RequestInit) => Promise<Response>;

/** The browser transport: same-origin fetch that never caches. */
export const browserTransport: Transport = (url, init) =>
  fetch(url, { ...init, credentials: 'same-origin', cache: 'no-store' });

/** The server answered with an error (RFC 9457 problem details when available). */
export class ApiError extends Error {
  readonly status: number;
  readonly problem: Problem | undefined;

  constructor(status: number, problem: Problem | undefined) {
    super(problem?.detail ?? problem?.title ?? `Request failed with status ${String(status)}`);
    this.name = 'ApiError';
    this.status = status;
    this.problem = problem;
  }

  /** The problem type's name, for example "invalid-credentials" ("" if unknown). */
  get problemType(): string {
    const type = this.problem?.type ?? '';
    return type.slice(type.lastIndexOf('/') + 1);
  }
}

/** The request did not get an answer (offline, server down, connection reset). */
export class NetworkError extends Error {
  constructor() {
    super('The server could not be reached.');
    this.name = 'NetworkError';
  }
}

export type ApiClient = {
  getSession(): Promise<Session>;
  signIn(email: string, password: string): Promise<Session>;
  signOut(): Promise<void>;
  listSenderIds(): Promise<string[]>;
  estimateMessage(body: string): Promise<MessageEstimate>;
  listMessages(params?: { cursor?: string; limit?: number }): Promise<MessageList>;
  getMessage(id: string): Promise<MessageDetail>;
  createMessage(
    request: MessageCreateRequest,
    idempotencyKey: string,
  ): Promise<{ message: Message; replayed: boolean }>;
};

export type ApiClientOptions = {
  transport: Transport;
  /** The current session's CSRF token, if signed in. */
  csrfToken: () => string | undefined;
  /** Called when a request is refused because there is no valid session (401). */
  onUnauthenticated?: () => void;
};

const BASE = '/portal/v1';

export function createApiClient({
  transport,
  csrfToken,
  onUnauthenticated,
}: ApiClientOptions): ApiClient {
  // Returns the parsed JSON body (typed by the caller from the generated schema) and the response.
  async function request(
    method: 'GET' | 'POST' | 'DELETE',
    path: string,
    options: { body?: unknown; headers?: Record<string, string>; signIn?: boolean } = {},
  ): Promise<{ data: unknown; response: Response }> {
    const headers: Record<string, string> = { Accept: 'application/json', ...options.headers };
    if (options.body !== undefined) {
      headers['Content-Type'] = 'application/json';
    }
    const token = csrfToken();
    if (method !== 'GET' && !options.signIn && token !== undefined) {
      headers['X-CSRF-Token'] = token;
    }
    let response: Response;
    try {
      response = await transport(BASE + path, {
        method,
        headers,
        body: options.body === undefined ? undefined : JSON.stringify(options.body),
      });
    } catch {
      throw new NetworkError();
    }
    let data: unknown = undefined;
    if (response.status !== 204) {
      try {
        data = await response.json();
      } catch {
        data = undefined;
      }
    }
    if (!response.ok) {
      if (response.status === 401 && !options.signIn) {
        onUnauthenticated?.();
      }
      throw new ApiError(response.status, isProblem(data) ? data : undefined);
    }
    return { data, response };
  }

  return {
    async getSession() {
      return (await request('GET', '/session')).data as Session;
    },
    async signIn(email, password) {
      const { data } = await request('POST', '/session', {
        body: { email, password },
        signIn: true,
      });
      return data as Session;
    },
    async signOut() {
      await request('DELETE', '/session');
    },
    async listSenderIds() {
      return ((await request('GET', '/sender-ids')).data as Schemas['SenderIdList']).data;
    },
    async estimateMessage(body) {
      const { data } = await request('POST', '/message-estimates', { body: { body } });
      return data as MessageEstimate;
    },
    async listMessages(params = {}) {
      const query = new URLSearchParams();
      if (params.limit !== undefined) query.set('limit', String(params.limit));
      if (params.cursor !== undefined) query.set('cursor', params.cursor);
      const suffix = query.size > 0 ? `?${query.toString()}` : '';
      return (await request('GET', `/messages${suffix}`)).data as MessageList;
    },
    async getMessage(id) {
      return (await request('GET', `/messages/${encodeURIComponent(id)}`)).data as MessageDetail;
    },
    async createMessage(body, idempotencyKey) {
      const { data, response } = await request('POST', '/messages', {
        body,
        headers: { 'Idempotency-Key': idempotencyKey },
      });
      return {
        message: data as Message,
        replayed: response.headers.get('Idempotent-Replayed') === 'true',
      };
    },
  };
}

function isProblem(value: unknown): value is Problem {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as { type?: unknown }).type === 'string' &&
    typeof (value as { status?: unknown }).status === 'number'
  );
}
