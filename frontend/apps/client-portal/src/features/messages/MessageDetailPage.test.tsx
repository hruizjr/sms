import { act, fireEvent, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { detailFixture, json, problem, signedIn } from '../../test/fake-api';
import { renderApp } from '../../test/render';
import { REFRESH_INTERVAL_MS } from './MessageDetailPage';

const PATH = '/messages/msg_06H1WH346HSF13TPPJR44P43PW';
const API = `GET /portal/v1${PATH}`;

afterEach(() => {
  vi.useRealTimers();
});

describe('message detail page', () => {
  it('shows the message, its status explanation and its status history', async () => {
    renderApp(PATH, signedIn('viewer', { [API]: () => json(200, detailFixture()) }));
    expect(await screen.findByText('Loading message…')).toBeInTheDocument();
    const summary = await screen.findByRole('region', { name: 'Summary' });
    expect(summary).toHaveTextContent('msg_06H1WH346HSF13TPPJR44P43PW');
    expect(summary).toHaveTextContent('+639000000001');
    expect(summary).toHaveTextContent('GSM-7');
    expect(summary).toHaveTextContent('The provider reported delivery to the handset.');
    expect(summary).toHaveTextContent('message text is not shown');
    const history = within(screen.getByRole('region', { name: 'Status history' })).getAllByRole(
      'listitem',
    );
    expect(history.map((li) => li.textContent)).toEqual([
      expect.stringContaining('Accepted'),
      expect.stringContaining('Provider accepted'),
      expect.stringContaining('Delivered'),
    ]);
  });

  it('explains an uncertain outcome without claiming non-delivery', async () => {
    renderApp(
      PATH,
      signedIn('owner', {
        [API]: () =>
          json(
            200,
            detailFixture({
              status: 'submission_uncertain',
              status_category: 'uncertain',
              status_reason: {
                code: 'submission_outcome_unknown',
                description: 'The provider did not answer.',
              },
            }),
          ),
      }),
    );
    const summary = await screen.findByRole('region', { name: 'Summary' });
    expect(summary).toHaveTextContent('may already have been delivered');
    expect(summary).toHaveTextContent('not resent automatically');
    expect(summary).toHaveTextContent('Reason: The provider did not answer.');
  });

  it('shows not found for a missing (or another organization’s) message', async () => {
    renderApp(PATH, signedIn('owner', { [API]: () => problem(404, 'not-found') }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Message not found');
  });

  it('shows an error state with a retry', async () => {
    renderApp(PATH, signedIn('owner', { [API]: () => problem(500, 'internal') }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load the message');
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('follows an in-progress message until it is final', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    let delivered = false;
    const { transport } = renderApp(
      PATH,
      signedIn('owner', {
        [API]: () =>
          json(
            200,
            delivered
              ? detailFixture()
              : detailFixture({ status: 'provider_accepted', status_category: 'in_progress' }),
          ),
      }),
    );
    expect(await screen.findByText('Updating automatically…')).toBeInTheDocument();
    delivered = true;
    await act(async () => {
      await vi.advanceTimersByTimeAsync(REFRESH_INTERVAL_MS);
    });
    expect(
      await screen.findByText('The provider reported delivery to the handset.'),
    ).toBeInTheDocument();
    const reads = transport.calls.filter((c) => c.path.endsWith('PW')).length;
    await act(async () => {
      await vi.advanceTimersByTimeAsync(REFRESH_INTERVAL_MS * 3);
    });
    expect(transport.calls.filter((c) => c.path.endsWith('PW')).length).toBe(reads);
  });

  it('shows a breadcrumb back to the messages and marks the current status', async () => {
    renderApp(PATH, signedIn('viewer', { [API]: () => json(200, detailFixture()) }));
    await screen.findByRole('region', { name: 'Summary' });
    const breadcrumb = within(screen.getByRole('navigation', { name: 'Breadcrumb' }));
    expect(breadcrumb.getByRole('link', { name: 'Messages' })).toHaveAttribute('href', '/messages');
    expect(breadcrumb.getByText('Message details')).toHaveAttribute('aria-current', 'page');
    const history = within(screen.getByRole('region', { name: 'Status history' })).getAllByRole(
      'listitem',
    );
    expect(history.at(-1)).toHaveTextContent('DeliveredCurrent');
    expect(history[0]).not.toHaveTextContent('Current');
  });

  it('copies the message ID', async () => {
    const writeText = vi.fn(() => Promise.resolve());
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
    try {
      renderApp(PATH, signedIn('viewer', { [API]: () => json(200, detailFixture()) }));
      fireEvent.click(await screen.findByRole('button', { name: 'Copy message ID' }));
      expect(await screen.findByText('Copied')).toBeInTheDocument();
      expect(writeText).toHaveBeenCalledWith('msg_06H1WH346HSF13TPPJR44P43PW');
    } finally {
      Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined });
    }
  });

  it('says when no status change is recorded yet', async () => {
    renderApp(
      PATH,
      signedIn('owner', { [API]: () => json(200, detailFixture({ status_history: [] })) }),
    );
    const history = await screen.findByRole('region', { name: 'Status history' });
    expect(history).toHaveTextContent('No status changes recorded yet.');
  });
});
