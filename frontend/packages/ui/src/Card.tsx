import { useId, type ReactNode } from 'react';
import { classNames } from './class-names';

export type CardProps = {
  /** Optional heading. When present, the card is a region named by it. */
  heading?: string;
  /** Heading level, so the card fits the page's heading outline. Defaults to 2. */
  headingLevel?: 2 | 3 | 4;
  className?: string;
  children: ReactNode;
};

const cardClasses = 'rounded-card border border-border bg-surface p-4 shadow-sm sm:p-6';

export function Card({ heading, headingLevel = 2, className, children }: CardProps) {
  const headingId = useId();
  if (heading === undefined) {
    return <div className={classNames(cardClasses, className)}>{children}</div>;
  }
  const Heading = `h${String(headingLevel)}` as 'h2' | 'h3' | 'h4';
  return (
    <section aria-labelledby={headingId} className={classNames(cardClasses, className)}>
      <Heading id={headingId} className="text-lg font-semibold text-text">
        {heading}
      </Heading>
      <div className="mt-2 text-text-muted">{children}</div>
    </section>
  );
}
