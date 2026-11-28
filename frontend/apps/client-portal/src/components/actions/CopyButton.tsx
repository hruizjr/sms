import { Icon } from '@sms/ui';
import { useEffect, useState } from 'react';

type CopyState = 'idle' | 'copied' | 'failed';

/** How long "Copied" is shown before the button returns to its normal label. */
export const COPIED_RESET_MS = 2000;

/**
 * A small icon button that copies a non-secret value (such as a message ID) to the clipboard. The
 * result is announced politely; when the browser cannot copy, the button says so and the value stays
 * visible next to it for manual copying.
 */
export function CopyButton({ value, label }: { value: string; label: string }) {
  const [state, setState] = useState<CopyState>('idle');

  useEffect(() => {
    if (state === 'idle') return;
    const timer = window.setTimeout(() => {
      setState('idle');
    }, COPIED_RESET_MS);
    return () => {
      window.clearTimeout(timer);
    };
  }, [state]);

  async function copy() {
    try {
      if (typeof navigator.clipboard === 'undefined') throw new Error('no clipboard');
      await navigator.clipboard.writeText(value);
      setState('copied');
    } catch {
      setState('failed');
    }
  }

  return (
    <span className="inline-flex items-center gap-1.5">
      <button
        type="button"
        onClick={() => void copy()}
        className="inline-flex size-7 shrink-0 items-center justify-center rounded-control text-text-muted transition-colors hover:bg-surface-muted hover:text-text"
      >
        <Icon name={state === 'copied' ? 'check' : 'copy'} className="size-3.5" />
        <span className="sr-only">{label}</span>
      </button>
      <span role="status" className="text-xs text-text-muted">
        {state === 'copied' && 'Copied'}
        {state === 'failed' && 'Copy not available'}
      </span>
    </span>
  );
}
