import { Icon, type IconName } from '@sms/ui';
import type { ReactNode } from 'react';

export type AlertTone = 'error' | 'success' | 'info' | 'warning';

const toneClasses: Record<AlertTone, { box: string; icon: string }> = {
  error: { box: 'border-danger/40 bg-danger-subtle', icon: 'text-danger' },
  success: { box: 'border-success/40 bg-success-subtle', icon: 'text-success' },
  info: { box: 'border-info/40 bg-info-subtle', icon: 'text-info' },
  warning: { box: 'border-warning/40 bg-warning-subtle', icon: 'text-warning' },
};

const toneIcons: Record<AlertTone, IconName> = {
  error: 'alert',
  success: 'check',
  info: 'info',
  warning: 'alert',
};

const toneLabels: Record<AlertTone, string> = {
  error: 'Error',
  success: 'Success',
  info: 'Note',
  warning: 'Warning',
};

/**
 * A message box. Errors are announced to screen readers immediately (role="alert"); other tones
 * politely (role="status"). The tone is also given in words and an icon, never by colour alone.
 */
export function Alert({
  tone,
  title,
  children,
  action,
}: {
  tone: AlertTone;
  title?: string;
  children?: ReactNode;
  action?: ReactNode;
}) {
  const classes = toneClasses[tone];
  return (
    <div
      role={tone === 'error' ? 'alert' : 'status'}
      className={`flex gap-3 rounded-card border px-4 py-3 text-sm text-text ${classes.box}`}
    >
      <Icon name={toneIcons[tone]} className={`mt-0.5 size-5 ${classes.icon}`} />
      <div className="min-w-0 flex-1">
        <p className="font-semibold">
          <span className="sr-only">{toneLabels[tone]}: </span>
          {title ?? toneLabels[tone]}
        </p>
        {children !== undefined && <div className="mt-0.5 text-text">{children}</div>}
        {action !== undefined && <div className="mt-3">{action}</div>}
      </div>
    </div>
  );
}
