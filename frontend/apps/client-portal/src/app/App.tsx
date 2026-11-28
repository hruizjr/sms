import { Button } from '@sms/ui';
import { useEffect } from 'react';
import { Alert } from '../components/feedback/Alert';
import { Skeleton } from '../components/feedback/Skeleton';
import { AppShell } from '../components/layout/AppShell';
import { PageHeader } from '../components/layout/PageHeader';
import { ApiKeysPage } from '../features/api-keys/ApiKeysPage';
import { AuditLogPage } from '../features/audit-log/AuditLogPage';
import { SignInPage } from '../features/auth/SignInPage';
import { SessionProvider, useSession } from '../features/auth/session';
import { ComposePage } from '../features/messages/ComposePage';
import { MessageDetailPage } from '../features/messages/MessageDetailPage';
import { MessagesPage } from '../features/messages/MessagesPage';
import { browserTransport, type Transport } from '../lib/api/client';
import { Link, navigate, usePath } from '../lib/router';
import { AppErrorBoundary } from './AppErrorBoundary';

export type AppProps = {
  /** How API requests are sent: the browser's fetch, or a fake in tests. */
  transport?: Transport;
  /** Generates idempotency keys for message submission (tests make it deterministic). */
  newIdempotencyKey?: () => string;
};

/**
 * The client portal (Phase 4 Groups 3 to 6): sign-in, the organization's messages, message details,
 * single-SMS composition, API key management and the security audit log, against the
 * session-authenticated portal API.
 * Routes:
 *
 *   /messages        message list (also the start page, "/")
 *   /messages/new    compose and send one SMS
 *   /messages/{id}   message details and status history
 *   /api-keys        API keys of the public API (owners)
 *   /audit-log       the organization's security audit log (owners)
 *
 * Without a valid session every route shows the sign-in page. All access control is enforced by the
 * server; hiding a link here is a convenience only.
 */
export function App({ transport = browserTransport, newIdempotencyKey }: AppProps) {
  return (
    <SessionProvider transport={transport}>
      <Portal newIdempotencyKey={newIdempotencyKey} />
    </SessionProvider>
  );
}

const MESSAGE_PATH = /^\/messages\/([A-Za-z0-9_-]{1,64})$/;

function Portal({ newIdempotencyKey }: { newIdempotencyKey: (() => string) | undefined }) {
  const { state, signOut, retry } = useSession();
  const path = usePath();

  useEffect(() => {
    if (path === '/' && state.status === 'signed-in') navigate('/messages', { replace: true });
  }, [path, state.status]);

  if (state.status === 'loading') {
    return (
      <AppShell>
        <Skeleton lines={2} label="Checking your session…" />
      </AppShell>
    );
  }
  if (state.status === 'unavailable') {
    return (
      <AppShell>
        <Alert
          tone="error"
          title="The portal is not available"
          action={
            <Button variant="secondary" size="sm" onClick={retry}>
              Try again
            </Button>
          }
        >
          The server could not be reached. Check your connection and try again.
        </Alert>
      </AppShell>
    );
  }
  if (state.status === 'signed-out') {
    return (
      <AppShell>
        <SignInPage notice={state.notice} />
      </AppShell>
    );
  }

  // The route key identifies the page ("/" and "/messages" are the same page), so a redirect does
  // not remount it, while moving to another page resets the error boundary.
  let page;
  let routeKey;
  const detail = MESSAGE_PATH.exec(path);
  if (path === '/' || path === '/messages') {
    page = <MessagesPage />;
    routeKey = 'messages';
  } else if (path === '/messages/new') {
    page = <ComposePage {...(newIdempotencyKey ? { newIdempotencyKey } : {})} />;
    routeKey = 'compose';
  } else if (path === '/api-keys') {
    page = <ApiKeysPage />;
    routeKey = 'api-keys';
  } else if (path === '/audit-log') {
    page = <AuditLogPage />;
    routeKey = 'audit-log';
  } else if (detail?.[1] !== undefined) {
    const justSent = new URLSearchParams(window.location.search).has('sent');
    page = <MessageDetailPage id={detail[1]} justSent={justSent} />;
    routeKey = `message:${detail[1]}`;
  } else {
    page = <NotFound />;
    routeKey = 'not-found';
  }

  return (
    <AppShell session={state.session} path={path} onSignOut={signOut}>
      <AppErrorBoundary key={routeKey}>{page}</AppErrorBoundary>
    </AppShell>
  );
}

function NotFound() {
  return (
    <div>
      <PageHeader
        title="Page not found"
        description="The address does not match a page of the portal."
      />
      <Link to="/messages" className="text-sm font-medium text-primary underline">
        Go to your messages
      </Link>
    </div>
  );
}
