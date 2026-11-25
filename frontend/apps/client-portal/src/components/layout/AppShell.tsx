import type { ReactNode } from 'react';
import { SkipLink } from './SkipLink';

export const MAIN_CONTENT_ID = 'main-content';

/**
 * Page structure: skip link, header (banner landmark) and main content (main landmark).
 * Mobile-first: content uses the full width on small screens with comfortable padding, and is
 * centred with a maximum width on large screens. There is no navigation yet: the portal has one
 * page, and navigation is added when more real pages exist (owner decision).
 */
export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col bg-surface-muted font-sans text-text">
      <SkipLink targetId={MAIN_CONTENT_ID}>Skip to main content</SkipLink>
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex min-h-14 w-full max-w-6xl items-center px-4 sm:px-6 lg:px-8">
          <p className="text-base font-semibold">Client Portal</p>
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
