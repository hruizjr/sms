import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AppErrorBoundary } from './AppErrorBoundary';

function FailsWhileRendering(): never {
  throw new Error('render failure (test)');
}

describe('AppErrorBoundary', () => {
  it('renders its children when nothing fails', () => {
    render(
      <AppErrorBoundary>
        <p>Healthy content</p>
      </AppErrorBoundary>,
    );
    expect(screen.getByText('Healthy content')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('shows a recovery screen when a component fails while rendering', () => {
    // React reports the caught error to the console; keep the test output clean.
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);

    render(
      <AppErrorBoundary>
        <FailsWhileRendering />
      </AppErrorBoundary>,
    );

    const alert = screen.getByRole('alert');
    expect(alert).toHaveTextContent('Something went wrong');
    // No error details reach the user.
    expect(alert).not.toHaveTextContent('render failure (test)');

    const reload = screen.getByRole('button', { name: 'Reload page' });
    reload.focus();
    expect(reload).toHaveFocus();
    expect(consoleError).toHaveBeenCalled();
  });
});
