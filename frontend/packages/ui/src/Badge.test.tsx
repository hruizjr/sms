import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Badge } from './Badge';

describe('Badge', () => {
  it('shows its text, with a decorative icon hidden from assistive technology', () => {
    const { container } = render(
      <Badge tone="success" icon="check">
        Delivered
      </Badge>,
    );
    expect(screen.getByText('Delivered')).toBeInTheDocument();
    const icon = container.querySelector('svg');
    expect(icon).toHaveAttribute('aria-hidden', 'true');
    expect(icon).toHaveAttribute('data-icon', 'check');
  });

  it('uses the tone tokens', () => {
    render(<Badge tone="danger">Revoked</Badge>);
    expect(screen.getByText('Revoked')).toHaveClass('bg-danger-subtle', 'text-danger');
  });
});
