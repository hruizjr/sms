import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CSRF, apiKeyFixture, json, problem, signedIn, type Handler } from '../../test/fake-api';
import { renderApp } from '../../test/render';

const LIST = 'GET /portal/v1/api-keys';
const CREATE = 'POST /portal/v1/api-keys';
const NEW_KEY = `smsk_06h1wh346hsf13tppjr44p43aa_06h1wh346hsf13tppjr44p43zz_${'s'.repeat(52)}`;

const keys = (...data: unknown[]) => json(200, { data });

function created(overrides = {}) {
  return json(201, {
    api_key: apiKeyFixture({
      id: '06h1wh346hsf13tppjr44p43zz',
      name: 'Billing system',
      scopes: ['messages:send'],
      created_at: '2026-11-27T09:00:00Z',
      ...overrides,
    }),
    key: NEW_KEY,
  });
}

function setClipboard(writeText: ((text: string) => Promise<void>) | undefined) {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: writeText === undefined ? undefined : { writeText },
  });
}

afterEach(() => {
  setClipboard(undefined);
});

async function openCreateDialog() {
  fireEvent.click(await screen.findByRole('button', { name: 'Create API key' }));
  return screen.getByRole('dialog', { name: 'Create API key' });
}

async function createKey(handler: Handler) {
  const result = renderApp('/api-keys', handler);
  const dialog = await openCreateDialog();
  fireEvent.change(within(dialog).getByLabelText('Name'), {
    target: { value: '  Billing system ' },
  });
  fireEvent.click(within(dialog).getByRole('checkbox', { name: /Read messages/ }));
  fireEvent.click(within(dialog).getByRole('button', { name: 'Create key' }));
  await screen.findByRole('region', { name: 'API key “Billing system” created' });
  return result;
}

