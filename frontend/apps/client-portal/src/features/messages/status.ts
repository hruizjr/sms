import type { IconName } from '@sms/ui';
import type { MessageStatus } from '../../lib/api/client';

export type StatusTone = 'progress' | 'success' | 'warning' | 'danger';

type StatusInfo = { label: string; tone: StatusTone; icon: IconName; explanation: string };

/*
 * How each customer-visible status is shown. The explanations follow the contract
 * (contracts/openapi/public-api-v1.yaml, MessageStatus): no status promises more than it means, and
 * uncertain statuses never claim non-delivery.
 */
export const STATUS_INFO: Record<MessageStatus, StatusInfo> = {
  accepted: {
    label: 'Accepted',
    tone: 'progress',
    icon: 'clock',
    explanation: 'Accepted by the platform; not yet acknowledged by a provider.',
  },
  provider_accepted: {
    label: 'Provider accepted',
    tone: 'progress',
    icon: 'clock',
    explanation:
      'The provider acknowledged the message. This is not a delivery report; the final outcome is still pending.',
  },
  submission_uncertain: {
    label: 'Uncertain',
    tone: 'warning',
    icon: 'question',
    explanation:
      'The provider’s answer was not received, so the outcome is unknown. The message may already have been delivered. It is not resent automatically.',
  },
  delivered: {
    label: 'Delivered',
    tone: 'success',
    icon: 'check',
    explanation: 'The provider reported delivery to the handset.',
  },
  undelivered: {
    label: 'Undelivered',
    tone: 'danger',
    icon: 'x',
    explanation: 'The provider reported that the message was not delivered.',
  },
  unknown: {
    label: 'Unknown',
    tone: 'warning',
    icon: 'question',
    explanation:
      'No final report arrived in time. This is not proof that the message was undelivered; a late report can still change it.',
  },
  rejected: {
    label: 'Rejected',
    tone: 'danger',
    icon: 'x',
    explanation: 'Not sent: the platform or the provider refused the message.',
  },
  failed: {
    label: 'Failed',
    tone: 'danger',
    icon: 'x',
    explanation: 'Not sent: handing the message to the provider failed.',
  },
};

/** Status info, with a safe fallback for statuses added to the contract later. */
export function statusInfo(status: string): StatusInfo {
  return (
    (STATUS_INFO as Partial<Record<string, StatusInfo>>)[status] ?? {
      label: status,
      tone: 'progress',
      icon: 'clock',
      explanation: 'The message is being processed.',
    }
  );
}

const dateTime = new Intl.DateTimeFormat('en-PH', {
  timeZone: 'Asia/Manila',
  year: 'numeric',
  month: 'short',
  day: 'numeric',
  hour: 'numeric',
  minute: '2-digit',
  second: '2-digit',
});

const shortDateTime = new Intl.DateTimeFormat('en-PH', {
  timeZone: 'Asia/Manila',
  year: 'numeric',
  month: 'short',
  day: 'numeric',
  hour: 'numeric',
  minute: '2-digit',
});

/**
 * Formats an RFC 3339 timestamp in Philippine time (AGENTS.md §7.10: Asia/Manila for display).
 * Lists use the short form (to the minute); details show seconds.
 */
export function formatDateTime(iso: string, form: 'full' | 'short' = 'full'): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return `${(form === 'short' ? shortDateTime : dateTime).format(date)} PHT`;
}

export function encodingLabel(encoding: string): string {
  return encoding === 'gsm7' ? 'GSM-7' : encoding === 'ucs2' ? 'UCS-2 (Unicode)' : encoding;
}
