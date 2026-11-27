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
  /** "none" lets content such as a table reach the card's edges. Defaults to "md". */
  padding?: 'md' | 'none';
  className?: string;
  children: ReactNode;
};

const cardClasses = 'rounded-card border border-border bg-surface shadow-sm';

/** A bordered panel. With a heading it is a named region with a header row. */
export function Card({
  heading,
  headingLevel = 2,
  description,
  actions,
  padding = 'md',
  className,
  children,
}: CardProps) {
  const headingId = useId();
  const bodyPadding = padding === 'md' ? 'p-4 sm:p-5' : '';
  if (heading === undefined) {
    return <div className={classNames(cardClasses, bodyPadding, className)}>{children}</div>;
  }
  const Heading = `h${String(headingLevel)}` as 'h2' | 'h3' | 'h4';
  return (
    <section aria-labelledby={headingId} className={classNames(cardClasses, className)}>
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border px-4 py-3 sm:px-5">
        <div className="min-w-0 space-y-0.5">
          <Heading id={headingId} className="text-sm font-semibold text-text">
            {heading}
          </Heading>
          {description !== undefined && <p className="text-sm text-text-muted">{description}</p>}
        </div>
        {actions !== undefined && <div className="flex shrink-0 gap-2">{actions}</div>}
      </div>
      <div className={classNames('text-sm text-text', bodyPadding)}>{children}</div>
    </section>
  );
}
