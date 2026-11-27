import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Card } from './Card';

describe('Card', () => {
  it('is a region named by its heading', () => {
    render(<Card heading="Account">Details</Card>);
    const region = screen.getByRole('region', { name: 'Account' });
    expect(region).toHaveTextContent('Details');
    expect(screen.getByRole('heading', { name: 'Account', level: 2 })).toBeInTheDocument();
  });

  it('uses the requested heading level', () => {
    render(
      <Card heading="Usage" headingLevel={3}>
        Details
      </Card>,
    );
    expect(screen.getByRole('heading', { name: 'Usage', level: 3 })).toBeInTheDocument();
  });

  it('is a plain container without a heading', () => {
    render(<Card>Plain content</Card>);
    expect(screen.queryByRole('region')).not.toBeInTheDocument();
    expect(screen.getByText('Plain content')).toBeInTheDocument();
  });
});

describe('Card header', () => {
  it('shows a description and actions next to the heading', () => {
    render(
      <Card
        heading="API keys"
        description="Keys of your organization"
        actions={<button>New</button>}
      >
        Rows
      </Card>,
    );
    const region = screen.getByRole('region', { name: 'API keys' });
    expect(region).toHaveTextContent('Keys of your organization');
    expect(screen.getByRole('button', { name: 'New' })).toBeInTheDocument();
  });
});
