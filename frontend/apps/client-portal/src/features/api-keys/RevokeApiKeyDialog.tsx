import { Button, Dialog } from '@sms/ui';
import { useRef, useState } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { ApiError, type ApiKey } from '../../lib/api/client';
import { useSession } from '../auth/session';
import { apiKeyErrorMessage } from './scopes';

/** Confirmation before revoking a key (a destructive action that cannot be undone). */
export function RevokeApiKeyDialog({
  apiKey,
  onClose,
  onRevoked,
  onGone,
}: {
  apiKey: ApiKey | undefined;
  onClose: () => void;
  onRevoked: (key: ApiKey) => void;
  /** The key no longer exists for this organization (another tab revoked it, for example). */
  onGone: () => void;
}) {
  const { api } = useSession();
  const [revoking, setRevoking] = useState(false);
  const [failure, setFailure] = useState<string | undefined>(undefined);
  const inFlight = useRef(false);
  const [shownFor, setShownFor] = useState<string | undefined>(undefined);
  if (shownFor !== apiKey?.id) {
    setShownFor(apiKey?.id);
    setFailure(undefined);
  }

  async function revoke(key: ApiKey) {
    if (inFlight.current) return;
    inFlight.current = true;
    setRevoking(true);
    setFailure(undefined);
    try {
      onRevoked(await api.revokeApiKey(key.id));
    } catch (error: unknown) {
      if (error instanceof ApiError && error.status === 404) {
        onGone();
      } else if (!(error instanceof ApiError && error.status === 401)) {
        setFailure(apiKeyErrorMessage(error, 'The key could not be revoked. Try again.'));
      }
    } finally {
      inFlight.current = false;
      setRevoking(false);
    }
  }

  return (
    <Dialog
      open={apiKey !== undefined}
      onClose={onClose}
      dismissible={!revoking}
      title={apiKey !== undefined ? `Revoke “${apiKey.name}”?` : 'Revoke API key?'}
      description="Applications using this key will be refused by the messaging API from their next request. This cannot be undone."
      footer={
        apiKey !== undefined && (
          <>
            <Button variant="secondary" disabled={revoking} onClick={onClose}>
              Cancel
            </Button>
            <Button variant="danger" loading={revoking} onClick={() => void revoke(apiKey)}>
              {revoking ? 'Revoking…' : 'Revoke key'}
            </Button>
          </>
        )
      }
    >
      {apiKey !== undefined && (
        <div className="space-y-3">
          {failure !== undefined && <Alert tone="error" title={failure} />}
          <p className="rounded-control border border-border bg-surface-subtle px-3 py-2 font-mono text-xs break-all text-text-muted">
            {apiKey.prefix}_…
          </p>
        </div>
      )}
    </Dialog>
  );
}
