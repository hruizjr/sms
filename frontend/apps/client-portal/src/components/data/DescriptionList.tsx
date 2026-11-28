import type { ReactNode } from 'react';

/*
 * Label and value pairs (a <dl>). "rows" puts the label beside the value on wide screens and draws a
 * separator between rows: dense and easy to scan, for records such as a message. "stacked" puts the
 * label above the value.
 */

export function DescriptionList({
  layout = 'rows',
  children,
}: {
  layout?: 'rows' | 'stacked';
  children: ReactNode;
}) {
  return (
    <dl
      data-layout={layout}
      className={
        layout === 'rows' ? 'divide-y divide-border' : 'grid gap-x-6 gap-y-4 sm:grid-cols-2'
      }
    >
      {children}
    </dl>
  );
}

export function DescriptionItem({
  term,
  layout = 'rows',
  children,
}: {
  term: string;
  layout?: 'rows' | 'stacked';
  children: ReactNode;
}) {
  if (layout === 'stacked') {
    return (
      <div className="min-w-0">
        <dt className="text-xs font-medium text-text-muted">{term}</dt>
        <dd className="mt-1 text-sm text-text">{children}</dd>
      </div>
    );
  }
  return (
    <div className="grid gap-1 py-3 first:pt-0 last:pb-0 sm:grid-cols-3 sm:gap-4">
      <dt className="text-sm text-text-muted">{term}</dt>
      <dd className="min-w-0 text-sm text-text sm:col-span-2">{children}</dd>
    </div>
  );
}
