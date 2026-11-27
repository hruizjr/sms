import type { ReactNode } from 'react';
import { classNames } from './class-names';
import { Icon, type IconName } from './Icon';

export type BadgeTone = 'neutral' | 'info' | 'success' | 'warning' | 'danger';

export type BadgeProps = {
  tone?: BadgeTone;
  /** Optional icon before the text. The text always carries the meaning. */
  icon?: IconName;
  className?: string;
  children: ReactNode;
};

const toneClasses: Record<BadgeTone, string> = {
  neutral: 'border-border bg-surface-subtle text-text-muted',
  info: 'border-info bg-info-subtle text-info',
  success: 'border-success bg-success-subtle text-success',
  warning: 'border-warning bg-warning-subtle text-warning',
  danger: 'border-danger bg-danger-subtle text-danger',
};

/** A compact label for a status or attribute: tinted background, border, optional icon and text. */
export function Badge({ tone = 'neutral', icon, className, children }: BadgeProps) {
  return (
    <span
      className={classNames(
        'inline-flex items-center gap-1 rounded-control border px-2 py-0.5 text-xs leading-5 font-medium whitespace-nowrap',
        toneClasses[tone],
        className,
      )}
    >
      {icon !== undefined && <Icon name={icon} className="size-3.5" />}
      {children}
    </span>
  );
}
