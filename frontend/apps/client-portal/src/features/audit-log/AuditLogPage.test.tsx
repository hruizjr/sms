import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { AuditEvent } from '../../lib/api/client';
import {
  auditEventFixture,
  json,
  problem,
  sessionFor,
  type FakeRequest,
  type Handler,
} from '../../test/fake-api';
import { renderApp } from '../../test/render';
import { manilaDayStart } from './events';

const BASE = '/portal/v1/audit-events';

const EVENTS: AuditEvent[] = [
  auditEventFixture({
    id: '0193f0a1-0000-7000-8000-000000000003',
    occurred_at: '2026-11-28T02:10:00Z',
    action: 'api_key.create',
    outcome: 'success',
    target: { type: 'api_key', id: '06h1wh346hsf13tppjr44p43zz', email: null },
    details: { api_key_name: 'Billing system', api_key_scopes: ['messages:send'] },
  }),
  auditEventFixture({
    id: '0193f0a1-0000-7000-8000-000000000002',
    occurred_at: '2026-11-28T02:05:00Z',
    action: 'portal.sign_in',
    outcome: 'failure',
    actor: { type: 'anonymous', id: null, email: null },
    target: {
      type: 'portal_user',
      id: '0193f0a1-0000-7000-8000-0000000000bb',
      email: 'sender@example.invalid',
    },
    source_ip: '198.51.100.7',
    details: { reason: 'invalid_password' },
  }),
  auditEventFixture({
    id: '0193f0a1-0000-7000-8000-000000000001',
    occurred_at: '2026-11-28T02:00:00Z',
    action: 'audit_log.list',
    outcome: 'denied',
    actor: {
      type: 'portal_user',
      id: '0193f0a1-0000-7000-8000-0000000000cc',
      email: 'viewer@example.invalid',
    },
    target: null,
    details: {
      reason: 'insufficient_role',
      role: 'viewer',
      required_permission: 'audit_log:read',
      dedup_interval_seconds: 300,
    },
  }),
];

/** A signed-in handler whose audit endpoint is answered by audit (with the parsed query). */
function withAudit(
  role: 'owner' | 'sender' | 'viewer',
  audit: (query: URLSearchParams, request: FakeRequest) => Response,
): Handler {
  return (request) => {
    if (request.method === 'GET' && request.path === '/portal/v1/session') {
      return json(200, sessionFor(role));
    }
    if (request.method === 'GET' && request.path.startsWith(BASE)) {
      return audit(new URL(request.path, 'https://portal.example.invalid').searchParams, request);
    }
    return problem(404, 'not-found');
  };
}

function page(data: AuditEvent[], next: string | null = null) {
  return json(200, { data, next_cursor: next });
}

function auditCalls(calls: FakeRequest[]): URLSearchParams[] {
  return calls
    .filter((c) => c.path.startsWith(BASE))
    .map((c) => new URL(c.path, 'https://portal.example.invalid').searchParams);
}

