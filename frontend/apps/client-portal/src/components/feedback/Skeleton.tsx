/**
 * Placeholders shown while data loads; hidden from assistive technology, which hears `label`.
 * "lines" is a block of text lines; "table" mimics the rows of a list, so the page does not jump
 * when the data arrives.
 */
export function Skeleton({
  lines = 3,
  label,
  variant = 'lines',
  bare = false,
}: {
  lines?: number;
  label: string;
  variant?: 'lines' | 'table';
  /** Inside a card that already draws the border. */
  bare?: boolean;
}) {
  if (variant === 'table') {
    return (
      <div
        role="status"
        aria-live="polite"
        className={
          bare ? '' : 'overflow-hidden rounded-card border border-border bg-surface shadow-xs'
        }
      >
        <span className="sr-only">{label}</span>
        <div aria-hidden="true" className="h-10 border-b border-border bg-surface-subtle" />
        {Array.from({ length: lines }, (_, i) => (
          <div
            key={i}
            aria-hidden="true"
            className="flex items-center gap-4 border-b border-border px-4 py-4 last:border-b-0 sm:px-5"
          >
            <div className="h-3.5 w-1/4 animate-pulse rounded-control bg-surface-muted" />
            <div className="h-5 w-20 animate-pulse rounded-full bg-surface-muted" />
            <div className="hidden h-3.5 w-1/6 animate-pulse rounded-control bg-surface-muted sm:block" />
            <div className="ml-auto h-3.5 w-24 animate-pulse rounded-control bg-surface-muted" />
          </div>
        ))}
      </div>
    );
  }
  return (
    <div
      role="status"
      aria-live="polite"
      className="space-y-3 rounded-card border border-border bg-surface p-4 shadow-xs sm:p-5"
    >
      <span className="sr-only">{label}</span>
      {Array.from({ length: lines }, (_, i) => (
        <div
          key={i}
          aria-hidden="true"
          className={`h-4 animate-pulse rounded-control bg-surface-muted ${i % 3 === 2 ? 'w-2/3' : 'w-full'}`}
        />
      ))}
    </div>
  );
}
