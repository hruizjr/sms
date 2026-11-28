import { useId, type ReactNode } from 'react';
import { classNames } from './class-names';

export type CardProps = {
  /** Optional heading. When present, the card is a region named by it. */
  heading?: string;
  /** Heading level, so the card fits the page's heading outline. Defaults to 2. */
  headingLevel?: 2 | 3 | 4;
  /** A short line under the heading. */
  description?: ReactNode;
  /** Controls at the right of the heading (for example a button). */
  actions?: ReactNode;
  /** A footer row with a subtle background (for example pagination or a note). */
  footer?: ReactNode;
  /** "none" lets content such as a table reach the card's edges. Defaults to "md". */
  padding?: 'md' | 'none';
  className?: string;
  children: ReactNode;
};

const cardClasses = 'overflow-hidden rounded-card border border-border bg-surface shadow-xs';

/** A bordered panel. With a heading it is a named region with a header row. */
export function Card({
  heading,
  headingLevel = 2,
  description,
  actions,
  footer,
  padding = 'md',
  className,
  children,
}: CardProps) {
  const headingId = useId();
  const bodyPadding = padding === 'md' ? 'p-4 sm:p-5' : '';
  const footerRow = footer !== undefined && (
    <div className="border-t border-border bg-surface-subtle px-4 py-3 text-sm text-text-muted sm:px-5">
      {footer}
    </div>
  );
  if (heading === undefined) {
    return (
      <div className={classNames(cardClasses, className)}>
        <div className={bodyPadding}>{children}</div>
        {footerRow}
      </div>
    );
  }
  const Heading = `h${String(headingLevel)}` as 'h2' | 'h3' | 'h4';
  return (
    <section aria-labelledby={headingId} className={classNames(cardClasses, className)}>
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b border-border px-4 py-3 sm:px-5">
        <div className="min-w-0">
          <Heading id={headingId} className="text-sm leading-6 font-semibold text-text">
            {heading}
          </Heading>
          {description !== undefined && (
            <p className="text-xs leading-5 text-text-muted">{description}</p>
          )}
        </div>
        {actions !== undefined && (
          <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>
        )}
      </div>
      <div className={classNames('text-sm text-text', bodyPadding)}>{children}</div>
      {footerRow}
    </section>
  );
}
