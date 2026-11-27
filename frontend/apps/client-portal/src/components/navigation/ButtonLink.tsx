import { Icon, type IconName } from '@sms/ui';
import type { ReactNode } from 'react';
import { Link } from '../../lib/router';

const variants = {
  primary: 'bg-primary text-primary-foreground shadow-sm hover:bg-primary-hover',
  secondary: 'border border-border-strong bg-surface text-text shadow-sm hover:bg-surface-muted',
} as const;

/** A link that looks like a button, for navigation actions such as "Send SMS". */
export function ButtonLink({
  to,
  variant = 'primary',
  icon,
  children,
}: {
  to: string;
  variant?: keyof typeof variants;
  icon?: IconName;
  children: ReactNode;
}) {
  return (
    <Link
      to={to}
      className={`inline-flex min-h-9 items-center justify-center gap-2 rounded-control px-3.5 text-sm font-medium whitespace-nowrap transition-colors ${variants[variant]}`}
    >
      {icon !== undefined && <Icon name={icon} />}
      {children}
    </Link>
  );
}
