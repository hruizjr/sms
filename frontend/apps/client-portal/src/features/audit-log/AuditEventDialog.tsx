import { Badge, Button, Dialog } from '@sms/ui';
import { CopyButton } from '../../components/actions/CopyButton';
import { DescriptionItem, DescriptionList } from '../../components/data/DescriptionList';
import type { AuditEvent } from '../../lib/api/client';
import { formatDateTime } from '../messages/status';
import {
  actorText,
  dedupIntervalText,
  eventTitle,
  outcomeBadge,
  reasonText,
  targetText,
} from './events';

/** Every recorded field of one audit event, read-only. */
export function AuditEventDialog({
  event,
  onClose,
}: {
  event: AuditEvent | undefined;
  onClose: () => void;
}) {
  return (
    <Dialog
      open={event !== undefined}
      onClose={onClose}
      title={event !== undefined ? eventTitle(event) : 'Audit event'}
      description={
        event !== undefined ? formatDateTime(event.occurred_at) : 'A recorded security event.'
      }
      footer={
        <Button variant="secondary" onClick={onClose} data-autofocus>
          Close
        </Button>
      }
    >
      {event !== undefined && <EventDetails event={event} />}
    </Dialog>
  );
}

function EventDetails({ event }: { event: AuditEvent }) {
  const outcome = outcomeBadge(event);
  const { details } = event;
  const reason = reasonText(event);
  const target = targetText(event);
  return (
    <DescriptionList>
      <DescriptionItem term="Outcome">
        <Badge tone={outcome.tone} icon={outcome.icon}>
          {outcome.label}
        </Badge>
      </DescriptionItem>
      {reason !== '' && <DescriptionItem term="Reason">{reason}</DescriptionItem>}
      <DescriptionItem term="By">
        <span className="break-words">{actorText(event)}</span>
      </DescriptionItem>
      {target !== '' && (
        <DescriptionItem term="Concerning">
          <span className="break-words">{target}</span>
          {event.target?.type === 'api_key' && event.target.id !== null && (
            <span className="block font-mono text-xs text-text-muted">
              Key ID {event.target.id}
            </span>
          )}
        </DescriptionItem>
      )}
      {details.sender_id !== undefined && (
        <DescriptionItem term="Sender ID">
          <span className="font-mono text-xs break-words">{details.sender_id}</span>
        </DescriptionItem>
      )}
      {details.api_key_scopes !== undefined && (
        <DescriptionItem term="Permissions">
          <span className="flex flex-wrap gap-1">
            {details.api_key_scopes.map((scope) => (
              <Badge key={scope} className="font-mono">
                {scope}
              </Badge>
            ))}
          </span>
        </DescriptionItem>
      )}
      {details.required_permission !== undefined && (
        <DescriptionItem term="Required permission">
          <span className="font-mono text-xs">{details.required_permission}</span>
          {details.role !== undefined && (
            <span className="text-text-muted"> · role {details.role}</span>
          )}
        </DescriptionItem>
      )}
      {details.role !== undefined && details.required_permission === undefined && (
        <DescriptionItem term="Role">{details.role}</DescriptionItem>
      )}
      {details.dedup_interval_seconds !== undefined && (
        <DescriptionItem term="Repeats">
          Repeated refusals of this user and action within{' '}
          {dedupIntervalText(details.dedup_interval_seconds)} are recorded once.
        </DescriptionItem>
      )}
      {details.locked_until !== undefined && (
        <DescriptionItem term="Locked until">
          {formatDateTime(details.locked_until)}
        </DescriptionItem>
      )}
      <DescriptionItem term="IP address">
        <span className="font-mono text-xs">{event.source_ip ?? 'Not recorded'}</span>
      </DescriptionItem>
      <DescriptionItem term="Browser or client">
        <span className="text-xs break-words text-text-muted">
          {event.user_agent ?? 'Not recorded'}
        </span>
      </DescriptionItem>
      <DescriptionItem term="Request ID">
        {event.request_id === null ? (
          'Not recorded'
        ) : (
          <span className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-xs break-all">{event.request_id}</span>
            <CopyButton value={event.request_id} label="Copy request ID" />
          </span>
        )}
      </DescriptionItem>
      <DescriptionItem term="Event ID">
        <span className="font-mono text-xs break-all text-text-muted">{event.id}</span>
      </DescriptionItem>
    </DescriptionList>
  );
}
