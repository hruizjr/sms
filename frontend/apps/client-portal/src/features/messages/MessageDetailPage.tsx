import { Button, Card } from '@sms/ui';
import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { Skeleton } from '../../components/feedback/Skeleton';
import { ApiError, NetworkError, type MessageDetail } from '../../lib/api/client';
import { Link } from '../../lib/router';
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
    <div className="space-y-6">
      <p>
        <Link to="/messages" className="text-sm font-medium text-primary underline">
          ← All messages
        </Link>
      </p>
      <h1 className="text-2xl font-semibold sm:text-3xl">Message details</h1>

      {justSent && state.status === 'loaded' && (
        <Alert tone="success" title="Message accepted">
          The platform accepted your message and is processing it. Its status below updates
          automatically.
        </Alert>
      )}

      {state.status === 'loading' && <Skeleton lines={5} label="Loading message…" />}

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
  );
}

function Details({ message }: { message: MessageDetail }) {
  const info = statusInfo(message.status);
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <Card heading="Summary">
        <dl className="grid grid-cols-1 gap-3 text-text sm:grid-cols-3">
          <Row term="Status">
            <StatusBadge status={message.status} />
            <p className="mt-2 text-sm text-text-muted">{info.explanation}</p>
            {message.status_reason !== null && (
              <p className="mt-1 text-sm text-text-muted">
                Reason: {message.status_reason.description}
              </p>
            )}
            {message.status_category === 'in_progress' && (
              <p className="mt-1 text-sm text-text-muted">Updating automatically…</p>
            )}
          </Row>
          <Row term="Message ID">
            <span className="font-mono break-all">{message.id}</span>
          </Row>
          <Row term="Recipient">{message.to}</Row>
          <Row term="Sender ID">{message.sender_id}</Row>
          <Row term="Encoding">{encodingLabel(message.encoding)}</Row>
          <Row term="Segments">{message.segment_count}</Row>
          <Row term="Submitted">{formatDateTime(message.accepted_at)}</Row>
          <Row term="Status updated">{formatDateTime(message.status_updated_at)}</Row>
        </dl>
        <p className="mt-4 text-sm text-text-muted">
          The message text is not shown: the platform does not display message content after
          submission.
        </p>
      </Card>
      <Card heading="Status history">
        <ol className="space-y-4">
          {message.status_history.map((change, i) => (
            <li key={`${change.status}-${change.occurred_at}-${String(i)}`} className="space-y-1">
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge status={change.status} />
                <span className="text-sm text-text-muted">
                  {formatDateTime(change.occurred_at)}
                </span>
              </div>
              {change.status_reason !== null && (
                <p className="text-sm text-text-muted">{change.status_reason.description}</p>
              )}
            </li>
          ))}
        </ol>
      </Card>
    </div>
  );
}

function Row({ term, children }: { term: string; children: ReactNode }) {
  return (
    <div className="sm:col-span-3 sm:grid sm:grid-cols-3 sm:gap-4">
      <dt className="text-sm font-medium text-text-muted">{term}</dt>
      <dd className="sm:col-span-2">{children}</dd>
    </div>
  );
}
