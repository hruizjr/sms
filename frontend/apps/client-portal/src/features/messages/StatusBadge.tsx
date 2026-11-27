import { statusInfo, type StatusTone } from './status';

const toneClasses: Record<StatusTone, string> = {
  progress: 'border-info text-info',
  success: 'border-success text-success',
  warning: 'border-warning text-warning',
  danger: 'border-danger text-danger',
};

/** A status shown as icon and text, never colour alone (AGENTS.md §8.9). */
export function StatusBadge({ status }: { status: string }) {
  const info = statusInfo(status);
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-control border bg-surface px-2 py-0.5 text-sm font-medium whitespace-nowrap ${toneClasses[info.tone]}`}
    >
      <span aria-hidden="true">{info.icon}</span>
      {info.label}
    </span>
  );
}
