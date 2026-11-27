import { Button, Dialog } from '@sms/ui';
import { useId, useRef, useState, type SyntheticEvent } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { TextField } from '../../components/form/Field';
import { ApiError, type ApiKeyCreated, type ApiKeyScope } from '../../lib/api/client';
import { useSession } from '../auth/session';
import { SCOPE_LABELS, SCOPES, apiKeyErrorMessage } from './scopes';

const MAX_NAME = 100;
const DEFAULT_SCOPES: ApiKeyScope[] = ['messages:send', 'messages:read'];

type Errors = { name?: string; scopes?: string };

/** The create form: a name and the permissions the key grants. */
export function CreateApiKeyDialog({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: (created: ApiKeyCreated) => void;
}) {
  const { api } = useSession();
  const formId = useId();
  const scopesId = useId();
  const [name, setName] = useState('');
  const [scopes, setScopes] = useState<ApiKeyScope[]>(DEFAULT_SCOPES);
  const [errors, setErrors] = useState<Errors>({});
  const [failure, setFailure] = useState<string | undefined>(undefined);
  const [saving, setSaving] = useState(false);
  const inFlight = useRef(false);

  // Start every opening with an empty form.
  const [wasOpen, setWasOpen] = useState(open);
  if (wasOpen !== open) {
    setWasOpen(open);
    if (open) {
      setName('');
      setScopes(DEFAULT_SCOPES);
      setErrors({});
      setFailure(undefined);
    }
  }

  function validate(): Errors {
    const found: Errors = {};
    const trimmed = name.trim();
    if (trimmed === '') found.name = 'Enter a name for the key.';
    else if (Array.from(trimmed).length > MAX_NAME)
      found.name = `Use at most ${String(MAX_NAME)} characters.`;
    if (scopes.length === 0) found.scopes = 'Choose at least one permission.';
    return found;
  }

  async function onSubmit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    if (inFlight.current) return; // a second click while the first request is in flight
    const found = validate();
    setErrors(found);
    setFailure(undefined);
    if (Object.keys(found).length > 0) return;
    inFlight.current = true;
    setSaving(true);
    try {
      // Keep the scopes in a fixed order, whatever order they were ticked in.
      const result = await api.createApiKey({
        name: name.trim(),
        scopes: SCOPES.filter((s) => scopes.includes(s)),
      });
      onCreated(result);
    } catch (error: unknown) {
      if (error instanceof ApiError && error.problemType === 'validation-error') {
        const fromServer: Errors = {};
        for (const e of error.problem?.errors ?? []) {
          if (e.pointer === '/name') fromServer.name = e.detail;
          if (e.pointer === '/scopes') fromServer.scopes = e.detail;
        }
        setErrors(fromServer);
        setFailure('Check the highlighted fields.');
      } else if (!(error instanceof ApiError && error.status === 401)) {
        setFailure(apiKeyErrorMessage(error, 'The key could not be created. Try again.'));
      }
    } finally {
      inFlight.current = false;
      setSaving(false);
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      dismissible={!saving}
      title="Create API key"
      description="The full key is shown once, right after it is created."
      footer={
        <>
          <Button variant="secondary" disabled={saving} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" form={formId} loading={saving}>
            {saving ? 'Creating…' : 'Create key'}
          </Button>
        </>
      }
    >
      <form id={formId} noValidate onSubmit={(e) => void onSubmit(e)} className="space-y-5">
        {failure !== undefined && <Alert tone="error" title={failure} />}
        <TextField
          label="Name"
          name="name"
          autoComplete="off"
          maxLength={MAX_NAME}
          placeholder="For example: Order notifications"
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setErrors((current) => ({ ...current, name: undefined }));
          }}
          error={errors.name}
          hint="Helps you recognize where the key is used. Do not include secrets."
          disabled={saving}
          data-autofocus
        />
        <fieldset
          aria-describedby={errors.scopes !== undefined ? `${scopesId}-error` : undefined}
          className="space-y-2"
        >
          <legend className="text-sm font-medium text-text">Permissions</legend>
          {SCOPES.map((scope) => (
            <label
              key={scope}
              className={`flex cursor-pointer gap-3 rounded-control border p-3 transition-colors hover:bg-surface-subtle ${
                scopes.includes(scope) ? 'border-primary bg-primary-subtle' : 'border-border'
              }`}
            >
              <input
                type="checkbox"
                name="scopes"
                value={scope}
                checked={scopes.includes(scope)}
                disabled={saving}
                onChange={(e) => {
                  setScopes((current) =>
                    e.target.checked ? [...current, scope] : current.filter((s) => s !== scope),
                  );
                  setErrors((current) => ({ ...current, scopes: undefined }));
                }}
                className="mt-0.5 size-4 shrink-0 accent-primary"
              />
              <span className="min-w-0">
                <span className="block text-sm font-medium text-text">
                  {SCOPE_LABELS[scope].title}{' '}
                  <code className="text-xs font-normal text-text-muted">{scope}</code>
                </span>
                <span className="block text-xs text-text-muted">
                  {SCOPE_LABELS[scope].description}
                </span>
              </span>
            </label>
          ))}
          {errors.scopes !== undefined && (
            <p id={`${scopesId}-error`} className="text-xs font-medium text-danger">
              {errors.scopes}
            </p>
          )}
        </fieldset>
      </form>
    </Dialog>
  );
}
