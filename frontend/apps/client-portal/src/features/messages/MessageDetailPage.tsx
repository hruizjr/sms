import { Button, Card, Icon } from '@sms/ui';
import { useCallback, useEffect, useState } from 'react';
import { CopyButton } from '../../components/actions/CopyButton';
import { DescriptionItem, DescriptionList } from '../../components/data/DescriptionList';
import { Alert } from '../../components/feedback/Alert';
import { Skeleton } from '../../components/feedback/Skeleton';
import { ApiError, NetworkError, type MessageDetail } from '../../lib/api/client';
import { PageHeader } from '../../components/layout/PageHeader';
import { useSession } from '../auth/session';
import { StatusBadge } from './StatusBadge';
import { encodingLabel, formatDateTime, statusInfo, type StatusTone } from './status';

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
        breadcrumbs={[{ to: '/messages', label: 'Messages' }]}
        title="Message details"
        meta={state.status === 'loaded' ? <StatusBadge status={state.message.status} /> : undefined}
        description={
          <span className="flex min-w-0 flex-wrap items-center gap-x-1 font-mono text-xs break-all">
            {id}
            {state.status === 'loaded' && <CopyButton value={id} label="Copy message ID" />}
          </span>
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

const PANEL_TONES: Record<StatusTone, { panel: string; icon: string }> = {
  progress: { panel: 'border-info/30 bg-info-subtle', icon: 'text-info' },
  success: { panel: 'border-success/30 bg-success-subtle', icon: 'text-success' },
  warning: { panel: 'border-warning/30 bg-warning-subtle', icon: 'text-warning' },
  danger: { panel: 'border-danger/30 bg-danger-subtle', icon: 'text-danger' },
};

const DOT_TONES: Record<StatusTone, string> = {
  progress: 'border-info/30 bg-info-subtle text-info',
  success: 'border-success/30 bg-success-subtle text-success',
  warning: 'border-warning/30 bg-warning-subtle text-warning',
  danger: 'border-danger/30 bg-danger-subtle text-danger',
};

function Details({ message }: { message: MessageDetail }) {
  const info = statusInfo(message.status);
  const tone = PANEL_TONES[info.tone];
  const history = message.status_history;
  return (
    <div className="grid items-start gap-6 lg:grid-cols-3">
      <Card
        heading="Summary"
        className="lg:col-span-2"
        footer="The message text is not shown: the platform does not display message content after submission."
      >
        <div className={`mb-5 flex gap-3 rounded-control border p-4 ${tone.panel}`}>
          <Icon name={info.icon} className={`mt-0.5 size-5 ${tone.icon}`} />
          <div className="min-w-0 space-y-1 text-sm text-text">
            <p className="font-semibold">{info.label}</p>
            <p>{info.explanation}</p>
            {message.status_reason !== null && (
              <p>
                <span className="font-medium">Reason:</span> {message.status_reason.description}
              </p>
            )}
            {message.status_category === 'in_progress' && (
              <p className="flex items-center gap-1.5 text-text-muted">
                <span aria-hidden="true" className="size-2 animate-pulse rounded-full bg-info" />
                Updating automatically…
              </p>
            )}
          </div>
        </div>
        <DescriptionList>
          <DescriptionItem term="Recipient">
            <span className="font-medium tabular-nums">{message.to}</span>
          </DescriptionItem>
          <DescriptionItem term="Sender ID">{message.sender_id}</DescriptionItem>
          <DescriptionItem term="Message ID">
            <span className="font-mono text-xs break-all">{message.id}</span>
          </DescriptionItem>
          <DescriptionItem term="Encoding">{encodingLabel(message.encoding)}</DescriptionItem>
          <DescriptionItem term="Segments">
            <span className="tabular-nums">{message.segment_count}</span>
          </DescriptionItem>
          <DescriptionItem term="Submitted">
            <span className="tabular-nums">{formatDateTime(message.accepted_at)}</span>
          </DescriptionItem>
          <DescriptionItem term="Status updated">
            <span className="tabular-nums">{formatDateTime(message.status_updated_at)}</span>
          </DescriptionItem>
        </DescriptionList>
      </Card>
      <Card heading="Status history" description="Oldest first, in the order the changes happened">
        {history.length === 0 ? (
          <p className="text-text-muted">No status changes recorded yet.</p>
        ) : (
          <ol>
            {history.map((change, i) => {
              const changeInfo = statusInfo(change.status);
              return (
                <li
                  key={`${change.status}-${change.occurred_at}-${String(i)}`}
                  className="relative flex gap-3 pb-5 last:pb-0"
                >
                  {i < history.length - 1 && (
                    <span
                      aria-hidden="true"
                      className="absolute top-8 bottom-1 left-3.5 w-px bg-border"
                    />
                  )}
                  <span
                    aria-hidden="true"
                    className={`relative flex size-7 shrink-0 items-center justify-center rounded-full border ${DOT_TONES[changeInfo.tone]}`}
                  >
                    <Icon name={changeInfo.icon} className="size-3.5" />
                  </span>
                  <div className="min-w-0 pt-0.5">
                    <p className="text-sm font-medium text-text">
                      {changeInfo.label}
                      {i === history.length - 1 && (
                        <span className="ml-2 text-xs font-normal text-text-muted">Current</span>
                      )}
                    </p>
                    <p className="text-xs text-text-muted tabular-nums">
                      {formatDateTime(change.occurred_at)}
                    </p>
                    {change.status_reason !== null && (
                      <p className="mt-1 text-sm text-text-muted">
                        {change.status_reason.description}
                      </p>
                    )}
                  </div>
                </li>
              );
            })}
          </ol>
        )}
      </Card>
    </div>
  );
}
