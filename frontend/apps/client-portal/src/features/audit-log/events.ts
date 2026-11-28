import type { BadgeTone, IconName } from '@sms/ui';
import type { AuditAction, AuditEvent, AuditOutcome } from '../../lib/api/client';

/*
 * Plain-language descriptions of security audit events (contracts/openapi/portal-api-v1.yaml,
 * AuditEvent). Only fields the API returns are shown; nothing is inferred.
 */

/** Filter labels, in the order of the filter menu. */
export const ACTION_LABELS: Record<AuditAction, string> = {
  'portal.sign_in': 'Sign-in',
  'portal.sign_out': 'Sign-out',
  'portal.account_locked': 'Account lock',
  'api_key.create': 'API key creation',
  'api_key.revoke': 'API key revocation',
  'api_key.list': 'API key list access',
  'audit_log.list': 'Audit log access',
  'portal_user.create': 'User creation',
  'portal_user.unlock': 'Account unlock',
  'portal_user.sessions_revoke': 'Session revocation',
};

export const ACTIONS = Object.keys(ACTION_LABELS) as AuditAction[];

export const OUTCOME_INFO: Record<
  AuditOutcome,
  { label: string; tone: BadgeTone; icon: IconName }
> = {
  success: { label: 'Success', tone: 'success', icon: 'check' },
  failure: { label: 'Failure', tone: 'danger', icon: 'alert' },
  denied: { label: 'Denied', tone: 'warning', icon: 'x' },
  // A security control was applied (an account lock): the stored outcome, shown as it is.
  enforced: { label: 'Enforced', tone: 'warning', icon: 'lock' },
};

export const OUTCOMES = Object.keys(OUTCOME_INFO) as AuditOutcome[];

/** The badge of an event: exactly its stored outcome (an account lock is stored as "enforced"). */
export function outcomeBadge(event: AuditEvent): {
  label: string;
  tone: BadgeTone;
  icon: IconName;
} {
  return OUTCOME_INFO[event.outcome];
}

/** A one-line title of what happened, by action and outcome. */
export function eventTitle(event: AuditEvent): string {
  const { action, outcome } = event;
  switch (action) {
    case 'portal.sign_in':
      return outcome === 'success'
        ? 'Signed in'
        : outcome === 'denied'
          ? 'Sign-in refused'
          : 'Sign-in failed';
    case 'portal.sign_out':
      return 'Signed out';
    case 'portal.account_locked':
      return 'Account locked';
    case 'api_key.create':
      return outcome === 'success'
        ? 'API key created'
        : outcome === 'denied'
          ? 'API key creation refused'
          : 'API key creation failed';
    case 'api_key.revoke':
      return outcome === 'success' ? 'API key revoked' : 'API key revocation refused';
    case 'api_key.list':
      return 'API key list refused';
    case 'audit_log.list':
      return 'Audit log access refused';
    case 'portal_user.create':
      return 'User created';
    case 'portal_user.unlock':
      return 'Account unlocked';
    case 'portal_user.sessions_revoke':
      return 'Sessions ended';
  }
}

const REASONS: Record<string, string> = {
  invalid_password: 'The password was wrong',
  account_locked: 'The account was locked',
  organization_not_active: 'The organization is not active',
  insufficient_role: 'The user’s role does not allow it',
  too_many_attempts: 'Too many failed attempts in a row',
};

/** The reason in plain language ("" when the event has none). */
export function reasonText(event: AuditEvent): string {
  const reason = event.details.reason;
  if (reason === undefined) return '';
  return REASONS[reason] ?? reason;
}

/** A short secondary line: the key concerned, the user administered, or the reason. */
export function eventSummary(event: AuditEvent): string {
  if (event.action.startsWith('portal_user.') && event.target?.email) return event.target.email;
  const name = event.details.api_key_name;
  const reason = reasonText(event);
  if (name !== undefined) return reason === '' ? `“${name}”` : `“${name}” · ${reason}`;
  return reason;
}

/** "5 minutes" (or seconds) for a deduplication interval. */
export function dedupIntervalText(seconds: number): string {
  if (seconds % 60 !== 0) return seconds === 1 ? '1 second' : `${String(seconds)} seconds`;
  const minutes = seconds / 60;
  return minutes === 1 ? '1 minute' : `${String(minutes)} minutes`;
}

/** Who acted: the user's email address, "Not signed in", or "System". */
export function actorText(event: AuditEvent): string {
  const { actor } = event;
  if (actor.email !== null) return actor.email;
  if (actor.type === 'anonymous') return 'Not signed in';
  if (actor.type === 'system') {
    return event.details.via !== undefined ? `System (${event.details.via})` : 'System';
  }
  return actor.id ?? 'Unknown user';
}

/** What the event concerned, in words ("" when nothing in particular). */
export function targetText(event: AuditEvent): string {
  const { target } = event;
  if (target === null) return '';
  switch (target.type) {
    case 'portal_user':
      return target.email ?? 'A portal user';
    case 'portal_session':
      return 'Portal session';
    case 'api_key':
      return event.details.api_key_name !== undefined
        ? `API key “${event.details.api_key_name}”`
        : 'API key';
    default:
      return target.type;
  }
}

const MANILA_OFFSET = '+08:00'; // Philippine Time has no daylight saving time

/**
 * The UTC instant at which a calendar date (YYYY-MM-DD) starts in Philippine Time, as RFC 3339, or
 * undefined for an invalid date. dayOffset 1 gives the start of the next day (an exclusive end).
 */
export function manilaDayStart(date: string, dayOffset = 0): string | undefined {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) return undefined;
  const start = new Date(`${date}T00:00:00${MANILA_OFFSET}`);
  if (Number.isNaN(start.getTime())) return undefined;
  return new Date(start.getTime() + dayOffset * 86_400_000).toISOString();
}
