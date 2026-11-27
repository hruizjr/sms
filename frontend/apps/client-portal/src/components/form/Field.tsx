import { useId, type ComponentPropsWithoutRef, type ReactNode } from 'react';

/*
 * Labelled form controls: every control has a visible label, an optional hint and an error message,
 * both associated with the control (aria-describedby), and aria-invalid when in error
 * (AGENTS.md §8.9).
 */

const controlClasses =
  'block w-full rounded-control border bg-surface px-3 py-2 text-base text-text shadow-sm transition-colors placeholder:text-text-muted hover:border-text-muted focus:border-focus disabled:cursor-not-allowed disabled:bg-surface-subtle disabled:text-text-muted sm:text-sm';

type FieldFrameProps = {
  label: string;
  hint?: ReactNode;
  error?: string;
  children: (ids: { id: string; describedBy: string | undefined; invalid: boolean }) => ReactNode;
};

function FieldFrame({ label, hint, error, children }: FieldFrameProps) {
  const id = useId();
  const hintId = `${id}-hint`;
  const errorId = `${id}-error`;
  const describedBy =
    [hint !== undefined ? hintId : '', error !== undefined ? errorId : '']
      .filter(Boolean)
      .join(' ') || undefined;
  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="block text-sm font-medium text-text">
        {label}
      </label>
      {children({ id, describedBy, invalid: error !== undefined })}
      {hint !== undefined && (
        <p id={hintId} className="text-xs text-text-muted">
          {hint}
        </p>
      )}
      {error !== undefined && (
        <p id={errorId} className="text-xs font-medium text-danger">
          {error}
        </p>
      )}
    </div>
  );
}

function borderFor(invalid: boolean): string {
  return invalid ? 'border-danger' : 'border-border-strong';
}

type Common = { label: string; hint?: ReactNode; error?: string };

export function TextField({
  label,
  hint,
  error,
  className,
  ...props
}: Common & ComponentPropsWithoutRef<'input'>) {
  return (
    <FieldFrame label={label} hint={hint} error={error}>
      {({ id, describedBy, invalid }) => (
        <input
          {...props}
          id={id}
          aria-describedby={describedBy}
          aria-invalid={invalid || undefined}
          className={`${controlClasses} ${borderFor(invalid)} ${className ?? ''}`}
        />
      )}
    </FieldFrame>
  );
}

export function TextAreaField({
  label,
  hint,
  error,
  className,
  ...props
}: Common & ComponentPropsWithoutRef<'textarea'>) {
  return (
    <FieldFrame label={label} hint={hint} error={error}>
      {({ id, describedBy, invalid }) => (
        <textarea
          {...props}
          id={id}
          aria-describedby={describedBy}
          aria-invalid={invalid || undefined}
          className={`${controlClasses} ${borderFor(invalid)} ${className ?? ''}`}
        />
      )}
    </FieldFrame>
  );
}

export function SelectField({
  label,
  hint,
  error,
  className,
  children,
  ...props
}: Common & ComponentPropsWithoutRef<'select'>) {
  return (
    <FieldFrame label={label} hint={hint} error={error}>
      {({ id, describedBy, invalid }) => (
        <select
          {...props}
          id={id}
          aria-describedby={describedBy}
          aria-invalid={invalid || undefined}
          className={`${controlClasses} ${borderFor(invalid)} ${className ?? ''}`}
        >
          {children}
        </select>
      )}
    </FieldFrame>
  );
}
