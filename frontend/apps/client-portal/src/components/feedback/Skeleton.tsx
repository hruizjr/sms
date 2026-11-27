/** Placeholder lines shown while data loads; hidden from assistive technology, which hears `label`. */
export function Skeleton({ lines = 3, label }: { lines?: number; label: string }) {
  return (
    <div role="status" aria-live="polite" className="space-y-3">
      <span className="sr-only">{label}</span>
      {Array.from({ length: lines }, (_, i) => (
        <div key={i} aria-hidden="true" className="h-10 animate-pulse rounded-control bg-border" />
      ))}
    </div>
  );
}
