import { Badge, Button, Card, Icon } from '@sms/ui';
import { useCallback, useEffect, useId, useRef, useState } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { EmptyState } from '../../components/feedback/EmptyState';
import { Skeleton } from '../../components/feedback/Skeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import {
  NetworkError,
  type AuditAction,
  type AuditEvent,
  type AuditEventQuery,
  type AuditOutcome,
} from '../../lib/api/client';
import { hasPermission, useSession, useSignedInSession } from '../auth/session';
import { formatDateTime } from '../messages/status';
import { AuditEventDialog } from './AuditEventDialog';
import {
  ACTIONS,
  ACTION_LABELS,
  OUTCOMES,
  OUTCOME_INFO,
  actorText,
  outcomeBadge,
  eventSummary,
  eventTitle,
  manilaDayStart,
} from './events';

const PAGE_SIZE = 50;

type Filters = { action: AuditAction | ''; outcome: AuditOutcome | ''; from: string; to: string };
const NO_FILTERS: Filters = { action: '', outcome: '', from: '', to: '' };

type ListState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'loaded'; events: AuditEvent[]; nextCursor: string | null };

function loadError(error: unknown): string {
  return error instanceof NetworkError
    ? 'The server could not be reached. Check your connection and try again.'
    : 'The audit log could not be loaded. Try again.';
}

/** The query for the filters; a date range is whole days in Philippine Time (end date included). */
function toQuery(f: Filters): AuditEventQuery {
  const query: AuditEventQuery = { limit: PAGE_SIZE };
  if (f.action !== '') query.action = f.action;
  if (f.outcome !== '') query.outcome = f.outcome;
  const from = f.from === '' ? undefined : manilaDayStart(f.from);
  const before = f.to === '' ? undefined : manilaDayStart(f.to, 1);
  if (from !== undefined) query.occurredFrom = from;
  if (before !== undefined) query.occurredBefore = before;
  return query;
}

function rangeError(f: Filters): string | undefined {
  return f.from !== '' && f.to !== '' && f.to < f.from
    ? 'The end date must be on or after the start date.'
    : undefined;
}

/**
 * The organization's security audit log (owners only, audit_log:read): newest first, filtered by
 * event, outcome and date, read-only. The server enforces the permission and the organization; the
 * page only hides itself for other roles.
 */
export function AuditLogPage() {
  const session = useSignedInSession();
  useEffect(() => {
    document.title = 'Audit log · Client Portal';
  }, []);

  return (
    <div>
      <PageHeader
        title="Audit log"
        description="Security events of your organization: sign-ins, account locks, API key changes and refused requests. Events are recorded as they happen and cannot be changed or deleted."
      />
      {hasPermission(session, 'audit_log:read') ? (
        <AuditLog />
      ) : (
        <Alert tone="info" title="Only organization owners can view the audit log">
          Ask an owner of your organization’s account if you need information about a security
          event.
        </Alert>
      )}
    </div>
  );
}

const control =
  'min-h-8 w-full min-w-0 rounded-control border border-border-strong bg-surface px-2.5 text-sm text-text shadow-xs hover:border-text-muted aria-invalid:border-danger';

