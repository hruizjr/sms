import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  CSRF,
  detailFixture,
  json,
  messageFixture,
  problem,
  signedIn,
  type FakeRequest,
  type Handler,
} from '../../test/fake-api';
import { renderApp } from '../../test/render';

const SENDER_IDS = 'GET /portal/v1/sender-ids';
const ESTIMATE = 'POST /portal/v1/message-estimates';
const CREATE = 'POST /portal/v1/messages';

function keys() {
  let n = 0;
  return () => {
    n += 1;
    return `key-${String(n)}`;
  };
}

function routes(extra: Record<string, Handler> = {}): Record<string, Handler> {
  return {
    [SENDER_IDS]: () => json(200, { data: ['EXAMPLECO', 'OTHERCO'] }),
    [ESTIMATE]: (r) => {
      const body = (r.body as { body: string }).body;
      return json(200, {
        encoding: /^[\x20-\x7e]*$/.test(body) ? 'gsm7' : 'ucs2',
        segment_count: body.length > 160 ? 2 : 1,
        characters: Array.from(body).length,
        max_characters: 1600,
      });
    },
    'GET /portal/v1/messages/msg_1': () =>
      json(200, detailFixture({ ...messageFixture({ id: 'msg_1' }) })),
    ...extra,
  };
}

async function fill(to: string, body: string, sender = 'EXAMPLECO') {
  fireEvent.change(await screen.findByLabelText('Recipient mobile number'), {
    target: { value: to },
  });
  fireEvent.change(screen.getByLabelText('Sender ID'), { target: { value: sender } });
  fireEvent.change(screen.getByLabelText('Message'), { target: { value: body } });
}

function creates(calls: FakeRequest[]) {
  return calls.filter((c) => c.method === 'POST' && c.path === '/portal/v1/messages');
}

