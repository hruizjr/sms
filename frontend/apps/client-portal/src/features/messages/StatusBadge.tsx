import { Badge, type BadgeTone } from '@sms/ui';
import { statusInfo, type StatusTone } from './status';

const tones: Record<StatusTone, BadgeTone> = {
  progress: 'info',
  success: 'success',
  warning: 'warning',
  danger: 'danger',
};

/** A status shown as icon and text, never colour alone (AGENTS.md §8.9). */
export function StatusBadge({ status }: { status: string }) {
  const info = statusInfo(status);
  return (
    <Badge tone={tones[info.tone]} icon={info.icon}>
      {info.label}
    </Badge>
  );
}