function AuditLog() {
  const { api } = useSession();
  const ids = { action: useId(), outcome: useId(), from: useId(), to: useId(), range: useId() };
  const [filters, setFilters] = useState<Filters>(NO_FILTERS);
  const [state, setState] = useState<ListState>({ status: 'loading' });
  const [reload, setReload] = useState(0);
  const [loadingMore, setLoadingMore] = useState(false);
  const [moreError, setMoreError] = useState<string | undefined>(undefined);
  const [selected, setSelected] = useState<AuditEvent | undefined>(undefined);
  const invalidRange = rangeError(filters);
  const filtered =
    filters.action !== '' || filters.outcome !== '' || filters.from !== '' || filters.to !== '';
  // The active query, so that "Load more" continues the list it belongs to.
  const query = useRef<AuditEventQuery>(toQuery(NO_FILTERS));

  useEffect(() => {
    if (rangeError(filters) !== undefined) return;
    let active = true;
    const q = toQuery(filters);
    query.current = q;
    api.listAuditEvents(q).then(
      (page) => {
        if (active) setState({ status: 'loaded', events: page.data, nextCursor: page.next_cursor });
      },
      (error: unknown) => {
        if (active) setState({ status: 'error', message: loadError(error) });
      },
    );
    return () => {
      active = false;
    };
  }, [api, reload, filters]);

  const refresh = useCallback(() => {
    setState({ status: 'loading' });
    setMoreError(undefined);
    setReload((n) => n + 1);
  }, []);

  function update(change: Partial<Filters>) {
    const next = { ...filters, ...change };
    setFilters(next);
    setMoreError(undefined);
    if (rangeError(next) === undefined) setState({ status: 'loading' });
  }

  async function loadMore(cursor: string) {
    setLoadingMore(true);
    setMoreError(undefined);
    try {
      const page = await api.listAuditEvents({ ...query.current, cursor });
      setState((current) =>
        current.status === 'loaded'
          ? {
              status: 'loaded',
              events: [...current.events, ...page.data],
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

  const loaded = state.status === 'loaded' && invalidRange === undefined ? state : undefined;
  return (
    <>
      <Card
        padding="none"
        footer={
          loaded !== undefined && loaded.events.length > 0 ? (
            <div className="flex flex-wrap items-center justify-between gap-3">
              <p aria-live="polite">
                Showing{' '}
                <span className="font-medium text-text tabular-nums">{loaded.events.length}</span>{' '}
                {loaded.events.length === 1 ? 'event' : 'events'}
                {loaded.nextCursor !== null && ' · more available'}
                <span className="hidden sm:inline"> · times in Philippine Time (PHT)</span>
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
        <form
          role="search"
          aria-label="Filter audit events"
          onSubmit={(e) => {
            e.preventDefault();
          }}
          className="border-b border-border px-4 py-3 sm:px-5"
        >
          <div className="grid grid-cols-2 gap-x-3 gap-y-2.5 md:grid-cols-4 xl:flex xl:items-end">
            <div className="col-span-2 space-y-1 md:col-span-1 xl:w-52">
              <label htmlFor={ids.action} className="block text-xs font-medium text-text-muted">
                Event
              </label>
              <select
                id={ids.action}
                value={filters.action}
                onChange={(e) => {
                  update({ action: e.target.value as AuditAction | '' });
                }}
                className={control}
              >
                <option value="">All events</option>
                {ACTIONS.map((a) => (
                  <option key={a} value={a}>
                    {ACTION_LABELS[a]}
                  </option>
                ))}
              </select>
            </div>
            <div className="col-span-2 space-y-1 md:col-span-1 xl:w-36">
              <label htmlFor={ids.outcome} className="block text-xs font-medium text-text-muted">
                Outcome
              </label>
              <select
                id={ids.outcome}
                value={filters.outcome}
                onChange={(e) => {
                  update({ outcome: e.target.value as AuditOutcome | '' });
                }}
                className={control}
              >
                <option value="">All outcomes</option>
                {OUTCOMES.map((o) => (
                  <option key={o} value={o}>
                    {OUTCOME_INFO[o].label}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1 xl:w-40">
              <label htmlFor={ids.from} className="block text-xs font-medium text-text-muted">
                From
              </label>
              <input
                id={ids.from}
                type="date"
                value={filters.from}
                onChange={(e) => {
                  update({ from: e.target.value });
                }}
                aria-invalid={invalidRange !== undefined || undefined}
                aria-describedby={invalidRange !== undefined ? ids.range : undefined}
                className={control}
              />
            </div>
            <div className="space-y-1 xl:w-40">
              <label htmlFor={ids.to} className="block text-xs font-medium text-text-muted">
                To (inclusive)
              </label>
              <input
                id={ids.to}
                type="date"
                value={filters.to}
                min={filters.from === '' ? undefined : filters.from}
                onChange={(e) => {
                  update({ to: e.target.value });
                }}
                aria-invalid={invalidRange !== undefined || undefined}
                aria-describedby={invalidRange !== undefined ? ids.range : undefined}
                className={control}
              />
            </div>
            <div className="col-span-2 flex items-center justify-end gap-2 md:col-span-4 xl:ml-auto">
              {filtered && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    update(NO_FILTERS);
                  }}
                >
                  Clear filters
                </Button>
              )}
              <Button
                variant="secondary"
                size="sm"
                onClick={refresh}
                disabled={state.status === 'loading' || invalidRange !== undefined}
                aria-label="Refresh audit log"
              >
                <Icon name="refresh" />
                <span className="hidden sm:inline">Refresh</span>
              </Button>
            </div>
          </div>
          {invalidRange !== undefined && (
            <p id={ids.range} className="mt-2 text-sm text-danger">
              {invalidRange}
            </p>
          )}
        </form>

        {invalidRange === undefined && state.status === 'loading' && (
          <Skeleton variant="table" bare lines={6} label="Loading audit events…" />
        )}

        {invalidRange === undefined && state.status === 'error' && (
          <div className="p-4 sm:p-5">
            <Alert
              tone="error"
              title="Could not load the audit log"
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

        {loaded?.events.length === 0 && filtered && (
          <EmptyState
            bare
            icon="shield"
            title="No events match these filters"
            action={
              <Button
                variant="secondary"
                onClick={() => {
                  update(NO_FILTERS);
                }}
              >
                Show all events
              </Button>
            }
          >
            Try another event type, outcome or date range.
          </EmptyState>
        )}

        {loaded?.events.length === 0 && !filtered && (
          <EmptyState bare icon="shield" title="No security events yet">
            Sign-ins, account locks and API key changes of your organization will appear here.
          </EmptyState>
        )}

        {loaded !== undefined && loaded.events.length > 0 && (
          <>
            <EventTable events={loaded.events} onSelect={setSelected} />
            {moreError !== undefined && (
              <div className="border-t border-border p-4 sm:px-5">
                <Alert tone="error" title={moreError} />
              </div>
            )}
          </>
        )}
      </Card>

      <AuditEventDialog
        event={selected}
        onClose={() => {
          setSelected(undefined);
        }}
      />
    </>
  );
}

const th = 'px-4 py-2.5 text-xs font-medium text-text-muted first:pl-5 last:pr-5';
const td = 'xl:px-4 xl:py-3 xl:first:pl-5 xl:last:pr-5';

/**
 * A table from 1280 px; below that each row becomes a compact card (no page-level horizontal
 * scrolling, AGENTS.md §8.8).
 * Every row has a "Details" button that opens all recorded fields of the event.
 */
function EventTable({
  events,
  onSelect,
}: {
  events: AuditEvent[];
  onSelect: (event: AuditEvent) => void;
}) {
  return (
    <table className="w-full text-left text-sm">
      <caption className="sr-only">Security audit events, newest first</caption>
      <thead className="hidden border-b border-border bg-surface-subtle xl:table-header-group">
        <tr>
          <th scope="col" className={th}>
            Time
          </th>
          <th scope="col" className={th}>
            Event
          </th>
          <th scope="col" className={th}>
            Outcome
          </th>
          <th scope="col" className={th}>
            By
          </th>
          <th scope="col" className={`${th} hidden xl:table-cell`}>
            IP address
          </th>
          <th scope="col" className={`${th} text-right`}>
            <span className="sr-only">Actions</span>
          </th>
        </tr>
      </thead>
      <tbody>
        {events.map((event) => {
          const outcome = outcomeBadge(event);
          const summary = eventSummary(event);
          const title = eventTitle(event);
          return (
            <tr
              key={event.id}
              className="grid grid-cols-2 gap-x-3 gap-y-1 border-t border-border px-4 py-3 transition-colors first:border-t-0 hover:bg-surface-subtle xl:table-row xl:p-0"
            >
              <td
                className={`order-3 col-span-2 text-xs whitespace-nowrap text-text-muted tabular-nums xl:order-none xl:text-sm ${td}`}
              >
                {formatDateTime(event.occurred_at, 'short')}
              </td>
              <td className={`order-1 min-w-0 xl:order-none ${td}`}>
                <p className="font-medium text-text">{title}</p>
                {summary !== '' && <p className="truncate text-xs text-text-muted">{summary}</p>}
              </td>
              <td className={`order-2 self-start justify-self-end xl:order-none ${td}`}>
                <Badge tone={outcome.tone} icon={outcome.icon}>
                  {outcome.label}
                </Badge>
              </td>
              <td className={`order-4 col-span-2 min-w-0 xl:order-none xl:col-span-1 ${td}`}>
                <span className="sr-only xl:hidden">By </span>
                <span className="block truncate text-text-muted xl:max-w-64 xl:text-text">
                  {actorText(event)}
                </span>
              </td>
              <td
                className={`hidden font-mono text-xs text-text-muted xl:table-cell ${td}`}
                title={event.user_agent ?? undefined}
              >
                {event.source_ip ?? '—'}
              </td>
              <td className={`order-5 col-span-2 xl:order-none xl:col-span-1 xl:text-right ${td}`}>
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label={`Details of “${title}”, ${formatDateTime(event.occurred_at, 'short')}`}
                  onClick={() => {
                    onSelect(event);
                  }}
                >
                  Details
                  <Icon name="chevronRight" />
                </Button>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
