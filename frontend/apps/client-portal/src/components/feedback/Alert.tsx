import type { ReactNode } from 'react';

export type AlertTone = 'error' | 'success' | 'info' | 'warning';

const toneClasses: Record<AlertTone, string> = {
  error: 'border-danger text-danger',
  success: 'border-success text-success',
  info: 'border-info text-info',
  warning: 'border-warning text-warning',
};

const toneLabels: Record<AlertTone, string> = {
  error: 'Error',
  success: 'Success',
  info: 'Note',
  warning: 'Warning',
};

/**
 * A message box. Errors are announced to screen readers immediately (role="alert"); other tones
 * politely (role="status"). The tone is also given in words, never by colour alone.
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
  return (
    <div
      role={tone === 'error' ? 'alert' : 'status'}
      className={`rounded-control border-l-4 bg-surface p-4 ${toneClasses[tone]}`}
    >
      <p className="font-semibold">
        <span className="sr-only">{toneLabels[tone]}: </span>
        {title ?? toneLabels[tone]}
      </p>
      {children !== undefined && <div className="mt-1 text-sm text-text">{children}</div>}
      {action !== undefined && <div className="mt-3">{action}</div>}
    </div>
  );
}
