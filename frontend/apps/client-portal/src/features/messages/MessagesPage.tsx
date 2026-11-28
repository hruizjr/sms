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

  const loaded = state.status === 'loaded' ? state : undefined;
  return (
    <div>
      <PageHeader
        title="Messages"
        description="SMS sent by your organization from this portal and through the API, newest first."
        actions={
          canSend ? (
            <ButtonLink to="/messages/new" icon="plus">
              Send SMS
            </ButtonLink>
          ) : undefined
        }
      />

      <Card
        padding="none"
        footer={
          loaded !== undefined && loaded.messages.length > 0 ? (
            <div className="flex flex-wrap items-center justify-between gap-3">
              <p aria-live="polite">
                Showing{' '}
                <span className="font-medium text-text tabular-nums">{loaded.messages.length}</span>{' '}
                {loaded.messages.length === 1 ? 'message' : 'messages'}
                {loaded.nextCursor !== null && ' · more available'}
              </p>
              {loaded.nextCursor !== null && (
                <Button
                  variant="secondary"
                  size="sm"
                  loading={loadingMore}
                  onClick={() => {
                    if (loaded.nextCursor !== null) void loadMore(loaded.nextCursor);
                  }}
                >
                  Load more
                </Button>
              )}
            </div>
          ) : undefined
        }
      >
        <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-3 sm:px-5">
          <div className="flex min-w-0 items-center gap-2.5">
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
              className="min-h-8 min-w-0 rounded-control border border-border-strong bg-surface px-2.5 text-sm text-text shadow-xs hover:border-text-muted"
            >
              <option value="">All statuses</option>
              {STATUS_FILTERS.map((s) => (
                <option key={s} value={s}>
                  {STATUS_INFO[s].label}
                </option>
              ))}
            </select>
          </div>
          <Button
            variant="secondary"
            size="sm"
            onClick={refresh}
            disabled={state.status === 'loading'}
            aria-label="Refresh messages"
          >
            <Icon name="refresh" />
            <span className="hidden sm:inline">Refresh</span>
          </Button>
        </div>

        {state.status === 'loading' && (
          <Skeleton variant="table" bare lines={6} label="Loading messages…" />
        )}

        {state.status === 'error' && (
          <div className="p-4 sm:p-5">
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
          </div>
        )}

        {loaded?.messages.length === 0 && filter !== '' && (
          <EmptyState
            bare
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

        {loaded?.messages.length === 0 && filter === '' && (
          <EmptyState
            bare
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

        {loaded !== undefined && loaded.messages.length > 0 && (
          <>
            <MessageTable messages={loaded.messages} />
            {moreError !== undefined && (
              <div className="border-t border-border p-4 sm:px-5">
                <Alert tone="error" title={moreError} />
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  );
}

const th = 'px-4 py-2.5 text-xs font-medium text-text-muted first:pl-5 last:pr-5';
const td = 'md:px-4 md:py-3 md:first:pl-5 md:last:pr-5';

/**
 * A table from 768 px; below that each row becomes a compact card (no page-level horizontal
 * scrolling, AGENTS.md §8.8). Segments and the last status update are shown from 1280 px. The whole
 * row opens the message: the message ID is the row's link, and its click area covers the row.
 */
function MessageTable({ messages }: { messages: Message[] }) {
  return (
    <table className="w-full text-left text-sm">
      <caption className="sr-only">Messages, newest first</caption>
      <thead className="hidden border-b border-border bg-surface-subtle md:table-header-group">
        <tr>
          <th scope="col" className={th}>
            Recipient
          </th>
          <th scope="col" className={th}>
            Status
          </th>
          <th scope="col" className={th}>
            Sender ID
          </th>
          <th scope="col" className={`${th} hidden text-right xl:table-cell`}>
            Segments
          </th>
          <th scope="col" className={th}>
            Submitted
          </th>
          <th scope="col" className={`${th} hidden xl:table-cell`}>
            Last update
          </th>
        </tr>
      </thead>
      <tbody>
        {messages.map((m) => (
          <tr
            key={m.id}
            className="relative grid grid-cols-2 gap-x-3 gap-y-1 border-t border-border px-4 py-3 transition-colors first:border-t-0 hover:bg-surface-subtle md:table-row md:p-0"
          >
            <td className={`min-w-0 ${td}`}>
              <p className="font-medium text-text tabular-nums">{m.to}</p>
              <Link
                to={`/messages/${m.id}`}
                className="block truncate font-mono text-xs text-text-muted after:absolute after:inset-0 hover:text-primary hover:underline md:max-w-56"
              >
                {m.id}
              </Link>
            </td>
            <td className={`self-start justify-self-end ${td}`}>
              <StatusBadge status={m.status} />
            </td>
            <td className={`text-text-muted md:text-text ${td}`}>
              <span className="sr-only md:hidden">Sender ID </span>
              {m.sender_id}
            </td>
            <td className={`hidden text-right tabular-nums xl:table-cell ${td}`}>
              {m.segment_count}
            </td>
            <td
              className={`justify-self-end text-xs whitespace-nowrap text-text-muted tabular-nums md:text-sm ${td}`}
            >
              {formatDateTime(m.accepted_at, 'short')}
            </td>
            <td
              className={`hidden text-sm whitespace-nowrap text-text-muted tabular-nums xl:table-cell ${td}`}
            >
              {formatDateTime(m.status_updated_at, 'short')}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
