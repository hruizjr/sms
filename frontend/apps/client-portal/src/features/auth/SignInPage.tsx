import { Button, Card } from '@sms/ui';
import { useEffect, useState, type SyntheticEvent } from 'react';
import { Alert } from '../../components/feedback/Alert';
import { TextField } from '../../components/form/Field';
import { ApiError, NetworkError } from '../../lib/api/client';
import { useSession } from './session';

type Errors = { email?: string; password?: string };

function validate(email: string, password: string): Errors {
  const errors: Errors = {};
  const trimmed = email.trim();
  if (trimmed === '') {
    errors.email = 'Enter your email address.';
  } else if (!/^[^\s@]+@[^\s@]+$/.test(trimmed)) {
    errors.email = 'Enter a valid email address, for example name@company.ph.';
  }
  if (password === '') {
    errors.password = 'Enter your password.';
  }
  return errors;
}

function failureMessage(error: unknown): string {
  if (error instanceof ApiError) {
    switch (error.problemType) {
      case 'invalid-credentials':
        return 'The email address or password is incorrect, or the account is temporarily locked after too many attempts. Try again later.';
      case 'account-not-active':
        return 'Your organization’s account is not active. Contact support.';
    }
    return 'Sign-in failed. Try again.';
  }
  if (error instanceof NetworkError) {
    return 'The server could not be reached. Check your connection and try again.';
  }
  return 'Sign-in failed. Try again.';
}

export function SignInPage({ notice }: { notice?: string }) {
  const { signIn } = useSession();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState<Errors>({});
  const [failure, setFailure] = useState<string | undefined>(undefined);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    document.title = 'Sign in · Client Portal';
  }, []);

  async function onSubmit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;
    const found = validate(email, password);
    setErrors(found);
    setFailure(undefined);
    if (Object.keys(found).length > 0) return;
    setSubmitting(true);
    try {
      await signIn(email.trim(), password);
    } catch (error: unknown) {
      setFailure(failureMessage(error));
      setPassword('');
      setSubmitting(false);
    }
  }

  return (
    <div className="mx-auto max-w-md space-y-6 py-6">
      <div className="space-y-2">
        <h1 className="text-2xl font-semibold">Sign in</h1>
        <p className="text-text-muted">Sign in to send SMS and see your organization’s messages.</p>
      </div>
      {notice !== undefined && <Alert tone="info" title={notice} />}
      {failure !== undefined && (
        <Alert tone="error" title="Sign-in failed">
          {failure}
        </Alert>
      )}
      <Card>
        <form noValidate onSubmit={(e) => void onSubmit(e)} className="space-y-4">
          <TextField
            label="Email address"
            type="email"
            name="email"
            autoComplete="username"
            value={email}
            onChange={(e) => {
              setEmail(e.target.value);
            }}
            error={errors.email}
            required
          />
          <TextField
            label="Password"
            type="password"
            name="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
            }}
            error={errors.password}
            required
          />
          <Button type="submit" loading={submitting} className="w-full">
            {submitting ? 'Signing in…' : 'Sign in'}
          </Button>
        </form>
      </Card>
      <p className="text-sm text-text-muted">
        Accounts are created by the platform team. There is no self-registration or password reset
        by email yet; contact your administrator if you cannot sign in.
      </p>
    </div>
  );
}
