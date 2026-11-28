import { Button, Icon, type IconName } from '@sms/ui';
import { useEffect, useId, useState, type ReactNode } from 'react';
import type { Permission, Session } from '../../lib/api/client';
import { Link } from '../../lib/router';
import { BrandMark } from './BrandMark';
import { SkipLink } from './SkipLink';

export const MAIN_CONTENT_ID = 'main-content';

export type AppShellProps = {
  /** The signed-in session; without it the page is a centred, navigation-free layout. */
  session?: Session;
  /** The current path, to mark the current navigation item. */
  path?: string;
  onSignOut?: () => Promise<void>;
  children: ReactNode;
};

export const ROLE_LABELS: Record<string, string> = {
  owner: 'Owner',
  sender: 'Sender',
  viewer: 'Viewer',
};

type NavItem = {
  to: string;
  label: string;
  icon: IconName;
  permission: Permission;
  isCurrent: (path: string) => boolean;
};

type NavSection = { label: string; items: NavItem[] };

/*
 * Information architecture: messaging work first, developer tools second. An item is shown only
 * when the session has its permission; the server enforces every permission regardless.
 */
const NAV: NavSection[] = [
  {
    label: 'Messaging',
    items: [
      {
        to: '/messages',
        label: 'Messages',
        icon: 'messages',
        permission: 'messages:read',
        isCurrent: (p) => p === '/' || (p.startsWith('/messages') && p !== '/messages/new'),
      },
      {
        to: '/messages/new',
        label: 'Send SMS',
        icon: 'send',
        permission: 'messages:send',
        isCurrent: (p) => p === '/messages/new',
      },
    ],
  },
  {
    label: 'Developers',
    items: [
      {
        to: '/api-keys',
        label: 'API keys',
        icon: 'key',
        permission: 'api_keys:manage',
        isCurrent: (p) => p.startsWith('/api-keys'),
      },
    ],
  },
];

/**
 * Page structure: skip link; a banner with the product, the navigation and the account (a fixed
 * sidebar on large screens, a top bar with a menu button on smaller ones); and the main content.
 */
export function AppShell({ session, path = '', onSignOut, children }: AppShellProps) {
  if (session === undefined) {
    return (
      <div className="flex min-h-dvh flex-col bg-surface-muted font-sans text-text">
        <SkipLink targetId={MAIN_CONTENT_ID}>Skip to main content</SkipLink>
        <header className="px-4 pt-8 sm:pt-12">
          <div className="mx-auto flex max-w-md items-center justify-center">
            <BrandMark />
          </div>
        </header>
        <main
          id={MAIN_CONTENT_ID}
          tabIndex={-1}
          className="mx-auto w-full max-w-md flex-1 px-4 py-6 focus:outline-none sm:py-8"
        >
          {children}
        </main>
      </div>
    );
  }
  return (
    <SignedInShell session={session} path={path} onSignOut={onSignOut}>
      {children}
    </SignedInShell>
  );
}

function SignedInShell({
  session,
  path,
  onSignOut,
  children,
}: {
  session: Session;
  path: string;
  onSignOut: (() => Promise<void>) | undefined;
  children: ReactNode;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [signingOut, setSigningOut] = useState(false);
  const panelId = useId();

  // Close the small-screen menu after navigating.
  const [menuPath, setMenuPath] = useState(path);
  if (menuPath !== path) {
    setMenuPath(path);
    setMenuOpen(false);
  }
  useEffect(() => {
    if (!menuOpen) return;
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') setMenuOpen(false);
    }
    window.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('keydown', onKey);
    };
  }, [menuOpen]);

  const sections = NAV.map((section) => ({
    ...section,
    items: section.items.filter((item) => session.permissions.includes(item.permission)),
  })).filter((section) => section.items.length > 0);
  const role = ROLE_LABELS[session.user.role] ?? session.user.role;

  return (
    <div className="min-h-dvh bg-surface-muted font-sans text-text">
      <SkipLink targetId={MAIN_CONTENT_ID}>Skip to main content</SkipLink>
      <header className="sticky top-0 z-20 border-b border-border bg-surface lg:fixed lg:inset-y-0 lg:left-0 lg:flex lg:w-64 lg:flex-col lg:border-r lg:border-b-0">
        <div className="flex h-14 items-center justify-between gap-3 px-4 lg:h-16 lg:px-5">
          <BrandMark />
          <Button
            variant="ghost"
            size="sm"
            className="lg:hidden"
            aria-expanded={menuOpen}
            aria-controls={panelId}
            onClick={() => {
              setMenuOpen((open) => !open);
            }}
          >
            <Icon name={menuOpen ? 'x' : 'menu'} className="size-5" />
            <span className="sr-only">{menuOpen ? 'Close menu' : 'Open menu'}</span>
          </Button>
        </div>

        <div
          id={panelId}
          className={`${menuOpen ? 'flex' : 'hidden'} flex-col border-t border-border lg:flex lg:flex-1 lg:overflow-y-auto lg:border-t-0`}
        >
          <div className="px-4 pt-4 lg:px-5">
            <p className="text-xs font-medium text-text-muted">Organization</p>
            <p className="truncate text-sm font-semibold" title={session.organization.name}>
              {session.organization.name}
            </p>
          </div>

          <nav aria-label="Main" className="flex-1 space-y-5 px-3 py-4">
            {sections.map((section) => (
              <div key={section.label} className="space-y-1">
                <p className="px-2 text-xs font-medium text-text-muted">{section.label}</p>
                <ul className="space-y-0.5">
                  {section.items.map((item) => {
                    const current = item.isCurrent(path);
                    return (
                      <li key={item.to}>
                        <Link
                          to={item.to}
                          aria-current={current ? 'page' : undefined}
                          className={`flex min-h-9 items-center gap-2.5 rounded-control px-2 text-sm font-medium transition-colors ${
                            current
                              ? 'bg-primary-subtle text-primary'
                              : 'text-text-muted hover:bg-surface-muted hover:text-text'
                          }`}
                        >
                          <Icon name={item.icon} />
                          {item.label}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </div>
            ))}
          </nav>

          <div className="border-t border-border p-3">
            <div className="flex items-center gap-3 px-2 py-1">
              <span
                aria-hidden="true"
                className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary-subtle text-xs font-semibold text-primary uppercase"
              >
                {session.user.email.slice(0, 1)}
              </span>
              <div className="min-w-0 text-sm">
                <p className="truncate font-medium" title={session.user.email}>
                  {session.user.email}
                </p>
                <p className="text-xs text-text-muted">
                  <span className="sr-only">Role: </span>
                  {role}
                </p>
              </div>
            </div>
            {onSignOut !== undefined && (
              <Button
                variant="ghost"
                size="sm"
                className="mt-1 w-full justify-start"
                loading={signingOut}
                onClick={() => {
                  setSigningOut(true);
                  void onSignOut().finally(() => {
                    setSigningOut(false);
                  });
                }}
              >
                {!signingOut && <Icon name="logOut" />}
                Sign out
              </Button>
            )}
          </div>
        </div>
      </header>

      <div className="lg:pl-64">
        <main
          id={MAIN_CONTENT_ID}
          tabIndex={-1}
          className="mx-auto w-full max-w-6xl px-4 py-6 focus:outline-none sm:px-6 lg:px-10 lg:py-8"
        >
          {children}
        </main>
      </div>
    </div>
  );
}
