import { Badge, Button, Icon } from '@sms/ui';
import { useEffect, useId, useRef, useState } from 'react';
import type { ApiKeyCreated } from '../../lib/api/client';
import { SCOPE_LABELS } from './scopes';

type CopyState = 'idle' | 'copied' | 'failed';

/** The public API's placeholder host (contracts/openapi/public-api-v1.yaml; D-145 is open). */
const API_HOST = 'https://api.example.invalid';

/** A request example with placeholders only: never the real key. */
function curlExample(): string {
  return [
    `curl -X POST ${API_HOST}/v1/messages \\`,
    '  -H "Authorization: Bearer $SMS_API_KEY" \\',
    '  -H "Idempotency-Key: $(uuidgen)" \\',
    '  -H "Content-Type: application/json" \\',
    `  -d '{"to": "+639XXXXXXXXX", "sender_id": "YOUR_SENDER_ID", "body": "Hello from our app"}'`,
  ].join('\n');
}

/**
 * The one-time display of a newly created key: the full value with a copy action, a clear warning,
 * the key's permissions and a request example. The value exists only in this component's props; it
 * disappears when the owner selects Done or leaves the page.
 */
export function NewApiKeyPanel({
  created,
  onDone,
}: {
  created: ApiKeyCreated;
  onDone: () => void;
}) {
  const headingId = useId();
  const keyFieldId = useId();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const keyField = useRef<HTMLInputElement>(null);
  const [copy, setCopy] = useState<CopyState>('idle');

  useEffect(() => {
    headingRef.current?.focus();
  }, []);

  async function copyKey() {
    try {
      if (typeof navigator.clipboard === 'undefined') throw new Error('no clipboard');
      await navigator.clipboard.writeText(created.key);
      setCopy('copied');
    } catch {
      // Let the owner copy it by hand: select the whole key.
      keyField.current?.focus();
      keyField.current?.select();
      setCopy('failed');
    }
  }

  const { api_key: key } = created;
  return (
    <section
      aria-labelledby={headingId}
      className="overflow-hidden rounded-card border border-success bg-surface shadow-sm"
    >
      <div className="flex items-start gap-3 border-b border-border bg-success-subtle px-4 py-3 sm:px-5">
        <Icon name="check" className="mt-0.5 size-5 text-success" />
        <div className="min-w-0">
          <h2
            id={headingId}
            ref={headingRef}
            tabIndex={-1}
            className="text-sm font-semibold text-text focus:outline-none"
          >
            API key “{key.name}” created
          </h2>
          <p className="text-sm text-text">
            Copy the key now and store it in your application’s secret storage.
          </p>
        </div>
      </div>

      <div className="space-y-5 p-4 sm:p-5">
        <div className="space-y-1.5">
          <label htmlFor={keyFieldId} className="block text-sm font-medium text-text">
            Your new API key
          </label>
          <div className="flex flex-col gap-2 sm:flex-row">
            <input
              id={keyFieldId}
              ref={keyField}
              readOnly
              value={created.key}
              autoComplete="off"
              spellCheck={false}
              aria-describedby={`${keyFieldId}-warning`}
              onFocus={(e) => {
                e.currentTarget.select();
              }}
              className="min-w-0 flex-1 rounded-control border border-border-strong bg-surface-subtle px-3 py-2 font-mono text-xs text-text"
            />
            <Button variant="secondary" onClick={() => void copyKey()}>
              <Icon name={copy === 'copied' ? 'check' : 'copy'} />
              {copy === 'copied' ? 'Copied' : 'Copy key'}
            </Button>
          </div>
          <p role="status" className="text-xs text-text-muted">
            {copy === 'copied' && 'The key was copied to your clipboard.'}
            {copy === 'failed' &&
              'Copying is not available in this browser. The key is selected: copy it with your keyboard.'}
          </p>
        </div>

        <div
          id={`${keyFieldId}-warning`}
          className="flex gap-3 rounded-control border border-warning bg-warning-subtle p-3 text-sm text-text"
        >
          <Icon name="alert" className="mt-0.5 size-5 text-warning" />
          <p>
            <strong className="font-semibold">This is the only time the full key is shown.</strong>{' '}
            The platform stores only a hash of it and cannot show it again after you leave this
            page. If you lose it, revoke it and create a new key.
          </p>
        </div>

        <div className="grid gap-5 lg:grid-cols-3">
          <div className="space-y-2">
            <h3 className="text-sm font-semibold text-text">Permissions</h3>
            <ul className="space-y-2">
              {key.scopes.map((scope) => (
                <li key={scope} className="space-y-0.5">
                  <Badge className="font-mono">{scope}</Badge>
                  <p className="text-xs text-text-muted">{SCOPE_LABELS[scope].description}</p>
                </li>
              ))}
            </ul>
          </div>
          <div className="min-w-0 space-y-2 lg:col-span-2">
            <h3 className="text-sm font-semibold text-text">Send a message with this key</h3>
            <pre className="overflow-x-auto rounded-control bg-code p-3 font-mono text-xs leading-relaxed text-code-foreground">
              <code>{curlExample()}</code>
            </pre>
            <p className="text-xs text-text-muted">
              Set <code className="font-mono">SMS_API_KEY</code> to your key. Replace the host with
              your platform’s API address, and use your own recipient and approved sender ID. Use a
              new <code className="font-mono">Idempotency-Key</code> for each message and reuse it
              when retrying the same message.
            </p>
          </div>
        </div>

        <div className="flex justify-end border-t border-border pt-4">
          <Button onClick={onDone}>Done</Button>
        </div>
      </div>
    </section>
  );
}