describe('compose page', () => {
  it('lists only approved sender IDs and shows the platform’s segment estimate', async () => {
    renderApp('/messages/new', signedIn('owner', routes()));
    const select = await screen.findByLabelText('Sender ID');
    expect([...(select as HTMLSelectElement).options].map((o) => o.value)).toEqual([
      '',
      'EXAMPLECO',
      'OTHERCO',
    ]);
    fireEvent.change(screen.getByLabelText('Message'), { target: { value: 'a'.repeat(161) } });
    expect(screen.getByTestId('message-counter')).toHaveTextContent('161 / 1600 characters');
    await waitFor(() => {
      expect(screen.getByTestId('message-counter')).toHaveTextContent('GSM-7 · 2 segments');
    });
    fireEvent.change(screen.getByLabelText('Message'), { target: { value: 'Salamat 😊' } });
    await waitFor(() => {
      expect(screen.getByTestId('message-counter')).toHaveTextContent(
        'UCS-2 (Unicode) · 1 segment',
      );
    });
  });

  it('validates the recipient, sender ID and text before review', async () => {
    const { transport } = renderApp('/messages/new', signedIn('owner', routes()));
    fireEvent.click(await screen.findByRole('button', { name: 'Review message' }));
    expect(screen.getByText('Enter the recipient’s mobile number.')).toBeInTheDocument();
    expect(screen.getByText('Choose a sender ID.')).toBeInTheDocument();
    expect(screen.getByText('Enter the message text.')).toBeInTheDocument();
    await fill('+14155550100', 'x'.repeat(1601));
    fireEvent.click(screen.getByRole('button', { name: 'Review message' }));
    expect(screen.getByText(/Enter a Philippine mobile number/)).toBeInTheDocument();
    expect(screen.getByText(/maximum is 1600/)).toBeInTheDocument();
    expect(creates(transport.calls)).toHaveLength(0);
  });

  it('reviews (one recipient, segments, no cost) and sends once, with CSRF and an idempotency key', async () => {
    const { transport } = renderApp(
      '/messages/new',
      signedIn(
        'owner',
        routes({
          [CREATE]: () =>
            json(
              202,
              messageFixture({ id: 'msg_1', status: 'accepted', status_category: 'in_progress' }),
            ),
        }),
      ),
      keys(),
    );
    await fill('+63 900 000 0001', 'Your code is 000000.');
    fireEvent.click(screen.getByRole('button', { name: 'Review message' }));
    const review = await screen.findByRole('region', { name: 'Review and send' });
    expect(review).toHaveTextContent('Recipients1 (+639000000001)');
    expect(review).toHaveTextContent('Not available: billing is not enabled yet.');
    await waitFor(() => {
      expect(review).toHaveTextContent('1 · GSM-7');
    });

    const send = screen.getByRole('button', { name: 'Send message' });
    fireEvent.click(send);
    fireEvent.click(send); // a double click
    expect(await screen.findByRole('heading', { name: 'Message details' })).toBeInTheDocument();
    expect(window.location.pathname).toBe('/messages/msg_1');
    expect(screen.getByText('Message accepted')).toBeInTheDocument();

    const sent = creates(transport.calls);
    expect(sent).toHaveLength(1);
    expect(sent[0]?.headers).toMatchObject({ 'X-CSRF-Token': CSRF, 'Idempotency-Key': 'key-1' });
    expect(sent[0]?.body).toEqual({
      to: '+639000000001',
      sender_id: 'EXAMPLECO',
      body: 'Your code is 000000.',
    });
  });

  it('retries with the same idempotency key after a lost response, and uses a new key after an edit', async () => {
    let attempt = 0;
    const { transport } = renderApp(
      '/messages/new',
      signedIn(
        'owner',
        routes({
          [CREATE]: () => {
            attempt += 1;
            if (attempt <= 2) throw new TypeError('Failed to fetch');
            return json(202, messageFixture({ id: 'msg_1' }));
          },
        }),
      ),
      keys(),
    );
    await fill('+639000000001', 'Hello');
    fireEvent.click(screen.getByRole('button', { name: 'Review message' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Send message' }));
    expect(await screen.findByRole('alert')).toHaveTextContent(/cannot create a duplicate/);

    // Same content: same key.
    fireEvent.click(screen.getByRole('button', { name: 'Send message' }));
    await waitFor(() => {
      expect(creates(transport.calls)).toHaveLength(2);
    });
    // Edit the text: a new intended send, so a new key.
    await screen.findByRole('alert');
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    fireEvent.change(screen.getByLabelText('Message'), { target: { value: 'Hello again' } });
    fireEvent.click(screen.getByRole('button', { name: 'Review message' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Send message' }));
    expect(await screen.findByRole('heading', { name: 'Message details' })).toBeInTheDocument();

    expect(creates(transport.calls).map((c) => c.headers['Idempotency-Key'])).toEqual([
      'key-1',
      'key-1',
      'key-2',
    ]);
  });

  it('shows the server’s reason when the message is not acceptable, and maps field errors', async () => {
    let response = problem(422, 'sender-id-not-approved', { title: 'Sender ID not approved' });
    renderApp('/messages/new', signedIn('owner', routes({ [CREATE]: () => response })), keys());
    await fill('+639000000001', 'Hello');
    fireEvent.click(screen.getByRole('button', { name: 'Review message' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Send message' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Sender ID not approved');
    // Back to editing so the user can fix it.
    expect(screen.getByRole('button', { name: 'Review message' })).toBeInTheDocument();

    response = problem(400, 'validation-error', {
      errors: [
        {
          pointer: '/to',
          code: 'invalid_phone_number',
          detail: 'Server says the number is invalid.',
        },
      ],
    });
    fireEvent.click(screen.getByRole('button', { name: 'Review message' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Send message' }));
    expect(await screen.findByText('Server says the number is invalid.')).toBeInTheDocument();
    expect(screen.getByLabelText('Recipient mobile number')).toHaveAttribute(
      'aria-invalid',
      'true',
    );
  });

  it('tells a viewer that their role cannot send, without showing the form', async () => {
    const { transport } = renderApp('/messages/new', signedIn('viewer', routes()));
    expect(await screen.findByText('Your role cannot send messages')).toBeInTheDocument();
    expect(screen.queryByLabelText('Message')).not.toBeInTheDocument();
    expect(transport.calls.some((c) => c.path.includes('sender-ids'))).toBe(false);
  });

  it('explains when there is no approved sender ID, and when sender IDs cannot be loaded', async () => {
    renderApp('/messages/new', signedIn('owner', { [SENDER_IDS]: () => json(200, { data: [] }) }));
    expect(await screen.findByText('No approved sender ID')).toBeInTheDocument();
  });

  it('shows an error with a retry when sender IDs cannot be loaded', async () => {
    let fail = true;
    renderApp(
      '/messages/new',
      signedIn(
        'owner',
        routes({
          [SENDER_IDS]: () =>
            fail ? problem(500, 'internal') : json(200, { data: ['EXAMPLECO'] }),
        }),
      ),
    );
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load your sender IDs');
    fail = false;
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    // A single sender ID is preselected.
    expect(await screen.findByLabelText('Sender ID')).toHaveValue('EXAMPLECO');
  });

  it('marks the current step and shows the message text in the review', async () => {
    renderApp('/messages/new', signedIn('owner', routes()));
    const steps = within(await screen.findByRole('list', { name: 'Progress' }));
    expect(steps.getByText('Compose').closest('li')).toHaveAttribute('aria-current', 'step');
    await fill('+639000000001', 'Line one\nLine two');
    fireEvent.click(screen.getByRole('button', { name: 'Review message' }));
    const review = await screen.findByRole('region', { name: 'Review and send' });
    // The text is shown as typed, line breaks included.
    expect(within(review).getByText(/Line one/).textContent).toBe('Line one\nLine two');
    expect(steps.getByText('Review and send').closest('li')).toHaveAttribute(
      'aria-current',
      'step',
    );
    expect(steps.getByText('Compose').closest('li')).toHaveTextContent('(completed)');
    // The fields are locked during the review; Edit unlocks them.
    expect(screen.getByLabelText('Message')).toBeDisabled();
    fireEvent.click(within(review).getByRole('button', { name: 'Edit' }));
    expect(screen.getByLabelText('Message')).toBeEnabled();
    expect(steps.getByText('Compose').closest('li')).toHaveAttribute('aria-current', 'step');
  });
});
