import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { App } from './App';

// Make the page fail while rendering, to prove App wraps it in the error boundary.
vi.mock('../features/home/HomePage', () => ({
  HomePage: (): never => {
    throw new Error('page render failure (test)');
  },
}));

describe('App with a failing page', () => {
  it('shows the recovery screen inside the shell, which stays usable', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);

    render(<App />);

    const main = screen.getByRole('main');
    expect(main).toContainElement(screen.getByRole('alert'));
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Something went wrong');
    expect(screen.getByRole('button', { name: 'Reload page' })).toBeInTheDocument();
    expect(screen.getByRole('banner')).toHaveTextContent('Client Portal');
    expect(screen.getByRole('link', { name: 'Skip to main content' })).toBeInTheDocument();
  });
});