describe('API keys page', () => {
  it('shows a loading state, then an empty state with a create action', async () => {
    renderApp('/api-keys', signedIn('owner', { [LIST]: () => keys() }));
    expect(await screen.findByText('Loading API keys…')).toBeInTheDocument();
    const empty = await screen.findByRole('region', { name: 'No API keys yet' });
    expect(within(empty).getByRole('button', { name: 'Create API key' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'API keys' })).toBeInTheDocument();
    expect(
      within(screen.getByRole('navigation', { name: 'Main' })).getByRole('link', {
        name: 'API keys',
      }),
    ).toHaveAttribute('aria-current', 'page');
  });

  it('reports a failure with a retry', async () => {
    let fail = true;
    renderApp(
      '/api-keys',
      signedIn('owner', { [LIST]: () => (fail ? problem(500, 'internal') : keys()) }),
    );
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load API keys');
    fail = false;
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('region', { name: 'No API keys yet' })).toBeInTheDocument();
  });

  it('lists active keys first with their metadata, and revoked keys without a revoke action', async () => {
    renderApp(
      '/api-keys',
      signedIn('owner', {
        [LIST]: () =>
          keys(
            apiKeyFixture({
              id: '06h1wh346hsf13tppjr44p43r1',
              name: 'Old integration',
              status: 'revoked',
              revoked_at: '2026-11-27T10:00:00Z',
            }),
            apiKeyFixture({ name: 'Order notifications', scopes: ['messages:read'] }),
          ),
      }),
    );
    const table = await screen.findByRole('table', { name: 'API keys, active keys first' });
    const rows = within(table).getAllByRole('row').slice(1);
    expect(rows).toHaveLength(2);
    const [active, revoked] = rows;
    if (active === undefined || revoked === undefined) throw new Error('rows missing');
    expect(rows[0]).toHaveTextContent('Order notifications');
    expect(rows[0]).toHaveTextContent('messages:read');
    expect(rows[0]).toHaveTextContent('Active');
    expect(rows[0]).toHaveTextContent('smsk_…r44p43pw');
    expect(rows[1]).toHaveTextContent('Old integration');
    expect(rows[1]).toHaveTextContent('Revoked');
    expect(
      within(active).getByRole('button', {
        name: 'Revoke API key Order notifications',
      }),
    ).toBeInTheDocument();
    expect(within(revoked).queryByRole('button')).not.toBeInTheDocument();
    expect(screen.getByText(/does not record when a key was last used/)).toBeInTheDocument();
  });

  it('validates the name and permissions before creating', async () => {
    const { transport } = renderApp('/api-keys', signedIn('owner', { [LIST]: () => keys() }));
    const dialog = await openCreateDialog();
    expect(within(dialog).getByLabelText('Name')).toHaveFocus();
    fireEvent.click(within(dialog).getByRole('checkbox', { name: /Send messages/ }));
    fireEvent.click(within(dialog).getByRole('checkbox', { name: /Read messages/ }));
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create key' }));
    expect(within(dialog).getByText('Enter a name for the key.')).toBeInTheDocument();
    expect(within(dialog).getByText('Choose at least one permission.')).toBeInTheDocument();
    expect(within(dialog).getByLabelText('Name')).toHaveAttribute('aria-invalid', 'true');
    expect(transport.calls.filter((c) => c.method === 'POST')).toHaveLength(0);
  });

  it('creates a key once, with the CSRF token, and shows it once with a warning, its permissions and an example', async () => {
    let resolve: (r: Response) => void = () => undefined;
    const { transport } = renderApp(
      '/api-keys',
      signedIn('owner', {
        [LIST]: () => keys(),
        [CREATE]: () =>
          new Promise<Response>((r) => {
            resolve = r;
          }),
      }),
    );
    const dialog = await openCreateDialog();
    fireEvent.change(within(dialog).getByLabelText('Name'), {
      target: { value: '  Billing system ' },
    });
    fireEvent.click(within(dialog).getByRole('checkbox', { name: /Read messages/ }));
    const submit = within(dialog).getByRole('button', { name: 'Create key' });
    fireEvent.click(submit);
    fireEvent.click(submit);
    expect(await within(dialog).findByRole('button', { name: 'Creating…' })).toBeDisabled();
    act(() => {
      resolve(created());
    });

    const posts = transport.calls.filter((c) => c.method === 'POST');
    expect(posts).toHaveLength(1);
    expect(posts[0]).toMatchObject({
      path: '/portal/v1/api-keys',
      headers: { 'X-CSRF-Token': CSRF },
      body: { name: 'Billing system', scopes: ['messages:send'] },
    });

    const panel = await screen.findByRole('region', { name: 'API key “Billing system” created' });
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(within(panel).getByLabelText('Your new API key')).toHaveValue(NEW_KEY);
    expect(within(panel).getByLabelText('Your new API key')).toHaveAccessibleDescription(
      /only time the full key is shown/,
    );
    expect(panel).toHaveTextContent('messages:send');
    expect(panel).not.toHaveTextContent('messages:read');
    const example = within(panel).getByText(/curl -X POST/);
    expect(example).toHaveTextContent('/v1/messages');
    expect(example).toHaveTextContent('Bearer $SMS_API_KEY');
    expect(example).toHaveTextContent('Idempotency-Key');
    expect(example.textContent).not.toContain(NEW_KEY);
    // The new key also appears in the list, without its secret.
    const table = screen.getByRole('table');
    expect(table).toHaveTextContent('Billing system');
    expect(table.textContent).not.toContain(NEW_KEY);
  });

  it('copies the key to the clipboard', async () => {
    const writeText = vi.fn(() => Promise.resolve());
    setClipboard(writeText);
    await createKey(signedIn('owner', { [LIST]: () => keys(), [CREATE]: () => created() }));
    fireEvent.click(screen.getByRole('button', { name: 'Copy key' }));
    expect(await screen.findByRole('button', { name: 'Copied' })).toBeInTheDocument();
    expect(writeText).toHaveBeenCalledWith(NEW_KEY);
    expect(screen.getByText('The key was copied to your clipboard.')).toBeInTheDocument();
  });

  it('selects the key for manual copying when the clipboard is not available', async () => {
    setClipboard(undefined);
    await createKey(signedIn('owner', { [LIST]: () => keys(), [CREATE]: () => created() }));
    fireEvent.click(screen.getByRole('button', { name: 'Copy key' }));
    expect(await screen.findByText(/copy it with your keyboard/)).toBeInTheDocument();
    expect(screen.getByLabelText('Your new API key')).toHaveFocus();
  });

  /* eslint-disable no-restricted-properties -- this test reads browser storage to prove the key is never written to it */
  it('forgets the key when dismissed and never stores it in the browser', async () => {
    await createKey(signedIn('owner', { [LIST]: () => keys(), [CREATE]: () => created() }));
    expect(window.localStorage.length).toBe(0);
    expect(window.sessionStorage.length).toBe(0);
    expect(document.cookie).toBe('');
    fireEvent.click(screen.getByRole('button', { name: 'Done' }));
    expect(screen.queryByLabelText('Your new API key')).not.toBeInTheDocument();
    expect(document.body.textContent).not.toContain(NEW_KEY);
    expect(screen.getByRole('button', { name: 'Create API key' })).toHaveFocus();
  });
  /* eslint-enable no-restricted-properties */

  it('maps server validation errors to the fields', async () => {
    renderApp(
      '/api-keys',
      signedIn('owner', {
        [LIST]: () => keys(),
        [CREATE]: () =>
          problem(400, 'validation-error', {
            errors: [{ pointer: '/name', code: 'invalid_value', detail: 'Server says no.' }],
          }),
      }),
    );
    const dialog = await openCreateDialog();
    fireEvent.change(within(dialog).getByLabelText('Name'), { target: { value: 'x' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create key' }));
    expect(await within(dialog).findByText('Server says no.')).toBeInTheDocument();
    expect(within(dialog).getByRole('alert')).toHaveTextContent('Check the highlighted fields.');
  });

  it('asks for confirmation before revoking, and can be cancelled', async () => {
    const { transport } = renderApp(
      '/api-keys',
      signedIn('owner', { [LIST]: () => keys(apiKeyFixture()) }),
    );
    fireEvent.click(
      await screen.findByRole('button', { name: 'Revoke API key Order notifications' }),
    );
    const dialog = screen.getByRole('dialog', { name: 'Revoke “Order notifications”?' });
    expect(dialog).toHaveAccessibleDescription(/cannot be undone/);
    fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(transport.calls.filter((c) => c.method === 'DELETE')).toHaveLength(0);
  });

  it('revokes a key with the CSRF token and shows it as revoked', async () => {
    const key = apiKeyFixture();
    const { transport } = renderApp(
      '/api-keys',
      signedIn('owner', {
        [LIST]: () => keys(key),
        [`DELETE /portal/v1/api-keys/${key.id}`]: () =>
          json(200, { ...key, status: 'revoked', revoked_at: '2026-11-27T11:00:00Z' }),
      }),
    );
    fireEvent.click(
      await screen.findByRole('button', { name: 'Revoke API key Order notifications' }),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Revoke key' }));
    expect(
      await screen.findByText('API key “Order notifications” was revoked.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    const rows = within(screen.getByRole('table')).getAllByRole('row');
    expect(rows[1]).toHaveTextContent('Revoked');
    expect(screen.queryByRole('button', { name: /Revoke API key/ })).not.toBeInTheDocument();
    expect(transport.calls.find((c) => c.method === 'DELETE')).toMatchObject({
      headers: { 'X-CSRF-Token': CSRF },
    });
  });

  it('keeps the dialog open with an error when revocation fails, and refreshes when the key is gone', async () => {
    const key = apiKeyFixture();
    let answer = () => problem(500, 'internal');
    let lists = 0;
    renderApp(
      '/api-keys',
      signedIn('owner', {
        [LIST]: () => {
          lists++;
          return keys(key);
        },
        [`DELETE /portal/v1/api-keys/${key.id}`]: () => answer(),
      }),
    );
    fireEvent.click(
      await screen.findByRole('button', { name: 'Revoke API key Order notifications' }),
    );
    const dialog = screen.getByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Revoke key' }));
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'The key could not be revoked. Try again.',
    );
    answer = () => problem(404, 'not-found');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Revoke key' }));
    expect(await screen.findByText(/already revoked/)).toBeInTheDocument();
    await waitFor(() => {
      expect(lists).toBe(2);
    });
  });

  it('tells senders and viewers that only owners manage keys, without calling the API', async () => {
    for (const role of ['sender', 'viewer'] as const) {
      const { transport, unmount } = renderApp('/api-keys', signedIn(role));
      expect(
        await screen.findByText('Only organization owners can manage API keys'),
      ).toBeInTheDocument();
      expect(screen.queryByRole('link', { name: 'API keys' })).not.toBeInTheDocument();
      expect(transport.calls.some((c) => c.path.startsWith('/portal/v1/api-keys'))).toBe(false);
      unmount();
    }
  });
});
