import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { problem } from '../../test/fake-api';
import { renderApp } from '../../test/render';

async function form() {
  return {
    email: await screen.findByLabelText('Email address'),
    password: screen.getByLabelText('Password'),
    submit: screen.getByRole('button', { name: 'Sign in' }),
  };
}

describe('sign-in page', () => {
  it('validates the fields before sending anything', async () => {
    const { transport } = renderApp('/', () => problem(401, 'unauthenticated'));
    const { email, submit } = await form();
    fireEvent.click(submit);
    expect(await screen.findByText('Enter your email address.')).toBeInTheDocument();
    expect(screen.getByText('Enter your password.')).toBeInTheDocument();
    expect(email).toHaveAttribute('aria-invalid', 'true');
    expect(email).toHaveAccessibleDescription('Enter your email address.');

    fireEvent.change(email, { target: { value: 'not-an-email' } });
    fireEvent.click(submit);
    expect(await screen.findByText(/Enter a valid email address/)).toBeInTheDocument();
    expect(transport.calls.filter((c) => c.method === 'POST')).toHaveLength(0);
  });

  it('reports invalid credentials without saying which part was wrong, and clears the password', async () => {
    renderApp('/', (r) =>
      r.method === 'POST' ? problem(401, 'invalid-credentials') : problem(401, 'unauthenticated'),
    );
    const { email, password, submit } = await form();
    fireEvent.change(email, { target: { value: 'user@example.invalid' } });
    fireEvent.change(password, { target: { value: 'wrong password' } });
    fireEvent.click(submit);
    expect(await screen.findByRole('alert')).toHaveTextContent(
      /email address or password is incorrect, or the account is temporarily locked/,
    );
    expect(password).toHaveValue('');
  });

  it('explains a sign-in rate limit with the wait from Retry-After, without naming the account', async () => {
    renderApp('/', (r) =>
      r.method === 'POST'
        ? problem(429, 'too-many-sign-in-attempts', {}, { 'Retry-After': '360' })
        : problem(401, 'unauthenticated'),
    );
    const { email, password, submit } = await form();
    fireEvent.change(email, { target: { value: 'user@example.invalid' } });
    fireEvent.change(password, { target: { value: 'a password' } });
    fireEvent.click(submit);
    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent(
      'There have been too many sign-in attempts from your network. Wait about 6 minutes and try again.',
    );
    expect(alert).not.toHaveTextContent('user@example.invalid');
    expect(password).toHaveValue('');
  });

  it('explains an inactive organization and a connection failure', async () => {
    let mode: 'inactive' | 'offline' = 'inactive';
    renderApp('/', (r) => {
      if (r.method !== 'POST') return problem(401, 'unauthenticated');
      if (mode === 'offline') throw new TypeError('Failed to fetch');
      return problem(403, 'account-not-active');
    });
    const { email, password, submit } = await form();
    fireEvent.change(email, { target: { value: 'user@example.invalid' } });
    fireEvent.change(password, { target: { value: 'a long password' } });
    fireEvent.click(submit);
    expect(await screen.findByRole('alert')).toHaveTextContent(/account is not active/);

    mode = 'offline';
    fireEvent.change(password, { target: { value: 'a long password' } });
    fireEvent.click(submit);
    expect(await screen.findByText(/server could not be reached/)).toBeInTheDocument();
  });

  it('disables the button while signing in', async () => {
    renderApp('/', (r) =>
      r.method === 'POST'
        ? new Promise<Response>(() => undefined)
        : problem(401, 'unauthenticated'),
    );
    const { email, password, submit } = await form();
    fireEvent.change(email, { target: { value: 'user@example.invalid' } });
    fireEvent.change(password, { target: { value: 'a long password' } });
    fireEvent.click(submit);
    expect(await screen.findByRole('button', { name: 'Signing in…' })).toBeDisabled();
  });
});
