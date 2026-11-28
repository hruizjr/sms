import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { COPIED_RESET_MS, CopyButton } from './CopyButton';

function setClipboard(writeText: ((text: string) => Promise<void>) | undefined) {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: writeText === undefined ? undefined : { writeText },
  });
}

afterEach(() => {
  setClipboard(undefined);
  vi.useRealTimers();
});

describe('CopyButton', () => {
  it('copies the value and announces it, then resets', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const writeText = vi.fn(() => Promise.resolve());
    setClipboard(writeText);
    render(<CopyButton value="msg_1" label="Copy message ID" />);
    fireEvent.click(screen.getByRole('button', { name: 'Copy message ID' }));
    expect(await screen.findByRole('status')).toHaveTextContent('Copied');
    expect(writeText).toHaveBeenCalledWith('msg_1');
    await act(async () => {
      await vi.advanceTimersByTimeAsync(COPIED_RESET_MS);
    });
    expect(screen.getByRole('status')).toHaveTextContent('');
  });

  it('says so when the browser cannot copy', async () => {
    setClipboard(undefined);
    render(<CopyButton value="msg_1" label="Copy message ID" />);
    fireEvent.click(screen.getByRole('button', { name: 'Copy message ID' }));
    expect(await screen.findByText('Copy not available')).toBeInTheDocument();
  });
});
