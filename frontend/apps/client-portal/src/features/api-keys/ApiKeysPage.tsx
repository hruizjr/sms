import { Badge, Button, Card, Icon } from '@sms/ui';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { EmptyState } from '../../components/feedback/EmptyState';
import { Skeleton } from '../../components/feedback/Skeleton';
import { PageHeader } from '../../components/layout/PageHeader';
import { NetworkError, type ApiKey, type ApiKeyCreated } from '../../lib/api/client';
import { hasPermission, useSession, useSignedInSession } from '../auth/session';
import { formatDateTime } from '../messages/status';
import { CreateApiKeyDialog } from './CreateApiKeyDialog';
import { NewApiKeyPanel } from './NewApiKeyPanel';
import { RevokeApiKeyDialog } from './RevokeApiKeyDialog';
import { SCOPE_LABELS } from './scopes';

type ListState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'loaded'; keys: ApiKey[] };

/** Active keys first (newest first), then revoked keys (most recently revoked first). */
function sortKeys(keys: ApiKey[]): ApiKey[] {
  const time = (k: ApiKey) => Date.parse(k.revoked_at ?? k.created_at);
  return [...keys].sort((a, b) =>
    a.status !== b.status ? (a.status === 'active' ? -1 : 1) : time(b) - time(a),
  );
}

/**
 * API keys of the organization for the public messaging API (owners only, MVP-03). A new key's full
 * value is held in memory by this page only until the owner dismisses it or leaves the page; it is
 * never written to browser storage and cannot be fetched again.
 */
export function ApiKeysPage() {
  const session = useSignedInSession();
  useEffect(() => {
    document.title = 'API keys · Client Portal';
  }, []);

  return (
    <div>
      <PageHeader
        title="API keys"
        description="API keys let your applications send and read messages through the public messaging API. Treat each key like a password."
      />
      {hasPermission(session, 'api_keys:manage') ? (
        <ApiKeys />
      ) : (
        <Alert tone="info" title="Only organization owners can manage API keys">
          Ask an owner of your organization’s account to create or revoke API keys.
        </Alert>
      )}
    </div>
  );
}

function ApiKeys() {
  const { api } = useSession();
  const [state, setState] = useState<ListState>({ status: 'loading' });
  const [reload, setReload] = useState(0);
  const [creating, setCreating] = useState(false);
  const [created, setCreated] = useState<ApiKeyCreated | undefined>(undefined);
  const [revoking, setRevoking] = useState<ApiKey | undefined>(undefined);
  const [notice, setNotice] = useState<string | undefined>(undefined);
  const createButton = useRef<HTMLButtonElement>(null);
  const returnFocus = useRef(false);

  // After the one-time key display is dismissed, return focus to "Create API key". Done after the
  // panel (which held focus) has been removed, so the browser cannot move focus back to the page.
  useEffect(() => {
    if (!returnFocus.current || created !== undefined) return;
    returnFocus.current = false;
    createButton.current?.focus();
  }, [created]);

  useEffect(() => {
    let active = true;
    api.listApiKeys().then(
      (keys) => {
        if (active) setState({ status: 'loaded', keys: sortKeys(keys) });
      },
      (error: unknown) => {
        if (!active) return;
        setState({
          status: 'error',
          message:
            error instanceof NetworkError
              ? 'The server could not be reached. Check your connection and try again.'
              : 'Your API keys could not be loaded. Try again.',
        });
      },
    );
    return () => {
      active = false;
    };
  }, [api, reload]);

  const refresh = useCallback(() => {
    setState({ status: 'loading' });
    setReload((n) => n + 1);
  }, []);

  function upsert(key: ApiKey) {
    setState((current) =>
      current.status === 'loaded'
        ? {
            status: 'loaded',
            keys: sortKeys([key, ...current.keys.filter((k) => k.id !== key.id)]),
          }
        : current,
    );
  }

  const openCreate = (
    <Button
      ref={createButton}
      onClick={() => {
        setNotice(undefined);
        setCreating(true);
      }}
    >
      <Icon name="plus" />
      Create API key
    </Button>
  );

  return (
    <div className="space-y-6">
      {created !== undefined && (
        <NewApiKeyPanel
          created={created}
          onDone={() => {
            returnFocus.current = true;
            setCreated(undefined);
          }}
        />
      )}

      {notice !== undefined && <Alert tone="success" title={notice} />}

      {state.status === 'loading' && <Skeleton lines={4} label="Loading API keys…" />}

      {state.status === 'error' && (
        <Alert
          tone="error"
          title="Could not load API keys"
          action={
            <Button variant="secondary" size="sm" onClick={refresh}>
              Try again
            </Button>
          }
        >
          {state.message}
        </Alert>
      )}

      {state.status === 'loaded' && state.keys.length === 0 && (
        <EmptyState icon="key" title="No API keys yet" action={openCreate}>
          Create a key to connect your application to the messaging API. You choose what each key
          may do, and you can revoke it at any time.
        </EmptyState>
      )}

      {state.status === 'loaded' && state.keys.length > 0 && (
        <Card
          heading="Keys"
          description={`${String(state.keys.filter((k) => k.status === 'active').length)} active · ${String(state.keys.filter((k) => k.status === 'revoked').length)} revoked`}
          actions={openCreate}
          padding="none"
          className="overflow-hidden"
        >
          <KeyTable
            keys={state.keys}
            onRevoke={(key) => {
              setNotice(undefined);
              setRevoking(key);
            }}
          />
          <p className="border-t border-border bg-surface-subtle px-4 py-3 text-xs text-text-muted sm:px-5">
            The platform does not record when a key was last used. Revoke keys you no longer need.
          </p>
        </Card>
      )}

      <CreateApiKeyDialog
        open={creating}
        onClose={() => {
          setCreating(false);
        }}
        onCreated={(result) => {
          setCreating(false);
          setCreated(result);
          upsert(result.api_key);
        }}
      />

      <RevokeApiKeyDialog
        apiKey={revoking}
        onClose={() => {
          setRevoking(undefined);
        }}
        onRevoked={(key) => {
          setRevoking(undefined);
          upsert(key);
          setNotice(`API key “${key.name}” was revoked.`);
        }}
        onGone={() => {
          setRevoking(undefined);
          setNotice('That API key was already revoked. The list has been refreshed.');
          refresh();
        }}
      />
    </div>
  );
}

