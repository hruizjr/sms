import { Button, Card } from '@sms/ui';
import { useEffect, useRef, useState, type SyntheticEvent } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { Skeleton } from '../../components/feedback/Skeleton';
import { SelectField, TextAreaField, TextField } from '../../components/form/Field';
import {
  ApiError,
  NetworkError,
  type MessageCreateRequest,
  type MessageEstimate,
} from '../../lib/api/client';
import { Link, navigate } from '../../lib/router';
import { hasPermission, useSession, useSignedInSession } from '../auth/session';
import { encodingLabel } from './status';

/** Philippine mobile numbers only (destination policy D-043), in E.164 form. */
const PH_MOBILE = /^\+639\d{9}$/;
const MAX_CHARACTERS = 1600;
export const ESTIMATE_DELAY_MS = 300;

type Fields = { to: string; senderId: string; body: string };
type FieldErrors = Partial<Record<keyof Fields, string>>;

type SenderIds = { status: 'loading' } | { status: 'error' } | { status: 'loaded'; ids: string[] };

type Estimate =
  | { status: 'idle' }
  | { status: 'pending' }
  | { status: 'ready'; value: MessageEstimate }
  | { status: 'error' };

/** The last estimate result, for the text it was computed for. */
type EstimateResult = { body: string; result: { value: MessageEstimate } | { error: true } };

function needsEstimate(body: string): boolean {
  return body !== '' && characterCount(body) <= MAX_CHARACTERS;
}

/** Removes spaces, dashes and brackets people type in phone numbers. */
function normalizeNumber(value: string): string {
  return value.replace(/[\s\-()]/g, '');
}

function characterCount(text: string): number {
  return Array.from(text).length;
}

function validate(fields: Fields): FieldErrors {
  const errors: FieldErrors = {};
  const to = normalizeNumber(fields.to);
  if (to === '') {
    errors.to = 'Enter the recipient’s mobile number.';
  } else if (!PH_MOBILE.test(to)) {
    errors.to =
      'Enter a Philippine mobile number in international format, for example +639171234567.';
  }
  if (fields.senderId === '') {
    errors.senderId = 'Choose a sender ID.';
  }
  const length = characterCount(fields.body);
  if (fields.body.trim() === '') {
    errors.body = 'Enter the message text.';
  } else if (length > MAX_CHARACTERS) {
    errors.body = `The message is ${String(length)} characters; the maximum is ${String(MAX_CHARACTERS)}.`;
  }
  return errors;
}

function defaultKey(): string {
  return crypto.randomUUID();
}

export type ComposePageProps = {
  /** Generates idempotency keys; injectable for tests. */
  newIdempotencyKey?: () => string;
};

export function ComposePage({ newIdempotencyKey = defaultKey }: ComposePageProps) {
  const session = useSignedInSession();
  useEffect(() => {
    document.title = 'Send SMS · Client Portal';
  }, []);

  return (
    <div className="space-y-6">
      <p>
        <Link to="/messages" className="text-sm font-medium text-primary underline">
          ← All messages
        </Link>
      </p>
      <h1 className="text-2xl font-semibold sm:text-3xl">Send SMS</h1>
      {hasPermission(session, 'messages:send') ? (
        <ComposeForm newIdempotencyKey={newIdempotencyKey} />
      ) : (
        <Alert tone="info" title="Your role cannot send messages">
          You can view your organization’s messages. Ask an owner of your organization’s account if
          you need to send SMS.
        </Alert>
      )}
    </div>
  );
}

