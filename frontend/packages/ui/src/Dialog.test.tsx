import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Dialog } from './Dialog';

describe('Dialog', () => {
  it('is a modal dialog named and described by its title and description', () => {
    render(
      <Dialog
        open
        onClose={() => undefined}
        title="Revoke key?"
        description="This cannot be undone."
      >
        Body
      </Dialog>,
    );
    const dialog = screen.getByRole('dialog', { name: 'Revoke key?' });
    expect(dialog).toHaveAttribute('open');
    expect(dialog).toHaveAccessibleDescription('This cannot be undone.');
    expect(dialog).toHaveTextContent('Body');
  });

  it('moves focus to the element marked data-autofocus', () => {
    render(
      <Dialog open onClose={() => undefined} title="Name it">
        <button>First</button>
        <input aria-label="Name" data-autofocus />
      </Dialog>,
    );
    expect(screen.getByLabelText('Name')).toHaveFocus();
  });

  it('renders nothing inside while closed', () => {
    render(
      <Dialog open={false} onClose={() => undefined} title="Hidden">
        Body
      </Dialog>,
    );
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(screen.queryByText('Body')).not.toBeInTheDocument();
  });

  it('asks its owner to close on Escape, unless it is not dismissible', () => {
    const onClose = vi.fn();
    const { rerender } = render(<Dialog open onClose={onClose} title="Confirm" />);
    const dialog = screen.getByRole('dialog');
    fireEvent(dialog, new Event('cancel', { cancelable: true }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(dialog).toHaveAttribute('open');

    rerender(<Dialog open onClose={onClose} title="Confirm" dismissible={false} />);
    fireEvent(screen.getByRole('dialog'), new Event('cancel', { cancelable: true }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('closes when open becomes false', () => {
    const { rerender } = render(<Dialog open onClose={() => undefined} title="Confirm" />);
    expect(screen.getByRole('dialog')).toHaveAttribute('open');
    rerender(<Dialog open={false} onClose={() => undefined} title="Confirm" />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
