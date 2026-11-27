import { Button, Card, Icon } from '@sms/ui';
import { useCallback, useEffect, useId, useState } from 'react';
import { EmptyState } from '../../components/feedback/EmptyState';
import { Alert } from '../../components/feedback/Alert';
import { Skeleton } from '../../components/feedback/Skeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { ButtonLink } from '../../components/navigation/ButtonLink';
import { NetworkError, type Message, type MessageStatus } from '../../lib/api/client';
import { Link } from '../../lib/router';
import { hasPermission, useSession, useSignedInSession } from '../auth/session';
import { StatusBadge } from './StatusBadge';
import { STATUS_INFO, formatDateTime } from './status';

const PAGE_SIZE = 25;

type ListState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'loaded'; messages: Message[]; nextCursor: string | null };

const STATUS_FILTERS = Object.keys(STATUS_INFO) as MessageStatus[];

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
  const filterId = useId();
  const [state, setState] = useState<ListState>({ status: 'loading' });
  const [filter, setFilter] = useState<MessageStatus | ''>('');
  const [loadingMore, setLoadingMore] = useState(false);
  const [moreError, setMoreError] = useState<string | undefined>(undefined);
  const [reload, setReload] = useState(0);

  useEffect(() => {
    document.title = 'Messages · Client Portal';
  }, []);

  useEffect(() => {
    let active = true;
    const params = filter === '' ? { limit: PAGE_SIZE } : { limit: PAGE_SIZE, status: filter };
    api.listMessages(params).then(
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
  }, [api, reload, filter]);

  const refresh = useCallback(() => {
    setState({ status: 'loading' });
    setMoreError(undefined);
    setReload((n) => n + 1);
  }, []);

  async function loadMore(cursor: string) {
    setLoadingMore(true);
    setMoreError(undefined);
    try {
      const page = await api.listMessages({
        limit: PAGE_SIZE,
        cursor,
        ...(filter === '' ? {} : { status: filter }),
      });
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
    <div>
      <PageHeader
        title="Messages"
        description="SMS sent by your organization from this portal and through the API, newest first."
        actions={
          <>
            <Button
              variant="secondary"
              onClick={refresh}
              disabled={state.status === 'loading'}
              aria-label="Refresh messages"
            >
              <Icon name="refresh" />
              Refresh
            </Button>
            {canSend && (
              <ButtonLink to="/messages/new" icon="plus">
                Send SMS
              </ButtonLink>
            )}
          </>
        }
      />

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <label htmlFor={filterId} className="text-sm font-medium text-text">
          Status
        </label>
        <select
          id={filterId}
          value={filter}
          onChange={(e) => {
            setFilter(e.target.value as MessageStatus | '');
            setState({ status: 'loading' });
            setMoreError(undefined);
          }}
          className="min-h-9 rounded-control border border-border-strong bg-surface px-3 text-sm text-text shadow-sm hover:border-text-muted"
        >
          <option value="">All statuses</option>
          {STATUS_FILTERS.map((s) => (
            <option key={s} value={s}>
              {STATUS_INFO[s].label}
            </option>
          ))}
        </select>
      </div>

      {state.status === 'loading' && <Skeleton lines={6} label="Loading messages…" />}

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

      {state.status === 'loaded' && state.messages.length === 0 && filter !== '' && (
        <EmptyState
          icon="messages"
          title="No messages with this status"
          action={
            <Button
              variant="secondary"
              onClick={() => {
                setFilter('');
                setState({ status: 'loading' });
              }}
            >
              Show all messages
            </Button>
          }
        >
          No message of your organization currently has the status “{STATUS_INFO[filter].label}”.
        </EmptyState>
      )}

      {state.status === 'loaded' && state.messages.length === 0 && filter === '' && (
        <EmptyState
          icon="messages"
          title="No messages yet"
          action={
            canSend ? (
              <ButtonLink to="/messages/new" icon="plus">
                Send your first SMS
              </ButtonLink>
            ) : undefined
          }
        >
          Messages your organization sends, from this portal or through the API, appear here.
        </EmptyState>
      )}

      {state.status === 'loaded' && state.messages.length > 0 && (
        <div className="space-y-4">
          <MessageTable messages={state.messages} />
          {moreError !== undefined && <Alert tone="error" title={moreError} />}
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-text-muted" aria-live="polite">
              Showing {state.messages.length} {state.messages.length === 1 ? 'message' : 'messages'}
              {state.nextCursor !== null && ' (more available)'}
            </p>
            {state.nextCursor !== null && (
              <Button
                variant="secondary"
                loading={loadingMore}
                onClick={() => {
                  if (state.nextCursor !== null) void loadMore(state.nextCursor);
                }}
              >
                Load more
              </Button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * A table on wide screens (1280 px and up); on smaller screens each row becomes a compact card (no page-level
 * horizontal scrolling, AGENTS.md §8.8). Long identifiers are truncated visually and kept whole in
 * the accessible name and the tooltip.
 */
function MessageTable({ messages }: { messages: Message[] }) {
  return (
    <Card padding="none" className="overflow-hidden">
      <table className="w-full text-left text-sm">
        <caption className="sr-only">Messages, newest first</caption>
        <thead className="hidden border-b border-border bg-surface-subtle text-xs text-text-muted xl:table-header-group">
          <tr>
            <th scope="col" className="px-4 py-2.5 font-medium">
              Recipient
            </th>
            <th scope="col" className="px-4 py-2.5 font-medium">
              Status
            </th>
            <th scope="col" className="px-4 py-2.5 font-medium">
              Sender ID
            </th>
            <th scope="col" className="px-4 py-2.5 font-medium">
              Message ID
            </th>
            <th scope="col" className="px-4 py-2.5 text-right font-medium">
              Submitted
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {messages.map((m) => (
            <tr
              key={m.id}
              className="grid grid-cols-2 gap-x-3 gap-y-1.5 px-4 py-3 transition-colors hover:bg-surface-subtle xl:table-row xl:p-0"
            >
              <td className="font-medium text-text tabular-nums xl:px-4 xl:py-3">{m.to}</td>
              <td className="justify-self-end xl:px-4 xl:py-3">
                <StatusBadge status={m.status} />
              </td>
              <td className="text-text-muted xl:px-4 xl:py-3 xl:text-text">
                <span className="text-xs xl:hidden">Sender </span>
                {m.sender_id}
              </td>
              <td className="col-span-2 row-start-3 min-w-0 xl:px-4 xl:py-3">
                <Link
                  to={`/messages/${m.id}`}
                  className="flex items-center gap-1 font-mono text-xs text-primary hover:underline"
                >
                  <span className="truncate xl:max-w-56" title={m.id}>
                    {m.id}
                  </span>
                  <Icon name="chevronRight" className="size-3.5" />
                </Link>
              </td>
              <td className="justify-self-end text-right text-xs whitespace-nowrap text-text-muted xl:px-4 xl:py-3">
                {formatDateTime(m.accepted_at, 'short')}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  );
}
