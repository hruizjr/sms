import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { json, signedIn } from '../../test/fake-api';
import { renderApp } from '../../test/render';

const emptyList = () => json(200, { data: [], next_cursor: null });

function openMenu() {
  const button = screen.getByRole('button', { name: 'Open menu' });
  expect(button).toHaveAttribute('aria-expanded', 'false');
  fireEvent.click(button);
  expect(button).toHaveAttribute('aria-expanded', 'true');
  return screen.findByRole('dialog', { name: 'Menu' });
}

describe('small-screen menu', () => {
  it('opens the navigation in a modal panel and closes it with the close button', async () => {
    renderApp('/messages', signedIn('owner', { 'GET /portal/v1/messages?limit=25': emptyList }));
    await screen.findByRole('heading', { level: 1, name: 'Messages' });
    const menu = await openMenu();
    const nav = within(within(menu).getByRole('navigation', { name: 'Main' }));
    expect(nav.getByRole('link', { name: 'Messages' })).toHaveAttribute('aria-current', 'page');
    expect(nav.getByRole('link', { name: 'API keys' })).toBeInTheDocument();
    expect(menu).toHaveTextContent('Example Co');
    expect(menu).toHaveTextContent('Role: Owner');
    // Focus starts on the current page's link.
    await waitFor(() => {
      expect(nav.getByRole('link', { name: 'Messages' })).toHaveFocus();
    });
    fireEvent.click(within(menu).getByRole('button', { name: 'Close menu' }));
    await waitFor(() => {
      expect(screen.queryByRole('dialog', { name: 'Menu' })).not.toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: 'Open menu' })).toHaveAttribute(
      'aria-expanded',
      'false',
    );
  });

  it('closes with Escape', async () => {
    renderApp('/messages', signedIn('owner', { 'GET /portal/v1/messages?limit=25': emptyList }));
    await screen.findByRole('heading', { level: 1, name: 'Messages' });
    const menu = await openMenu();
    fireEvent(menu, new Event('cancel', { cancelable: true }));
    await waitFor(() => {
      expect(screen.queryByRole('dialog', { name: 'Menu' })).not.toBeInTheDocument();
    });
  });

  it('closes after navigating to another page', async () => {
    renderApp(
      '/messages',
      signedIn('owner', {
        'GET /portal/v1/messages?limit=25': emptyList,
        'GET /portal/v1/api-keys': () => json(200, { data: [] }),
      }),
    );
    await screen.findByRole('heading', { level: 1, name: 'Messages' });
    const menu = await openMenu();
    fireEvent.click(within(menu).getByRole('link', { name: 'API keys' }));
    expect(await screen.findByRole('heading', { level: 1, name: 'API keys' })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.queryByRole('dialog', { name: 'Menu' })).not.toBeInTheDocument();
    });
    expect(window.location.pathname).toBe('/api-keys');
  });

  it('shows only the pages the role may use', async () => {
    renderApp('/messages', signedIn('viewer', { 'GET /portal/v1/messages?limit=25': emptyList }));
    await screen.findByRole('heading', { level: 1, name: 'Messages' });
    const menu = await openMenu();
    const nav = within(within(menu).getByRole('navigation', { name: 'Main' }));
    expect(nav.getAllByRole('link').map((link) => link.textContent)).toEqual(['Messages']);
  });
});