function ComposeForm({ newIdempotencyKey }: { newIdempotencyKey: () => string }) {
  const { api } = useSession();
  const [fields, setFields] = useState<Fields>({ to: '', senderId: '', body: '' });
  const [errors, setErrors] = useState<FieldErrors>({});
  const [senderIds, setSenderIds] = useState<SenderIds>({ status: 'loading' });
  const [senderReload, setSenderReload] = useState(0);
  const [estimateResult, setEstimateResult] = useState<EstimateResult | undefined>(undefined);
  const [reviewing, setReviewing] = useState(false);
  const [sending, setSending] = useState(false);
  const [failure, setFailure] = useState<{ title: string; detail: string } | undefined>(undefined);

  // One idempotency key per intended send: reused on every retry of the same content, replaced
  // when the content changes. A repeated click or a retry after a lost response therefore cannot
  // create a second message (ADR-0018).
  const intent = useRef<{ content: string; key: string } | undefined>(undefined);
  const inFlight = useRef(false);

  useEffect(() => {
    let active = true;
    api.listSenderIds().then(
      (ids) => {
        if (!active) return;
        setSenderIds({ status: 'loaded', ids });
        if (ids.length === 1 && ids[0] !== undefined) {
          const only = ids[0];
          setFields((f) => (f.senderId === '' ? { ...f, senderId: only } : f));
        }
      },
      () => {
        if (active) setSenderIds({ status: 'error' });
      },
    );
    return () => {
      active = false;
    };
  }, [api, senderReload]);

  // Character count and segment estimate from the platform's shared segmentation (debounced).
  useEffect(() => {
    const body = fields.body;
    if (!needsEstimate(body)) return;
    let active = true;
    const timer = window.setTimeout(() => {
      api.estimateMessage(body).then(
        (value) => {
          if (active) setEstimateResult({ body, result: { value } });
        },
        () => {
          if (active) setEstimateResult({ body, result: { error: true } });
        },
      );
    }, ESTIMATE_DELAY_MS);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [api, fields.body]);
  const estimate: Estimate = !needsEstimate(fields.body)
    ? { status: 'idle' }
    : estimateResult?.body !== fields.body
      ? { status: 'pending' }
      : 'value' in estimateResult.result
        ? { status: 'ready', value: estimateResult.result.value }
        : { status: 'error' };

  function update(name: keyof Fields, value: string) {
    setFields((f) => ({ ...f, [name]: value }));
    setErrors((e) => ({ ...e, [name]: undefined }));
  }

  function request(): MessageCreateRequest {
    return { to: normalizeNumber(fields.to), sender_id: fields.senderId, body: fields.body };
  }

  function onReview(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const found = validate(fields);
    setErrors(found);
    setFailure(undefined);
    if (Object.keys(found).length === 0) setReviewing(true);
  }

  async function send() {
    if (inFlight.current) return; // a second click while the first request is in flight
    inFlight.current = true;
    setSending(true);
    setFailure(undefined);
    const body = request();
    const content = JSON.stringify(body);
    if (intent.current?.content !== content) {
      intent.current = { content, key: newIdempotencyKey() };
    }
    try {
      const { message } = await api.createMessage(body, intent.current.key);
      intent.current = undefined;
      navigate(`/messages/${message.id}?sent=1`);
    } catch (error: unknown) {
      handleSendError(error);
    } finally {
      inFlight.current = false;
      setSending(false);
    }
  }

  function handleSendError(error: unknown) {
    if (error instanceof ApiError) {
      const detail = error.problem?.detail ?? 'The message was not accepted.';
      switch (error.problemType) {
        case 'validation-error': {
          const found: FieldErrors = {};
          for (const e of error.problem?.errors ?? []) {
            if (e.pointer === '/to') found.to = e.detail;
            if (e.pointer === '/sender_id') found.senderId = e.detail;
            if (e.pointer === '/body') found.body = e.detail;
          }
          setErrors(found);
          setReviewing(false);
          setFailure({ title: 'Check the highlighted fields', detail });
          return;
        }
        case 'sender-id-not-approved':
        case 'destination-not-supported':
        case 'message-too-long':
          setReviewing(false);
          setFailure({ title: error.problem?.title ?? 'Message not accepted', detail });
          return;
        case 'idempotency-request-in-progress':
          setFailure({
            title: 'Your message is still being processed',
            detail: 'Wait a moment, then select Send message again. No duplicate will be created.',
          });
          return;
        case 'idempotency-key-reused':
          intent.current = undefined;
          setFailure({ title: 'Please send again', detail });
          return;
        case 'forbidden':
          setReviewing(false);
          setFailure({ title: 'You cannot send messages', detail });
          return;
      }
      if (error.status === 401) return; // the session ended; the sign-in page is shown centrally
    }
    // A lost response or a server error: the message may or may not have been accepted. Keep the
    // same key so that sending again is safe.
    setFailure({
      title: error instanceof NetworkError ? 'Connection problem' : 'Something went wrong',
      detail:
        'We could not confirm whether your message was accepted. Select Send message again: it is sent with the same request key, so it cannot create a duplicate.',
    });
  }

  if (senderIds.status === 'loading') {
    return <Skeleton lines={3} label="Loading sender IDs…" />;
  }
  if (senderIds.status === 'error') {
    return (
      <Alert
        tone="error"
        title="Could not load your sender IDs"
        action={
          <Button
            variant="secondary"
            size="sm"
            onClick={() => {
              setSenderIds({ status: 'loading' });
              setSenderReload((n) => n + 1);
            }}
          >
            Try again
          </Button>
        }
      >
        The form needs your approved sender IDs. Try again.
      </Alert>
    );
  }
  if (senderIds.ids.length === 0) {
    return (
      <Alert tone="info" title="No approved sender ID">
        Your organization has no approved sender ID yet, so it cannot send SMS. Contact support to
        request one.
      </Alert>
    );
  }

  const characters = characterCount(fields.body);
  const segmentsText =
    estimate.status === 'ready'
      ? `${encodingLabel(estimate.value.encoding)} · ${String(estimate.value.segment_count)} ${
          estimate.value.segment_count === 1 ? 'segment' : 'segments'
        }`
      : estimate.status === 'pending'
        ? 'Calculating segments…'
        : estimate.status === 'error'
          ? 'Segment estimate unavailable'
          : '';

  return (
    <div className="grid gap-6 lg:grid-cols-3">
      <Card className="lg:col-span-2">
        <form noValidate onSubmit={onReview} className="space-y-4">
          {failure !== undefined && (
            <Alert tone="error" title={failure.title}>
              {failure.detail}
            </Alert>
          )}
          <TextField
            label="Recipient mobile number"
            type="tel"
            name="to"
            inputMode="tel"
            autoComplete="off"
            placeholder="+639171234567"
            value={fields.to}
            onChange={(e) => {
              update('to', e.target.value);
            }}
            disabled={reviewing}
            error={errors.to}
            hint="One Philippine mobile number in international format (+639…)."
          />
          <SelectField
            label="Sender ID"
            name="sender_id"
            value={fields.senderId}
            onChange={(e) => {
              update('senderId', e.target.value);
            }}
            disabled={reviewing}
            error={errors.senderId}
            hint="Only your organization’s approved sender IDs are listed."
          >
            <option value="">Choose a sender ID</option>
            {senderIds.ids.map((id) => (
              <option key={id} value={id}>
                {id}
              </option>
            ))}
          </SelectField>
          <TextAreaField
            label="Message"
            name="body"
            rows={6}
            value={fields.body}
            onChange={(e) => {
              update('body', e.target.value);
            }}
            disabled={reviewing}
            error={errors.body}
            hint={
              <span aria-live="polite" data-testid="message-counter">
                {characters} / {MAX_CHARACTERS} characters
                {segmentsText !== '' && ` · ${segmentsText}`}
              </span>
            }
          />
          {!reviewing && <Button type="submit">Review message</Button>}
        </form>
      </Card>

      {reviewing && (
        <Card heading="Review and send" className="lg:col-span-1">
          <dl className="space-y-2 text-text">
            <div>
              <dt className="text-sm text-text-muted">Recipients</dt>
              <dd>1 ({normalizeNumber(fields.to)})</dd>
            </div>
            <div>
              <dt className="text-sm text-text-muted">Sender ID</dt>
              <dd>{fields.senderId}</dd>
            </div>
            <div>
              <dt className="text-sm text-text-muted">Segments</dt>
              <dd>
                {estimate.status === 'ready'
                  ? `${String(estimate.value.segment_count)} (${encodingLabel(estimate.value.encoding)})`
                  : 'Calculated by the platform on submission'}
              </dd>
            </div>
            <div>
              <dt className="text-sm text-text-muted">Estimated cost</dt>
              <dd>Not available: billing is not enabled yet.</dd>
            </div>
          </dl>
          <p className="mt-4 text-sm text-text-muted">
            Acceptance means the platform stored your message for sending; delivery is reported
            later by the provider.
          </p>
          <div className="mt-4 flex flex-wrap gap-2">
            <Button loading={sending} onClick={() => void send()}>
              {sending ? 'Sending…' : 'Send message'}
            </Button>
            <Button
              variant="secondary"
              disabled={sending}
              onClick={() => {
                setReviewing(false);
              }}
            >
              Edit
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}