describe('AuditLogPage', () => {
  it('lists the organization’s events for owners, newest first, in plain language', async () => {
    renderApp(
      '/audit-log',
      withAudit('owner', () => page(EVENTS)),
    );
    expect(await screen.findByRole('heading', { name: 'Audit log', level: 1 })).toBeInTheDocument();
    const table = await screen.findByRole('table', { name: 'Security audit events, newest first' });
    const rows = within(table).getAllByRole('row').slice(1);
    expect(rows).toHaveLength(3);
    const [created, failed, denied] = rows;
    if (created === undefined || failed === undefined || denied === undefined) throw new Error();
    expect(created).toHaveTextContent('API key created');
    expect(created).toHaveTextContent('“Billing system”');
    expect(created).toHaveTextContent('Success');
    expect(created).toHaveTextContent('owner@example.invalid');
    expect(created).toHaveTextContent('Nov 28, 2026, 10:10 AM PHT');
    expect(failed).toHaveTextContent('Sign-in failed');
    expect(failed).toHaveTextContent('The password was wrong');
    expect(failed).toHaveTextContent('Failure');
    expect(failed).toHaveTextContent('Not signed in');
    expect(failed).toHaveTextContent('198.51.100.7');
    expect(denied).toHaveTextContent('Audit log access refused');
    expect(denied).toHaveTextContent('The user’s role does not allow it');
    expect(denied).toHaveTextContent('Denied');
    expect(screen.getByText(/Showing/)).toHaveTextContent('Showing 3 events');
    // The navigation offers the page to owners and marks it current.
    const nav = screen.getByRole('navigation', { name: 'Main' });
    expect(within(nav).getByRole('link', { name: 'Audit log' })).toHaveAttribute(
      'aria-current',
      'page',
    );
  });

  it('shows an account lock with its stored outcome “Enforced” and the end of the lock', async () => {
    renderApp(
      '/audit-log',
      withAudit('owner', () =>
        page([
          auditEventFixture({
            action: 'portal.account_locked',
            outcome: 'enforced',
            actor: { type: 'system', id: null, email: null },
            target: {
              type: 'portal_user',
              id: '0193f0a1-0000-7000-8000-0000000000bb',
              email: 'sender@example.invalid',
            },
            details: { reason: 'too_many_attempts', locked_until: '2026-11-28T02:20:00Z' },
          }),
        ]),
      ),
    );
    const table = await screen.findByRole('table');
    const row = within(table).getAllByRole('row')[1];
    expect(row).toHaveTextContent('Account locked');
    // The badge shows the stored outcome itself: no override relabels another value.
    expect(row).toHaveTextContent('Enforced');
    expect(row).not.toHaveTextContent('Success');
    expect(row).toHaveTextContent('System');
    fireEvent.click(within(table).getByRole('button', { name: /^Details of “Account locked”/ }));
    const dialog = await screen.findByRole('dialog', { name: 'Account locked' });
    expect(dialog).toHaveTextContent('sender@example.invalid');
    expect(dialog).toHaveTextContent('Locked untilNov 28, 2026, 10:20:00 AM PHT');
  });

  it('filters by event, outcome and Philippine-time date range, and clears the filters', async () => {
    const { transport } = renderApp(
      '/audit-log',
      withAudit('owner', () => page(EVENTS)),
    );
    await screen.findByRole('table');
    fireEvent.change(screen.getByLabelText('Event'), { target: { value: 'portal.sign_in' } });
    fireEvent.change(screen.getByLabelText('Outcome'), { target: { value: 'failure' } });
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '2026-11-28' } });
    fireEvent.change(screen.getByLabelText('To (inclusive)'), {
      target: { value: '2026-11-28' },
    });
    await waitFor(() => {
      expect(auditCalls(transport.calls).at(-1)?.get('occurred_before')).not.toBeNull();
    });
    const last = auditCalls(transport.calls).at(-1);
    expect(last?.get('action')).toBe('portal.sign_in');
    expect(last?.get('outcome')).toBe('failure');
    // 28 November in Philippine Time (UTC+8) is [27 Nov 16:00 UTC, 28 Nov 16:00 UTC).
    expect(last?.get('occurred_from')).toBe('2026-11-27T16:00:00.000Z');
    expect(last?.get('occurred_before')).toBe('2026-11-28T16:00:00.000Z');
    expect(last?.get('limit')).toBe('50');

    fireEvent.click(await screen.findByRole('button', { name: 'Clear filters' }));
    await waitFor(() => {
      expect([...(auditCalls(transport.calls).at(-1)?.keys() ?? [])]).toEqual(['limit']);
    });
    expect(screen.getByLabelText('Event')).toHaveValue('');
  });

  it('offers every stored outcome, including “Enforced”, as a filter', async () => {
    const { transport } = renderApp(
      '/audit-log',
      withAudit('owner', () => page(EVENTS)),
    );
    await screen.findByRole('table');
    const options = within(screen.getByLabelText('Outcome'))
      .getAllByRole('option')
      .map((o) => o.textContent);
    expect(options).toEqual(expect.arrayContaining(['Success', 'Failure', 'Denied', 'Enforced']));
    fireEvent.change(screen.getByLabelText('Outcome'), { target: { value: 'enforced' } });
    await waitFor(() => {
      expect(auditCalls(transport.calls).at(-1)?.get('outcome')).toBe('enforced');
    });
  });

  it('describes administration events and does not claim tamper-proof storage', async () => {
    renderApp(
      '/audit-log',
      withAudit('owner', () =>
        page([
          auditEventFixture({
            action: 'portal_user.create',
            outcome: 'success',
            actor: { type: 'system', id: null, email: null },
            target: {
              type: 'portal_user',
              id: '0193f0a1-0000-7000-8000-0000000000dd',
              email: 'new@example.invalid',
            },
            request_id: null,
            source_ip: null,
            user_agent: null,
            details: { role: 'sender', via: 'smsctl' },
          }),
        ]),
      ),
    );
    const table = await screen.findByRole('table');
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('User created');
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('System (smsctl)');
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('new@example.invalid');
    expect(screen.queryByText(/cannot be changed or deleted/)).not.toBeInTheDocument();
    expect(screen.getByText(/this is not tamper-proof storage/)).toBeInTheDocument();
    fireEvent.click(within(table).getByRole('button', { name: /^Details of “User created”/ }));
    const dialog = await screen.findByRole('dialog', { name: 'User created' });
    expect(dialog).toHaveTextContent('Rolesender');
  });

  it('refuses an end date before the start date without a request', async () => {
    const { transport } = renderApp(
      '/audit-log',
      withAudit('owner', () => page(EVENTS)),
    );
    await screen.findByRole('table');
    const before = auditCalls(transport.calls).length;
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '2026-11-28' } });
    fireEvent.change(screen.getByLabelText('To (inclusive)'), {
      target: { value: '2026-11-27' },
    });
    expect(
      await screen.findByText('The end date must be on or after the start date.'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('To (inclusive)')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
    // Only the "From" change was sent.
    expect(auditCalls(transport.calls).length).toBe(before + 1);
  });

  it('shows the empty states', async () => {
    renderApp(
      '/audit-log',
      withAudit('owner', (q) => (q.has('outcome') ? page([]) : page([]))),
    );
    expect(
      await screen.findByRole('region', { name: 'No security events yet' }),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('Outcome'), { target: { value: 'denied' } });
    expect(
      await screen.findByRole('region', { name: 'No events match these filters' }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Show all events' }));
    expect(
      await screen.findByRole('region', { name: 'No security events yet' }),
    ).toBeInTheDocument();
  });

  it('shows an error with a retry', async () => {
    let fail = true;
    renderApp(
      '/audit-log',
      withAudit('owner', () => (fail ? problem(500, 'internal') : page(EVENTS))),
    );
    expect(await screen.findByText('Could not load the audit log')).toBeInTheDocument();
    fail = false;
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('table')).toBeInTheDocument();
  });

  it('loads more events with the cursor and the same filters', async () => {
    const { transport } = renderApp(
      '/audit-log',
      withAudit('owner', (q) =>
        q.get('cursor') === 'next-page'
          ? page([
              auditEventFixture({
                id: '0193f0a1-0000-7000-8000-0000000000ff',
                action: 'portal.sign_out',
              }),
            ])
          : page(EVENTS, 'next-page'),
      ),
    );
    await screen.findByRole('table');
    fireEvent.change(screen.getByLabelText('Event'), { target: { value: 'portal.sign_out' } });
    fireEvent.click(await screen.findByRole('button', { name: 'Load more' }));
    await waitFor(() => {
      expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(5);
    });
    const last = auditCalls(transport.calls).at(-1);
    expect(last?.get('cursor')).toBe('next-page');
    expect(last?.get('action')).toBe('portal.sign_out');
    expect(screen.getByText(/Showing/)).toHaveTextContent('Showing 4 events');
    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument();
  });

  it('shows every recorded field of an event in a dialog', async () => {
    renderApp(
      '/audit-log',
      withAudit('owner', () => page(EVENTS)),
    );
    const table = await screen.findByRole('table');
    fireEvent.click(
      within(table).getByRole('button', { name: /^Details of “Audit log access refused”/ }),
    );
    const dialog = await screen.findByRole('dialog', { name: 'Audit log access refused' });
    expect(dialog).toHaveTextContent('Denied');
    expect(dialog).toHaveTextContent('The user’s role does not allow it');
    expect(dialog).toHaveTextContent('viewer@example.invalid');
    expect(dialog).toHaveTextContent('audit_log:read');
    expect(dialog).toHaveTextContent('role viewer');
    expect(dialog).toHaveTextContent(
      'Repeated refusals of this user and action within 5 minutes are recorded once.',
    );
    expect(dialog).toHaveTextContent('203.0.113.10');
    expect(dialog).toHaveTextContent('Mozilla/5.0 (Example)');
    expect(dialog).toHaveTextContent('9b2f4c1e7d3a5b60');
    expect(within(dialog).getByRole('button', { name: 'Copy request ID' })).toBeInTheDocument();
    expect(within(dialog).getByRole('button', { name: 'Close' })).toHaveFocus();
    fireEvent.click(within(dialog).getByRole('button', { name: 'Close' }));
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    fireEvent.click(within(table).getByRole('button', { name: /^Details of “API key created”/ }));
    const keyDialog = await screen.findByRole('dialog', { name: 'API key created' });
    expect(keyDialog).toHaveTextContent('API key “Billing system”');
    expect(keyDialog).toHaveTextContent('Key ID 06h1wh346hsf13tppjr44p43zz');
    expect(keyDialog).toHaveTextContent('messages:send');
  });

  it.each(['sender', 'viewer'] as const)(
    'is not offered to a %s, and the page asks nothing of the server',
    async (role) => {
      const { transport } = renderApp(
        '/audit-log',
        withAudit(role, () => page(EVENTS)),
      );
      expect(
        await screen.findByText('Only organization owners can view the audit log'),
      ).toBeInTheDocument();
      const nav = screen.getByRole('navigation', { name: 'Main' });
      expect(within(nav).queryByRole('link', { name: 'Audit log' })).not.toBeInTheDocument();
      expect(auditCalls(transport.calls)).toHaveLength(0);
    },
  );
});

describe('manilaDayStart', () => {
  it('converts a calendar date in Philippine Time to UTC', () => {
    expect(manilaDayStart('2026-01-01')).toBe('2025-12-31T16:00:00.000Z');
    expect(manilaDayStart('2026-02-28', 1)).toBe('2026-02-28T16:00:00.000Z');
    expect(manilaDayStart('28/11/2026')).toBeUndefined();
    expect(manilaDayStart('2026-13-01')).toBeUndefined();
  });
});
