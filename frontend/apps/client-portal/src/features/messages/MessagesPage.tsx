import { Button, Card } from '@sms/ui';
import { useCallback, useEffect, useState } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { Skeleton } from '../../components/feedback/Skeleton';
import { NetworkError, type Message } from '../../lib/api/client';
import { Link } from '../../lib/router';
import { hasPermission, useSession, useSignedInSession } from '../auth/session';
import { StatusBadge } from './StatusBadge';
import { formatDateTime } from './status';

const PAGE_SIZE = 25;

type ListState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'loaded'; messages: Message[]; nextCursor: string | null };

function loadError(error: unknown): string {
  return error instanceof NetworkError
    ? 'The server could not be reached. Check your connection and try again.'
    : 'Your messages could not be loaded. Try again.';
}

/** The organization's messages, newest first, with loading, empty, error and success states. */
export function MessagesPage() {
  const { api } = useSession();
  const session = useSignedInSession();
  const canSend = hasPermission(session, 'messages:send');
  const [state, setState] = useState<ListState>({ status: 'loading' });
  const [loadingMore, setLoadingMore] = useState(false);
  const [moreError, setMoreError] = useState<string | undefined>(undefined);
  const [reload, setReload] = useState(0);

  useEffect(() => {
    document.title = 'Messages · Client Portal';
  }, []);

  useEffect(() => {
    let active = true;
    api.listMessages({ limit: PAGE_SIZE }).then(
      (page) => {
        if (active) {
          setState({ status: 'loaded', messages: page.data, nextCursor: page.next_cursor });
        }
      },
      (error: unknown) => {
        if (active) setState({ status: 'error', message: loadError(error) });
      },
    );
    return () => {
      active = false;
    };
  }, [api, reload]);

  const refresh = useCallback(() => {
    setState({ status: 'loading' });
    setMoreError(undefined);
    setReload((n) => n + 1);
  }, []);

  async function loadMore(cursor: string) {
    setLoadingMore(true);
    setMoreError(undefined);
    try {
      const page = await api.listMessages({ limit: PAGE_SIZE, cursor });
      setState((current) =>
        current.status === 'loaded'
          ? {
              status: 'loaded',
              messages: [...current.messages, ...page.data],
              nextCursor: page.next_cursor,
            }
          : current,
      );
    } catch (error: unknown) {
      setMoreError(loadError(error));
    } finally {
      setLoadingMore(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold sm:text-3xl">Messages</h1>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            onClick={refresh}
            disabled={state.status === 'loading'}
            aria-label="Refresh messages"
          >
            Refresh
          </Button>
          {canSend && (
            <Link
              to="/messages/new"
              className="inline-flex min-h-10 items-center rounded-control bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary-hover"
            >
              Send SMS
            </Link>
          )}
        </div>
      </div>

      {state.status === 'loading' && <Skeleton lines={4} label="Loading messages…" />}

      {state.status === 'error' && (
        <Alert
          tone="error"
          title="Could not load messages"
          action={
            <Button variant="secondary" size="sm" onClick={refresh}>
              Try again
            </Button>
          }
        >
          {state.message}
        </Alert>
      )}

      {state.status === 'loaded' && state.messages.length === 0 && (
        <Card heading="No messages yet">
          <p>Messages your organization sends, from this portal or through the API, appear here.</p>
          {canSend && (
            <p className="mt-3">
              <Link to="/messages/new" className="font-medium text-primary underline">
                Send your first SMS
              </Link>
            </p>
          )}
        </Card>
      )}

      {state.status === 'loaded' && state.messages.length > 0 && (
        <>
          <MessageTable messages={state.messages} />
          {moreError !== undefined && <Alert tone="error" title={moreError} />}
          {state.nextCursor !== null && (
            <div>
              <Button
                variant="secondary"
                loading={loadingMore}
                onClick={() => {
                  if (state.nextCursor !== null) void loadMore(state.nextCursor);
                }}
              >
                Load more
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

/**
 * A table on wide screens; on small screens each row becomes a stacked card (no page-level
 * horizontal scrolling, AGENTS.md §8.8).
 */
function MessageTable({ messages }: { messages: Message[] }) {
  return (
    <div className="rounded-card border border-border bg-surface">
      <table className="w-full text-left text-sm">
        <caption className="sr-only">Messages, newest first</caption>
        <thead className="hidden border-b border-border text-text-muted md:table-header-group">
          <tr>
            <th scope="col" className="px-4 py-3 font-medium">
              Recipient
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              Sender ID
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              Message ID
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              Submitted
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              Status
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {messages.map((m) => (
            <tr key={m.id} className="block p-4 md:table-row md:p-0">
              <Cell label="Recipient">{m.to}</Cell>
              <Cell label="Sender ID">{m.sender_id}</Cell>
              <Cell label="Message ID">
                <Link
                  to={`/messages/${m.id}`}
                  className="font-mono break-all text-primary underline"
                >
                  {m.id}
                </Link>
              </Cell>
              <Cell label="Submitted">{formatDateTime(m.accepted_at)}</Cell>
              <Cell label="Status">
                <StatusBadge status={m.status} />
              </Cell>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Cell({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <td className="flex justify-between gap-4 py-1 md:table-cell md:px-4 md:py-3">
      <span className="font-medium text-text-muted md:hidden">{label}</span>
      <span className="text-right md:text-left">{children}</span>
    </td>
  );
}
