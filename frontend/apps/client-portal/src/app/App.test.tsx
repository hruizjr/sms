import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import indexHtml from '../../index.html?raw';
import { SESSION_ENDED_NOTICE } from '../features/auth/session';
import { CSRF, json, problem, sessionFor, signedIn } from '../test/fake-api';
import { renderApp } from '../test/render';

const emptyList = () => json(200, { data: [], next_cursor: null });

describe('portal shell and session', () => {
  it('shows a loading state while the session is checked', () => {
    renderApp('/messages', () => new Promise<Response>(() => undefined));
    expect(screen.getByRole('status')).toHaveTextContent('Checking your session…');
    expect(screen.getByRole('main')).toBeInTheDocument();
  });

  it('shows the sign-in page without a session, on any path, without navigation', async () => {
    renderApp('/messages/new', () => problem(401, 'unauthenticated'));
    expect(await screen.findByRole('heading', { level: 1, name: 'Sign in' })).toBeInTheDocument();
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
    expect(screen.queryByText(SESSION_ENDED_NOTICE)).not.toBeInTheDocument();
  });

  it('reports an unreachable server with a retry', async () => {
    let up = false;
    renderApp('/messages', (r) => {
      if (!up) throw new TypeError('Failed to fetch');
      return signedIn('owner', { 'GET /portal/v1/messages?limit=25': emptyList })(r);
    });
    expect(await screen.findByRole('alert')).toHaveTextContent('The portal is not available');
    up = true;
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('heading', { level: 1, name: 'Messages' })).toBeInTheDocument();
  });

  it('signs in, then shows the messages with the organization and user in the header', async () => {
    let signedInYet = false;
    const { transport } = renderApp('/', (r) => {
      if (r.method === 'POST' && r.path === '/portal/v1/session') {
        signedInYet = true;
        return json(200, sessionFor('owner'));
      }
      if (!signedInYet) return problem(401, 'unauthenticated');
      return signedIn('owner', { 'GET /portal/v1/messages?limit=25': emptyList })(r);
    });
    fireEvent.change(await screen.findByLabelText('Email address'), {
      target: { value: 'owner@example.invalid' },
    });
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'a long password' } });
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'Messages' })).toBeInTheDocument();
    expect(window.location.pathname).toBe('/messages');
    const banner = screen.getByRole('banner');
    expect(banner).toHaveTextContent('Example Co');
    expect(banner).toHaveTextContent('owner@example.invalid (Owner)');
    const nav = within(screen.getByRole('navigation', { name: 'Main' }));
    expect(nav.getByRole('link', { name: 'Messages' })).toHaveAttribute('aria-current', 'page');
    expect(nav.getByRole('link', { name: 'Send SMS' })).toBeInTheDocument();
    expect(transport.calls.find((c) => c.method === 'POST')?.body).toEqual({
      email: 'owner@example.invalid',
      password: 'a long password',
    });
  });

  it('hides sending from a viewer (the server enforces it as well)', async () => {
    renderApp('/messages', signedIn('viewer', { 'GET /portal/v1/messages?limit=25': emptyList }));
    expect(await screen.findByRole('heading', { level: 1, name: 'Messages' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Send SMS' })).not.toBeInTheDocument();
    expect(screen.getByRole('banner')).toHaveTextContent('(Viewer)');
  });

  it('returns to the sign-in page with a notice when the session ends', async () => {
    let expired = false;
    renderApp('/messages', (r) =>
      expired
        ? problem(401, 'unauthenticated')
        : signedIn('owner', { 'GET /portal/v1/messages?limit=25': emptyList })(r),
    );
    // Wait until the list has loaded (Refresh is disabled while loading).
    expect(await screen.findByRole('region', { name: 'No messages yet' })).toBeInTheDocument();
    expired = true;
    fireEvent.click(screen.getByRole('button', { name: 'Refresh messages' }));
    expect(await screen.findByText(SESSION_ENDED_NOTICE, { exact: false })).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'Sign in' })).toBeInTheDocument();
  });

  it('signs out on the server with the CSRF token', async () => {
    const { transport } = renderApp(
      '/messages',
      signedIn('owner', {
        'GET /portal/v1/messages?limit=25': emptyList,
        'DELETE /portal/v1/session': () => new Response(null, { status: 204 }),
      }),
    );
    fireEvent.click(await screen.findByRole('button', { name: 'Sign out' }));
    expect(await screen.findByRole('heading', { level: 1, name: 'Sign in' })).toBeInTheDocument();
    expect(transport.calls.at(-1)).toMatchObject({
      method: 'DELETE',
      headers: { 'X-CSRF-Token': CSRF },
    });
  });

  it('starts with a skip link that targets the main content', async () => {
    const { container } = renderApp('/messages', () => problem(401, 'unauthenticated'));
    await screen.findByRole('heading', { name: 'Sign in' });
    const skipLink = screen.getByRole('link', { name: 'Skip to main content' });
    const firstFocusable = container.querySelector('a[href], button, input, select, textarea');
    expect(firstFocusable).toBe(skipLink);
    expect(skipLink).toHaveAttribute('href', `#${screen.getByRole('main').id}`);
    expect(screen.getByRole('main')).toHaveAttribute('tabindex', '-1');
  });

  it('shows a not-found page for unknown paths', async () => {
    renderApp('/nowhere', signedIn('owner'));
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Page not found' }),
    ).toBeInTheDocument();
  });

  it('declares the document language and title', () => {
    expect(indexHtml).toMatch(/<html lang="en">/);
    expect(indexHtml).toMatch(/<title>[^<]+<\/title>/);
  });

  /* eslint-disable no-restricted-properties -- this test reads browser storage to prove the portal never writes to it */
  it('never stores anything in browser storage or readable cookies', async () => {
    renderApp('/messages', signedIn('owner', { 'GET /portal/v1/messages?limit=25': emptyList }));
    await screen.findByRole('heading', { level: 1, name: 'Messages' });
    await waitFor(() => {
      expect(window.localStorage.length).toBe(0);
    });
    expect(window.sessionStorage.length).toBe(0);
    expect(document.cookie).toBe('');
  });
  /* eslint-enable no-restricted-properties */
});
