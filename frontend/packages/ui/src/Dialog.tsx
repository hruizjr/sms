import { useEffect, useId, useRef, type ReactNode } from 'react';
import { classNames } from './class-names';

export type DialogProps = {
  /** Whether the dialog is shown. */
  open: boolean;
  /** Called when the user dismisses the dialog (Escape). Not called while `dismissible` is false. */
  onClose: () => void;
  title: string;
  /** Short explanation under the title; it also describes the dialog to assistive technology. */
  description?: ReactNode;
  /** Buttons, right-aligned in the footer. */
  footer?: ReactNode;
  /** When false (for example while a request is in flight), Escape does not close the dialog. */
  dismissible?: boolean;
  children?: ReactNode;
};

/**
 * A modal dialog on the native <dialog> element opened with showModal(): focus moves into it (to
 * the element marked data-autofocus, else the first control); the browser keeps keyboard focus
 * inside it, makes the rest of the page inert, closes it with Escape and returns focus to the
 * element that opened it (AGENTS.md §8.9).
 */
export function Dialog({
  open,
  onClose,
  title,
  description,
  footer,
  dismissible = true,
  children,
}: DialogProps) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  const descriptionId = useId();

  useEffect(() => {
    const dialog = ref.current;
    if (dialog === null) return;
    if (open && !dialog.open) {
      dialog.showModal();
      // Focus the element marked data-autofocus, else the first control (what browsers do too).
      (
        dialog.querySelector<HTMLElement>('[data-autofocus]') ??
        dialog.querySelector<HTMLElement>('input, select, textarea, button')
      )?.focus();
    }
    if (!open && dialog.open) dialog.close();
  }, [open]);

  // Escape fires "cancel": keep the dialog open and let the owner decide.
  useEffect(() => {
    const dialog = ref.current;
    if (dialog === null) return;
    function onCancel(event: Event) {
      event.preventDefault();
      if (dismissible) onClose();
    }
    dialog.addEventListener('cancel', onCancel);
    return () => {
      dialog.removeEventListener('cancel', onCancel);
    };
  }, [dismissible, onClose]);

  return (
    <dialog
      ref={ref}
      aria-labelledby={titleId}
      aria-describedby={description !== undefined ? descriptionId : undefined}
      className={classNames(
        'm-auto w-full max-w-lg rounded-card border border-border bg-surface p-0 text-text shadow-xl',
        'backdrop:bg-text/50',
      )}
    >
      {open && (
        <div className="flex max-h-dvh flex-col">
          <div className="space-y-1 px-5 pt-5">
            <h2 id={titleId} className="text-base font-semibold text-text">
              {title}
            </h2>
            {description !== undefined && (
              <div id={descriptionId} className="text-sm text-text-muted">
                {description}
              </div>
            )}
          </div>
          {children !== undefined && (
            <div className="overflow-y-auto px-5 pt-4 text-sm">{children}</div>
          )}
          {footer !== undefined && (
            <div className="mt-5 flex flex-col-reverse gap-2 border-t border-border bg-surface-subtle px-5 py-3 sm:flex-row sm:justify-end">
              {footer}
            </div>
          )}
        </div>
      )}
    </dialog>
  );
}
