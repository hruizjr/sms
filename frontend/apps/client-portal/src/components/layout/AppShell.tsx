import { Button } from '@sms/ui';
import { useState, type ReactNode } from 'react';
import type { Session } from '../../lib/api/client';
import { Link } from '../../lib/router';
import { SkipLink } from './SkipLink';

export const MAIN_CONTENT_ID = 'main-content';

export type AppShellProps = {
  /** The signed-in session; without it the header shows no navigation. */
  session?: Session;
  /** The current path, to mark the current navigation item. */
  path?: string;
  onSignOut?: () => Promise<void>;
  children: ReactNode;
};

const ROLE_LABELS: Record<string, string> = { owner: 'Owner', sender: 'Sender', viewer: 'Viewer' };

/**
 * Page structure: skip link, header (banner landmark) with the organization, the signed-in user and
 * navigation, and the main content (main landmark). Mobile-first: the header wraps on small
 * screens; content is centred with a maximum width on large screens.
 */
export function AppShell({ session, path = '', onSignOut, children }: AppShellProps) {
  const [signingOut, setSigningOut] = useState(false);
  const canSend = session?.permissions.includes('messages:send') ?? false;
  const composing = path === '/messages/new';
  const nav = [
    { to: '/messages', label: 'Messages', current: path.startsWith('/messages') && !composing },
    ...(canSend ? [{ to: '/messages/new', label: 'Send SMS', current: composing }] : []),
  ];

  return (
    <div className="flex min-h-dvh flex-col bg-surface-muted font-sans text-text">
      <SkipLink targetId={MAIN_CONTENT_ID}>Skip to main content</SkipLink>
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex min-h-14 w-full max-w-6xl flex-wrap items-center gap-x-6 gap-y-2 px-4 py-2 sm:px-6 lg:px-8">
          <p className="text-base font-semibold">Client Portal</p>
          {session !== undefined && (
            <>
              <nav aria-label="Main" className="flex gap-1">
                {nav.map((item) => (
                  <Link
                    key={item.to}
                    to={item.to}
                    aria-current={item.current ? 'page' : undefined}
                    className={`rounded-control px-3 py-2 text-sm font-medium hover:bg-surface-muted ${
                      item.current ? 'bg-surface-muted text-text' : 'text-text-muted'
                    }`}
                  >
                    {item.label}
                  </Link>
                ))}
              </nav>
              <div className="ml-auto flex flex-wrap items-center gap-3 text-sm">
                <p className="text-text-muted">
                  <span className="font-medium text-text">{session.organization.name}</span>
                  {' · '}
                  {session.user.email} ({ROLE_LABELS[session.user.role] ?? session.user.role})
                </p>
                {onSignOut !== undefined && (
                  <Button
                    variant="secondary"
                    size="sm"
                    loading={signingOut}
                    onClick={() => {
                      setSigningOut(true);
                      void onSignOut().finally(() => {
                        setSigningOut(false);
                      });
                    }}
                  >
                    Sign out
                  </Button>
                )}
              </div>
            </>
          )}
        </div>
      </header>
      <main
        id={MAIN_CONTENT_ID}
        tabIndex={-1}
        className="mx-auto w-full max-w-6xl flex-1 px-4 py-6 focus:outline-none sm:px-6 sm:py-8 lg:px-8"
      >
        {children}
      </main>
    </div>
  );
}
