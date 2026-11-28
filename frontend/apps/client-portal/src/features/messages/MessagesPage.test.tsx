import { fireEvent, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { json, messageFixture, problem, signedIn } from '../../test/fake-api';
import { renderApp } from '../../test/render';

const FIRST = 'GET /portal/v1/messages?limit=25';

describe('messages page', () => {
  it('shows a loading state, then the messages with their key fields', async () => {
    renderApp(
      '/messages',
      signedIn('owner', {
        [FIRST]: () =>
          json(200, {
            data: [
              messageFixture({ status_updated_at: '2026-11-27T08:07:00Z' }),
              messageFixture({
                id: 'msg_2',
                to: '+639000000030',
                status: 'submission_uncertain',
                status_category: 'uncertain',
              }),
            ],
            next_cursor: null,
          }),
      }),
    );
    expect(await screen.findByText('Loading messages…')).toBeInTheDocument();
    const table = await screen.findByRole('table', { name: 'Messages, newest first' });
    const [, firstRow, secondRow, ...rest] = within(table).getAllByRole('row');
    if (firstRow === undefined || secondRow === undefined) throw new Error('expected two rows');
    expect(rest).toHaveLength(0);
    const first = within(firstRow);
    expect(first.getByText('+639000000001')).toBeInTheDocument();
    expect(first.getByText('EXAMPLECO')).toBeInTheDocument();
    expect(
      within(table)
        .getAllByRole('columnheader')
        .map((th) => th.textContent),
    ).toEqual(['Recipient', 'Status', 'Sender ID', 'Segments', 'Submitted', 'Last update']);
    expect(first.getByRole('link', { name: 'msg_06H1WH346HSF13TPPJR44P43PW' })).toHaveAttribute(
      'href',
      '/messages/msg_06H1WH346HSF13TPPJR44P43PW',
    );
    // Submitted and last update, in Philippine time; segments are a column of their own.
    const cells = first.getAllByRole('cell').map((cell) => cell.textContent);
    expect(cells).toHaveLength(6);
    expect(cells[3]).toBe('1');
    expect(cells[4]).toMatch(/^Nov 27, 2026.*4:00.PM PHT$/);
    expect(cells[5]).toMatch(/^Nov 27, 2026.*4:07.PM PHT$/);
    expect(first.getByText('Delivered')).toBeInTheDocument();
    expect(within(secondRow).getByText('Uncertain')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument();
  });

  it('shows an empty state with a link to send the first SMS', async () => {
    renderApp(
      '/messages',
      signedIn('sender', { [FIRST]: () => json(200, { data: [], next_cursor: null }) }),
    );
    expect(await screen.findByRole('region', { name: 'No messages yet' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Send your first SMS' })).toHaveAttribute(
      'href',
      '/messages/new',
    );
  });

  it('shows an error state and recovers with Try again', async () => {
    let fail = true;
    renderApp(
      '/messages',
      signedIn('owner', {
        [FIRST]: () =>
          fail
            ? problem(500, 'internal')
            : json(200, { data: [messageFixture()], next_cursor: null }),
      }),
    );
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load messages');
    fail = false;
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('table')).toBeInTheDocument();
  });

  it('loads the next page with the cursor', async () => {
    const { transport } = renderApp(
      '/messages',
      signedIn('owner', {
        [FIRST]: () => json(200, { data: [messageFixture({ id: 'msg_1' })], next_cursor: 'c1' }),
        'GET /portal/v1/messages?limit=25&cursor=c1': () =>
          json(200, { data: [messageFixture({ id: 'msg_2' })], next_cursor: null }),
      }),
    );
    fireEvent.click(await screen.findByRole('button', { name: 'Load more' }));
    expect(await screen.findByRole('link', { name: 'msg_2' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'msg_1' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument();
    expect(transport.calls.map((c) => c.path)).toContain('/portal/v1/messages?limit=25&cursor=c1');
  });

  it('opens a message from the list', async () => {
    renderApp(
      '/messages',
      signedIn('owner', {
        [FIRST]: () => json(200, { data: [messageFixture({ id: 'msg_1' })], next_cursor: null }),
        'GET /portal/v1/messages/msg_1': () =>
          json(200, { ...messageFixture({ id: 'msg_1' }), status_history: [] }),
      }),
    );
    fireEvent.click(await screen.findByRole('link', { name: 'msg_1' }));
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Message details' }),
    ).toBeInTheDocument();
    expect(window.location.pathname).toBe('/messages/msg_1');
  });

  it('filters by status, explains an empty result and returns to all messages', async () => {
    const { transport } = renderApp(
      '/messages',
      signedIn('owner', {
        [FIRST]: () => json(200, { data: [messageFixture()], next_cursor: null }),
        'GET /portal/v1/messages?limit=25&status=failed': () =>
          json(200, { data: [], next_cursor: null }),
      }),
    );
    await screen.findByRole('table');
    fireEvent.change(screen.getByLabelText('Status'), { target: { value: 'failed' } });
    const empty = await screen.findByRole('region', { name: 'No messages with this status' });
    expect(empty).toHaveTextContent('“Failed”');
    expect(transport.calls.map((c) => c.path)).toContain(
      '/portal/v1/messages?limit=25&status=failed',
    );
    fireEvent.click(within(empty).getByRole('button', { name: 'Show all messages' }));
    expect(await screen.findByRole('table')).toBeInTheDocument();
    expect(screen.getByLabelText('Status')).toHaveValue('');
  });

  it('shows the count of loaded messages and offers sending only to roles that may send', async () => {
    renderApp(
      '/messages',
      signedIn('viewer', {
        [FIRST]: () => json(200, { data: [messageFixture()], next_cursor: null }),
      }),
    );
    await screen.findByRole('table');
    expect(screen.getByText(/Showing/)).toHaveTextContent('Showing 1 message');
    expect(screen.queryByRole('link', { name: 'Send SMS' })).not.toBeInTheDocument();
  });
});
