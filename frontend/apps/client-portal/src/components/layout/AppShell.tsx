import { Icon, type IconName } from '@sms/ui';
import { useEffect, useId, useRef, useState, type ReactNode } from 'react';
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

/** Controls on the dark navigation surface: their own hover colours and focus indicator. */
const navControl =
  'flex min-h-9 w-full items-center gap-3 rounded-control px-3 text-sm font-medium transition-colors focus-visible:outline-nav-focus';

/**
 * Page structure: skip link; a banner with the product, the navigation and the account; and the
 * main content. Wide screens (1024 px and up) have a fixed dark sidebar. Smaller screens have a top
 * bar whose menu button opens the same navigation in a modal panel.
 */
export function AppShell({ session, path = '', onSignOut, children }: AppShellProps) {
  if (session === undefined) {
    return (
      <div className="flex min-h-dvh flex-col bg-surface-muted font-sans text-text">
        <SkipLink targetId={MAIN_CONTENT_ID}>Skip to main content</SkipLink>
        <header className="px-4 pt-10 sm:pt-16">
          <div className="mx-auto flex max-w-sm items-center justify-center">
            <BrandMark />
          </div>
        </header>
        <main
          id={MAIN_CONTENT_ID}
          tabIndex={-1}
          className="mx-auto w-full max-w-sm flex-1 px-4 py-8 focus:outline-none"
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
  const menuId = useId();

  // Close the small-screen menu after navigating.
  const [menuPath, setMenuPath] = useState(path);
  if (menuPath !== path) {
    setMenuPath(path);
    setMenuOpen(false);
  }

  return (
    <div className="min-h-dvh bg-surface-muted font-sans text-text">
      <SkipLink targetId={MAIN_CONTENT_ID}>Skip to main content</SkipLink>
      <header className="sticky top-0 z-20 lg:fixed lg:inset-y-0 lg:left-0 lg:w-60">
        <div className="flex h-14 items-center justify-between gap-3 bg-nav px-4 lg:hidden">
          <BrandMark tone="inverse" />
          <button
            type="button"
            aria-haspopup="dialog"
            aria-expanded={menuOpen}
            aria-controls={menuId}
            onClick={() => {
              setMenuOpen(true);
            }}
            className="flex size-9 items-center justify-center rounded-control text-nav-foreground transition-colors hover:bg-nav-raised hover:text-nav-active-foreground focus-visible:outline-nav-focus"
          >
            <Icon name="menu" className="size-5" />
            <span className="sr-only">Open menu</span>
          </button>
        </div>

        <div className="hidden h-full bg-nav lg:block">
          <Navigation session={session} path={path} onSignOut={onSignOut} />
        </div>

        <MenuPanel
          id={menuId}
          open={menuOpen}
          onClose={() => {
            setMenuOpen(false);
          }}
        >
          <Navigation
            session={session}
            path={path}
            onSignOut={onSignOut}
            onClose={() => {
              setMenuOpen(false);
            }}
          />
        </MenuPanel>
      </header>

      <div className="lg:pl-60">
        <main
          id={MAIN_CONTENT_ID}
          tabIndex={-1}
          className="mx-auto w-full max-w-7xl px-4 py-6 focus:outline-none sm:px-6 lg:px-8 lg:py-8"
        >
          {children}
        </main>
      </div>
    </div>
  );
}

/**
 * The small-screen menu: a modal panel on the native <dialog> element, so the browser keeps focus
 * inside it, makes the page behind it inert, and returns focus to the menu button when it closes.
 * It closes with Escape, the close button, a click outside the panel, or navigation.
 */
function MenuPanel({
  id,
  open,
  onClose,
  children,
}: {
  id: string;
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (dialog === null) return;
    if (open && !dialog.open) {
      dialog.showModal();
      // Start on the current page's link, else the first control.
      (
        dialog.querySelector<HTMLElement>('[aria-current="page"]') ??
        dialog.querySelector<HTMLElement>('a, button')
      )?.focus();
    }
    if (!open && dialog.open) dialog.close();
  }, [open]);

  useEffect(() => {
    const dialog = ref.current;
    if (dialog === null) return;
    function onCancel(event: Event) {
      event.preventDefault();
      onClose();
    }
    dialog.addEventListener('cancel', onCancel);
    return () => {
      dialog.removeEventListener('cancel', onCancel);
    };
  }, [onClose]);

  return (
    <dialog
      ref={ref}
      id={id}
      aria-label="Menu"
      onClick={(event) => {
        // A click on the backdrop targets the dialog element itself; the panel fills the rest.
        if (event.target === event.currentTarget) onClose();
      }}
      className="m-0 h-dvh max-h-none w-full max-w-xs bg-nav p-0 text-nav-foreground shadow-xl backdrop:bg-text/50 lg:hidden"
    >
      {open && children}
    </dialog>
  );
}

function Navigation({
  session,
  path,
  onSignOut,
  onClose,
}: {
  session: Session;
  path: string;
  onSignOut: (() => Promise<void>) | undefined;
  /** In the small-screen menu: closes it. */
  onClose?: () => void;
}) {
  const [signingOut, setSigningOut] = useState(false);
  const sections = NAV.map((section) => ({
    ...section,
    items: section.items.filter((item) => session.permissions.includes(item.permission)),
  })).filter((section) => section.items.length > 0);
  const role = ROLE_LABELS[session.user.role] ?? session.user.role;

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-nav-raised px-5 lg:h-16">
        <BrandMark tone="inverse" />
        {onClose !== undefined && (
          <button
            type="button"
            onClick={onClose}
            className="-mr-2 flex size-9 items-center justify-center rounded-control text-nav-foreground transition-colors hover:bg-nav-raised hover:text-nav-active-foreground focus-visible:outline-nav-focus"
          >
            <Icon name="x" className="size-5" />
            <span className="sr-only">Close menu</span>
          </button>
        )}
      </div>

      <div className="px-3 pt-4">
        <div className="flex items-center gap-3 rounded-control border border-nav-raised px-3 py-2.5">
          <span
            aria-hidden="true"
            className="flex size-8 shrink-0 items-center justify-center rounded-control bg-nav-raised text-nav-foreground"
          >
            <Icon name="building" />
          </span>
          <div className="min-w-0">
            <p className="text-xs text-nav-muted">Organization</p>
            <p
              className="line-clamp-2 text-sm font-medium break-words text-nav-active-foreground"
              title={session.organization.name}
            >
              {session.organization.name}
            </p>
          </div>
        </div>
      </div>

      <nav aria-label="Main" className="flex-1 space-y-6 overflow-y-auto px-3 py-5">
        {sections.map((section) => (
          <div key={section.label}>
            <p className="mb-1.5 px-3 text-xs font-medium tracking-wide text-nav-muted uppercase">
              {section.label}
            </p>
            <ul className="space-y-0.5">
              {section.items.map((item) => {
                const current = item.isCurrent(path);
                return (
                  <li key={item.to}>
                    <Link
                      to={item.to}
                      aria-current={current ? 'page' : undefined}
                      className={`relative ${navControl} ${
                        current
                          ? 'bg-nav-raised text-nav-active-foreground'
                          : 'text-nav-foreground hover:bg-nav-raised hover:text-nav-active-foreground'
                      }`}
                    >
                      {current && (
                        <span
                          aria-hidden="true"
                          className="absolute inset-y-2 left-0 w-0.5 rounded-full bg-nav-focus"
                        />
                      )}
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

      <div className="shrink-0 border-t border-nav-raised p-3">
        <div className="flex items-center gap-3 px-3 py-2">
          <span
            aria-hidden="true"
            className="flex size-8 shrink-0 items-center justify-center rounded-full bg-nav-raised text-xs font-semibold text-nav-active-foreground uppercase"
          >
            {session.user.email.slice(0, 1)}
          </span>
          <div className="min-w-0 text-sm">
            <p
              className="truncate font-medium text-nav-active-foreground"
              title={session.user.email}
            >
              {session.user.email}
            </p>
            <p className="text-xs text-nav-muted">
              <span className="sr-only">Role: </span>
              {role}
            </p>
          </div>
        </div>
        {onSignOut !== undefined && (
          <button
            type="button"
            disabled={signingOut}
            aria-busy={signingOut || undefined}
            onClick={() => {
              setSigningOut(true);
              void onSignOut().finally(() => {
                setSigningOut(false);
              });
            }}
            className={`mt-1 ${navControl} text-nav-foreground hover:bg-nav-raised hover:text-nav-active-foreground disabled:cursor-not-allowed disabled:opacity-60`}
          >
            {signingOut ? (
              <span
                aria-hidden="true"
                className="size-4 animate-spin rounded-full border-2 border-current border-t-transparent"
              />
            ) : (
              <Icon name="logOut" />
            )}
            Sign out
          </button>
        )}
      </div>
    </div>
  );
}
