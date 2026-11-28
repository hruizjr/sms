import { Button, Dialog, Icon } from '@sms/ui';
import { useId, useRef, useState, type SyntheticEvent } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { TextField } from '../../components/form/Field';
import { ApiError, type ApiKeyCreated, type ApiKeyScope } from '../../lib/api/client';
import { useSession } from '../auth/session';
import { SCOPE_LABELS, SCOPES, apiKeyErrorMessage } from './scopes';

const MAX_NAME = 100;
const DEFAULT_SCOPES: ApiKeyScope[] = ['messages:send', 'messages:read'];

type Errors = { name?: string; scopes?: string; password?: string };

/**
 * The create form: a name, the permissions the key grants, and the owner's current password
 * (re-authentication, owner decision G4-7; the server checks it). The password is held in this
 * component's state only while the dialog is in use and is cleared after every attempt.
 */
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
  const [password, setPassword] = useState('');
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
      setPassword('');
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
    if (password === '') found.password = 'Enter your current password to confirm.';
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
        current_password: password,
      });
      setPassword('');
      onCreated(result);
    } catch (error: unknown) {
      // Never keep a password that was just sent: the owner types it again to retry.
      setPassword('');
      if (error instanceof ApiError && error.problemType === 'reauthentication-failed') {
        setErrors({
          password:
            'The password is incorrect, or your account is temporarily locked after too many attempts.',
        });
      } else if (error instanceof ApiError && error.problemType === 'validation-error') {
        const fromServer: Errors = {};
        for (const e of error.problem?.errors ?? []) {
          if (e.pointer === '/name') fromServer.name = e.detail;
          if (e.pointer === '/scopes') fromServer.scopes = e.detail;
          if (e.pointer === '/current_password') fromServer.password = e.detail;
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
      description="Confirm with your password. The full key is shown once, right after it is created."
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
        <div className="space-y-3 rounded-control border border-border bg-surface-subtle p-4">
          <p className="flex items-center gap-2 text-sm font-medium text-text">
            <Icon name="lock" className="text-text-muted" />
            Confirm it’s you
          </p>
          <TextField
            label="Your current password"
            type="password"
            name="current_password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
              setErrors((current) => ({ ...current, password: undefined }));
            }}
            error={errors.password}
            hint="Required to create an API key. Repeated wrong attempts lock your account for 15 minutes."
            disabled={saving}
            required
          />
        </div>
      </form>
    </Dialog>
  );
}
