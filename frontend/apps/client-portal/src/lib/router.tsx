import { useSyncExternalStore, type MouseEvent, type ReactNode } from 'react';

/*
 * A minimal client-side router on the browser History API (owner decision G3-6: no router library in
 * this group). It provides the current path, navigation, and links that behave like normal links
 * (open in a new tab with a modifier key or middle click).
 */

const listeners = new Set<() => void>();

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  window.addEventListener('popstate', listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener('popstate', listener);
  };
}

/** The current path (window.location.pathname), re-rendering on navigation. */
export function usePath(): string {
  return useSyncExternalStore(subscribe, () => window.location.pathname);
}

/** Moves to path and notifies every usePath subscriber. */
export function navigate(path: string, options: { replace?: boolean } = {}): void {
  if (path !== window.location.pathname) {
    if (options.replace === true) {
      window.history.replaceState(null, '', path);
    } else {
      window.history.pushState(null, '', path);
    }
  }
  for (const listener of listeners) listener();
}

export type LinkProps = {
  to: string;
  className?: string;
  children: ReactNode;
  'aria-current'?: 'page' | undefined;
};

export function Link({ to, className, children, ...rest }: LinkProps) {
  function onClick(event: MouseEvent<HTMLAnchorElement>) {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
      return; // let the browser open a new tab or window
    }
    event.preventDefault();
    navigate(to);
  }
  return (
    <a href={to} onClick={onClick} className={className} {...rest}>
      {children}
    </a>
  );
}
