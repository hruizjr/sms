import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { signedIn } from '../test/fake-api';
import { renderApp } from '../test/render';

// Make the page fail while rendering, to prove App wraps it in the error boundary.
vi.mock('../features/messages/MessagesPage', () => ({
  MessagesPage: (): never => {
    throw new Error('page render failure (test)');
  },
}));

describe('App with a failing page', () => {
  it('shows the recovery screen inside the shell, which stays usable', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);

    renderApp('/messages', signedIn('owner'));

    const alert = await screen.findByRole('alert');
    expect(screen.getByRole('main')).toContainElement(alert);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Something went wrong');
    expect(screen.getByRole('button', { name: 'Reload page' })).toBeInTheDocument();
    expect(screen.getByRole('banner')).toHaveTextContent('Client Portal');
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument();
  });
});
