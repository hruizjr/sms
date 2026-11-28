import type { ComponentPropsWithRef } from 'react';
import { classNames } from './class-names';

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'danger-secondary' | 'ghost';
export type ButtonSize = 'sm' | 'md';

export type ButtonProps = {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Shows a progress indicator and disables the button, preventing repeated activation. */
  loading?: boolean;
} & ComponentPropsWithRef<'button'>;

const variantClasses: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-primary-foreground shadow-xs hover:bg-primary-hover',
  secondary: 'border border-border-strong bg-surface text-text shadow-xs hover:bg-surface-subtle',
  danger: 'bg-danger text-danger-foreground shadow-xs hover:bg-danger-hover',
  /** A destructive action that opens a confirmation (for example "Revoke" in a table row). */
  'danger-secondary':
    'border border-border-strong bg-surface text-danger shadow-xs hover:bg-danger-subtle',
  ghost: 'text-text-muted hover:bg-surface-muted hover:text-text',
};

const sizeClasses: Record<ButtonSize, string> = {
  sm: 'min-h-8 gap-1.5 px-2.5 text-sm',
  md: 'min-h-9 gap-2 px-3.5 text-sm',
};

/**
 * A native button, so keyboard operation (Tab, Enter, Space) and focus come from the browser.
 * Defaults to type="button" so that it never submits a form by accident.
 */
export function Button({
  variant = 'primary',
  size = 'md',
  loading = false,
  disabled = false,
  type = 'button',
  className,
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      {...props}
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={classNames(
        'inline-flex items-center justify-center rounded-control font-medium whitespace-nowrap transition-colors',
        'disabled:cursor-not-allowed disabled:opacity-60',
        variantClasses[variant],
        sizeClasses[size],
        className,
      )}
    >
      {loading && (
        <span
          aria-hidden="true"
          data-testid="button-spinner"
          className="size-4 animate-spin rounded-full border-2 border-current border-t-transparent"
        />
      )}
      {children}
    </button>
  );
}
