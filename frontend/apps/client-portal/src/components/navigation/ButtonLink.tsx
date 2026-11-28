import { Icon, type IconName } from '@sms/ui';
import type { ReactNode } from 'react';
import { Link } from '../../lib/router';

const variants = {
  primary: 'bg-primary text-primary-foreground shadow-xs hover:bg-primary-hover',
  secondary: 'border border-border-strong bg-surface text-text shadow-xs hover:bg-surface-subtle',
} as const;

const sizes = {
  sm: 'min-h-8 gap-1.5 px-2.5',
  md: 'min-h-9 gap-2 px-3.5',
} as const;

/** A link that looks like a button (the same variants as Button), for actions such as "Send SMS". */
export function ButtonLink({
  to,
  variant = 'primary',
  size = 'md',
  icon,
  children,
}: {
  to: string;
  variant?: keyof typeof variants;
  size?: keyof typeof sizes;
  icon?: IconName;
  children: ReactNode;
}) {
  return (
    <Link
      to={to}
      className={`inline-flex items-center justify-center rounded-control text-sm font-medium whitespace-nowrap transition-colors ${variants[variant]} ${sizes[size]}`}
    >
      {icon !== undefined && <Icon name={icon} />}
      {children}
    </Link>
  );
}
