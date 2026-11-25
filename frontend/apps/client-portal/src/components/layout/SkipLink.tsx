type SkipLinkProps = { targetId: string; children: string };

/**
 * First focusable element on the page. Hidden until it receives keyboard focus, then lets keyboard
 * and screen reader users jump past the header to the main content (WCAG 2.2 SC 2.4.1).
 * Padding is applied on focus because `not-sr-only` resets padding to 0.
 */
export function SkipLink({ targetId, children }: SkipLinkProps) {
  return (
    <a
      href={`#${targetId}`}
      className="sr-only rounded-control border border-border-strong bg-surface font-medium text-text focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-10 focus:px-4 focus:py-2 focus:shadow-md"
    >
      {children}
    </a>
  );
}
