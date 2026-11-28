import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DescriptionItem, DescriptionList } from './DescriptionList';

describe('DescriptionList', () => {
  it('pairs each term with its value', () => {
    render(
      <DescriptionList>
        <DescriptionItem term="Recipient">+639000000001</DescriptionItem>
        <DescriptionItem term="Segments">1</DescriptionItem>
      </DescriptionList>,
    );
    const terms = screen.getAllByRole('term').map((t) => t.textContent);
    const values = screen.getAllByRole('definition').map((d) => d.textContent);
    expect(terms).toEqual(['Recipient', 'Segments']);
    expect(values).toEqual(['+639000000001', '1']);
  });

  it('supports the stacked layout', () => {
    const { container } = render(
      <DescriptionList layout="stacked">
        <DescriptionItem term="Sender ID" layout="stacked">
          EXAMPLECO
        </DescriptionItem>
      </DescriptionList>,
    );
    expect(container.querySelector('dl')).toHaveAttribute('data-layout', 'stacked');
    expect(screen.getByRole('definition')).toHaveTextContent('EXAMPLECO');
  });
});
