import { Icon, type IconName } from '@sms/ui';
import { useId, type ReactNode } from 'react';

/** A named region explaining why a list is empty, with an optional next step. */
export function EmptyState({
  icon,
  title,
  children,
  action,
}: {
  icon: IconName;
  title: string;
  children: ReactNode;
  action?: ReactNode;
}) {
  const headingId = useId();
  return (
    <section
      aria-labelledby={headingId}
      className="flex flex-col items-center rounded-card border border-dashed border-border-strong bg-surface px-6 py-12 text-center"
    >
      <span className="flex size-10 items-center justify-center rounded-full bg-surface-muted text-text-muted">
        <Icon name={icon} className="size-5" />
      </span>
      <h2 id={headingId} className="mt-4 text-sm font-semibold text-text">
        {title}
      </h2>
      <p className="mt-1 max-w-sm text-sm text-text-muted">{children}</p>
      {action !== undefined && <div className="mt-5">{action}</div>}
    </section>
  );
}