/**
 * Wide screens (1280 px and up): a table. Smaller screens: each row becomes a stacked card with the status next to
 * the name (CSS order; the table order is unchanged for assistive technology).
 */
function KeyTable({ keys, onRevoke }: { keys: ApiKey[]; onRevoke: (key: ApiKey) => void }) {
  return (
    <table className="w-full text-left text-sm">
      <caption className="sr-only">API keys, active keys first</caption>
      <thead className="hidden border-b border-border bg-surface-subtle text-xs text-text-muted xl:table-header-group">
        <tr>
          <th scope="col" className="px-5 py-2.5 font-medium">
            Name
          </th>
          <th scope="col" className="px-4 py-2.5 font-medium">
            Permissions
          </th>
          <th scope="col" className="px-4 py-2.5 font-medium">
            Created
          </th>
          <th scope="col" className="px-4 py-2.5 font-medium">
            Status
          </th>
          <th scope="col" className="px-5 py-2.5 text-right font-medium">
            <span className="sr-only">Actions</span>
          </th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {keys.map((key) => {
          const revoked = key.status === 'revoked';
          return (
            <tr
              key={key.id}
              className="grid grid-cols-2 gap-x-3 gap-y-2 px-4 py-4 transition-colors hover:bg-surface-subtle sm:px-5 xl:table-row xl:p-0"
            >
              <td className="order-1 min-w-0 xl:px-5 xl:py-3">
                <p className={`truncate font-medium ${revoked ? 'text-text-muted' : 'text-text'}`}>
                  {key.name}
                </p>
                <p className="font-mono text-xs text-text-muted" title={key.prefix}>
                  smsk_…{key.id.slice(-8)}
                  <span className="sr-only"> (key ID {key.id})</span>
                </p>
              </td>
              <td className="order-3 col-span-2 xl:px-4 xl:py-3">
                <span className="sr-only xl:hidden">Permissions: </span>
                <span className="flex flex-wrap gap-1">
                  {key.scopes.map((scope) => (
                    <Badge key={scope} className="font-mono">
                      {scope}
                    </Badge>
                  ))}
                </span>
                <span className="sr-only">
                  {' '}
                  ({key.scopes.map((scope) => SCOPE_LABELS[scope].title).join(', ')})
                </span>
              </td>
              <td className="order-4 col-span-2 text-xs text-text-muted xl:px-4 xl:py-3 xl:whitespace-nowrap">
                <span className="xl:hidden">Created </span>
                {formatDateTime(key.created_at, 'short')}
                {key.revoked_at !== null && (
                  <span className="xl:hidden">
                    {' · Revoked '}
                    {formatDateTime(key.revoked_at, 'short')}
                  </span>
                )}
              </td>
              <td className="order-2 self-start justify-self-end xl:px-4 xl:py-3">
                {revoked ? (
                  <span className="flex flex-col gap-0.5">
                    <Badge tone="neutral" icon="x">
                      Revoked
                    </Badge>
                    {key.revoked_at !== null && (
                      <span className="hidden text-xs whitespace-nowrap text-text-muted xl:block">
                        {formatDateTime(key.revoked_at, 'short')}
                      </span>
                    )}
                  </span>
                ) : (
                  <Badge tone="success" icon="check">
                    Active
                  </Badge>
                )}
              </td>
              <td
                className={`order-5 col-span-2 xl:px-5 xl:py-3 xl:text-right ${revoked ? 'hidden xl:table-cell' : ''}`}
              >
                {!revoked && (
                  <Button
                    variant="danger-secondary"
                    size="sm"
                    aria-label={`Revoke API key ${key.name}`}
                    onClick={() => {
                      onRevoke(key);
                    }}
                  >
                    Revoke
                  </Button>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
