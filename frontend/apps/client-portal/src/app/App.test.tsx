import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import indexHtml from '../../index.html?raw';
import { App } from './App';

describe('App shell', () => {
  it('has a banner and a main landmark, and no navigation yet', () => {
    render(<App />);
    expect(screen.getByRole('banner')).toHaveTextContent('Client Portal');
    expect(screen.getByRole('main')).toBeInTheDocument();
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
  });

  it('has exactly one level-1 heading, inside the main content', () => {
    render(<App />);
    const headings = screen.getAllByRole('heading', { level: 1 });
    expect(headings).toHaveLength(1);
    expect(within(screen.getByRole('main')).getByRole('heading', { level: 1 })).toHaveTextContent(
      'Client Portal',
    );
  });

  it('starts with a skip link that targets the main content', () => {
    const { container } = render(<App />);
    const skipLink = screen.getByRole('link', { name: 'Skip to main content' });
    const firstFocusable = container.querySelector('a[href], button, input, select, textarea');
    expect(firstFocusable).toBe(skipLink);

    const main = screen.getByRole('main');
    expect(skipLink).toHaveAttribute('href', `#${main.id}`);
    // tabIndex -1 lets the skip link move focus into <main> without adding it to the tab order.
    expect(main).toHaveAttribute('tabindex', '-1');
    main.focus();
    expect(main).toHaveFocus();
  });

  it('shows the placeholder content as named regions', () => {
    render(<App />);
    expect(screen.getByRole('region', { name: 'Portal foundation' })).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Features' })).toBeInTheDocument();
  });

  it('has no controls without a meaningful destination', () => {
    render(<App />);
    // Only the skip link: no navigation menu, menu button, login or other placeholder controls.
    expect(screen.queryAllByRole('button')).toEqual([]);
    expect(screen.getAllByRole('link')).toHaveLength(1);
  });
});

describe('index.html', () => {
  it('declares the language, a responsive viewport and a title', () => {
    expect(indexHtml).toContain('<html lang="en">');
    expect(indexHtml).toContain(
      '<meta name="viewport" content="width=device-width, initial-scale=1.0" />',
    );
    expect(indexHtml).toContain('<title>Client Portal</title>');
  });
});
