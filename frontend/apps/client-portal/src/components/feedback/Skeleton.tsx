/** Placeholder lines shown while data loads; hidden from assistive technology, which hears `label`. */
export function Skeleton({ lines = 3, label }: { lines?: number; label: string }) {
  return (
    <div
      role="status"
      aria-live="polite"
      className="space-y-3 rounded-card border border-border bg-surface p-4 shadow-sm sm:p-5"
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
