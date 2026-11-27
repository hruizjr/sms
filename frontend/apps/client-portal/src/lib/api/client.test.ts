import { describe, expect, it, vi } from 'vitest';
import { CSRF, fakeTransport, json, problem, sessionFor } from '../../test/fake-api';
import { ApiError, createApiClient, NetworkError } from './client';

function client(handler: Parameters<typeof fakeTransport>[0], token: string | undefined = CSRF) {
  const transport = fakeTransport(handler);
  const onUnauthenticated = vi.fn();
  const api = createApiClient({ transport, csrfToken: () => token, onUnauthenticated });
  return { api, transport, onUnauthenticated };
}

describe('portal API client', () => {
  it('reads with GET to the same origin and sends no CSRF token', async () => {
    const { api, transport } = client(() => json(200, sessionFor()));
    await expect(api.getSession()).resolves.toMatchObject({ user: { role: 'owner' } });
    expect(transport.calls).toEqual([
      {
        method: 'GET',
        path: '/portal/v1/session',
        headers: { Accept: 'application/json' },
        body: undefined,
      },
    ]);
  });

  it('sends the CSRF token and the Idempotency-Key with a message submission', async () => {
    const { api, transport } = client(() =>
      json(202, { id: 'msg_1' }, { 'Idempotent-Replayed': 'true' }),
    );
    const request = { to: '+639000000001', sender_id: 'EXAMPLECO', body: 'Hello' };
    await expect(api.createMessage(request, 'key-1')).resolves.toEqual({
      message: { id: 'msg_1' },
      replayed: true,
    });
    const [call] = transport.calls;
    expect(call?.path).toBe('/portal/v1/messages');
    expect(call?.headers).toMatchObject({
      'X-CSRF-Token': CSRF,
      'Idempotency-Key': 'key-1',
      'Content-Type': 'application/json',
    });
    expect(call?.body).toEqual(request);
  });

  it('never sends a tenant: the body is exactly the message fields', async () => {
    const { api, transport } = client(() => json(202, { id: 'msg_1' }));
    await api.createMessage({ to: '+639000000001', sender_id: 'X', body: 'b' }, 'k');
    expect(Object.keys(transport.calls[0]?.body as object).sort()).toEqual([
      'body',
      'sender_id',
      'to',
    ]);
  });

  it('signs in without a CSRF token and does not treat a failed sign-in as an ended session', async () => {
    const { api, transport, onUnauthenticated } = client(() => problem(401, 'invalid-credentials'));
    const error = await api.signIn('a@example.invalid', 'pw').catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).problemType).toBe('invalid-credentials');
    expect(transport.calls[0]?.headers['X-CSRF-Token']).toBeUndefined();
    expect(onUnauthenticated).not.toHaveBeenCalled();
  });

  it('reports a 401 on any other request centrally', async () => {
    const { api, onUnauthenticated } = client(() => problem(401, 'unauthenticated'));
    await expect(api.listMessages()).rejects.toBeInstanceOf(ApiError);
    expect(onUnauthenticated).toHaveBeenCalledOnce();
  });

  it('turns a failed connection into a NetworkError', async () => {
    const { api } = client(() => {
      throw new TypeError('Failed to fetch');
    });
    await expect(api.getMessage('msg_1')).rejects.toBeInstanceOf(NetworkError);
  });

  it('builds list queries and encodes path parameters', async () => {
    const { api, transport } = client(() => json(200, { data: [], next_cursor: null }));
    await api.listMessages({ limit: 25, cursor: 'abc/def' });
    await api.getMessage('msg_../x').catch(() => undefined);
    expect(transport.calls.map((c) => c.path)).toEqual([
      '/portal/v1/messages?limit=25&cursor=abc%2Fdef',
      '/portal/v1/messages/msg_..%2Fx',
    ]);
  });

  it('signs out with DELETE and the CSRF token', async () => {
    const { api, transport } = client(() => new Response(null, { status: 204 }));
    await api.signOut();
    expect(transport.calls[0]).toMatchObject({
      method: 'DELETE',
      path: '/portal/v1/session',
      headers: { 'X-CSRF-Token': CSRF },
    });
  });
});
