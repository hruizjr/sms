import { Icon, type IconName } from '@sms/ui';
import { useId, type ReactNode } from 'react';

/** A named region explaining why a list is empty, with an optional next step. */
export function EmptyState({
  icon,
  title,
  children,
  action,
  bare = false,
}: {
  icon: IconName;
  title: string;
  children: ReactNode;
  action?: ReactNode;
  /** Inside a card that already draws the border. */
  bare?: boolean;
}) {
  const headingId = useId();
  return (
    <section
      aria-labelledby={headingId}
      className={`flex flex-col items-center px-6 py-14 text-center ${
        bare ? '' : 'rounded-card border border-border bg-surface shadow-xs'
      }`}
    >
      <span className="flex size-11 items-center justify-center rounded-full border border-border bg-surface-subtle text-text-muted">
        <Icon name={icon} className="size-5" />
      </span>
      <h2 id={headingId} className="mt-4 text-base font-semibold text-text">
        {title}
      </h2>
      <p className="mt-1.5 max-w-md text-sm text-text-muted">{children}</p>
      {action !== undefined && <div className="mt-6">{action}</div>}
    </section>
  );
}
