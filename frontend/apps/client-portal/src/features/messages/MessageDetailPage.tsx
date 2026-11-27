import { Button, Card } from '@sms/ui';
import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { Skeleton } from '../../components/feedback/Skeleton';
import { ApiError, NetworkError, type MessageDetail } from '../../lib/api/client';
import { PageHeader } from '../../components/layout/PageHeader';
import { useSession } from '../auth/session';
import { StatusBadge } from './StatusBadge';
import { encodingLabel, formatDateTime, statusInfo } from './status';

/** While a message is in progress, its status is refreshed at this interval. */
export const REFRESH_INTERVAL_MS = 3000;

type DetailState =
  | { status: 'loading' }
  | { status: 'not-found' }
  | { status: 'error'; message: string }
  | { status: 'loaded'; message: MessageDetail };

export function MessageDetailPage({ id, justSent = false }: { id: string; justSent?: boolean }) {
  const { api } = useSession();
  const [state, setState] = useState<DetailState>({ status: 'loading' });
  const [reload, setReload] = useState(0);

  useEffect(() => {
    document.title = `Message ${id} · Client Portal`;
  }, [id]);

  useEffect(() => {
    let active = true;
    api.getMessage(id).then(
      (message) => {
        if (active) setState({ status: 'loaded', message });
      },
      (error: unknown) => {
        if (!active) return;
        if (error instanceof ApiError && (error.status === 404 || error.status === 400)) {
          setState({ status: 'not-found' });
        } else {
          setState({
            status: 'error',
            message:
              error instanceof NetworkError
                ? 'The server could not be reached. Check your connection and try again.'
                : 'The message could not be loaded. Try again.',
          });
        }
      },
    );
    return () => {
      active = false;
    };
  }, [api, id, reload]);

  // Follow an in-progress message until it reaches a final or uncertain status.
  const inProgress = state.status === 'loaded' && state.message.status_category === 'in_progress';
  useEffect(() => {
    if (!inProgress) return;
    const timer = window.setTimeout(() => {
      setReload((n) => n + 1);
    }, REFRESH_INTERVAL_MS);
    return () => {
      window.clearTimeout(timer);
    };
  }, [inProgress, reload]);

  const retry = useCallback(() => {
    setState({ status: 'loading' });
    setReload((n) => n + 1);
  }, []);

  return (
    <div>
      <PageHeader
        back={{ to: '/messages', label: 'Messages' }}
        title="Message details"
        meta={state.status === 'loaded' ? <StatusBadge status={state.message.status} /> : undefined}
        description={
          state.status === 'loaded' ? (
            <span className="font-mono text-xs break-all">{state.message.id}</span>
          ) : undefined
        }
      />

      <div className="space-y-6">
        {justSent && state.status === 'loaded' && (
          <Alert tone="success" title="Message accepted">
            The platform accepted your message and is processing it. Its status below updates
            automatically.
          </Alert>
        )}

        {state.status === 'loading' && <Skeleton lines={6} label="Loading message…" />}

        {state.status === 'not-found' && (
          <Alert tone="error" title="Message not found">
            No message with this ID exists in your organization.
          </Alert>
        )}

        {state.status === 'error' && (
          <Alert
            tone="error"
            title="Could not load the message"
            action={
              <Button variant="secondary" size="sm" onClick={retry}>
                Try again
              </Button>
            }
          >
            {state.message}
          </Alert>
        )}

        {state.status === 'loaded' && <Details message={state.message} />}
      </div>
    </div>
  );
}

function Details({ message }: { message: MessageDetail }) {
  const info = statusInfo(message.status);
  const history = message.status_history;
  return (
    <div className="grid gap-6 lg:grid-cols-3">
      <Card heading="Summary" className="lg:col-span-2">
        <div className="mb-5 flex gap-3 rounded-control border border-border bg-surface-subtle p-3">
          <StatusBadge status={message.status} />
          <div className="min-w-0 space-y-1 text-sm">
            <p>{info.explanation}</p>
            {message.status_reason !== null && (
              <p className="text-text-muted">Reason: {message.status_reason.description}</p>
            )}
            {message.status_category === 'in_progress' && (
              <p className="flex items-center gap-1.5 text-text-muted">
                <span aria-hidden="true" className="size-2 animate-pulse rounded-full bg-info" />
                Updating automatically…
              </p>
            )}
          </div>
        </div>
        <dl className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
          <Row term="Recipient">
            <span className="tabular-nums">{message.to}</span>
          </Row>
          <Row term="Sender ID">{message.sender_id}</Row>
          <Row term="Message ID" wide>
            <span className="font-mono text-xs break-all">{message.id}</span>
          </Row>
          <Row term="Encoding">{encodingLabel(message.encoding)}</Row>
          <Row term="Segments">{message.segment_count}</Row>
          <Row term="Submitted">{formatDateTime(message.accepted_at)}</Row>
          <Row term="Status updated">{formatDateTime(message.status_updated_at)}</Row>
        </dl>
        <p className="mt-5 border-t border-border pt-4 text-xs text-text-muted">
          The message text is not shown: the platform does not display message content after
          submission.
        </p>
      </Card>
      <Card heading="Status history" description="In the order the changes happened">
        <ol className="space-y-0">
          {history.map((change, i) => (
            <li
              key={`${change.status}-${change.occurred_at}-${String(i)}`}
              className="relative flex gap-3 pb-5 last:pb-0"
            >
              {i < history.length - 1 && (
                <span
                  aria-hidden="true"
                  className="absolute top-5 bottom-0 left-1 w-px bg-border"
                />
              )}
              <span
                aria-hidden="true"
                className={`relative mt-1.5 size-2.5 shrink-0 rounded-full border-2 ${
                  i === history.length - 1
                    ? 'border-primary bg-primary'
                    : 'border-border-strong bg-surface'
                }`}
              />
              <div className="min-w-0 space-y-1">
                <StatusBadge status={change.status} />
                <p className="text-xs text-text-muted">{formatDateTime(change.occurred_at)}</p>
                {change.status_reason !== null && (
                  <p className="text-sm text-text-muted">{change.status_reason.description}</p>
                )}
              </div>
            </li>
          ))}
        </ol>
      </Card>
    </div>
  );
}

function Row({
  term,
  wide = false,
  children,
}: {
  term: string;
  wide?: boolean;
  children: ReactNode;
}) {
  return (
    <div className={wide ? 'sm:col-span-2' : undefined}>
      <dt className="text-xs font-medium text-text-muted">{term}</dt>
      <dd className="mt-1 text-sm text-text">{children}</dd>
    </div>
  );
}
