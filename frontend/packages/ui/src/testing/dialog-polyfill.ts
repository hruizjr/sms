/*
 * jsdom does not implement HTMLDialogElement.showModal() and close(). This test-only stand-in sets
 * and removes the open attribute and fires "close", which is what the Dialog component relies on.
 * Browsers provide the real behaviour (focus containment, inert page, Escape).
 */
export function installDialogPolyfill(): void {
  const proto = HTMLDialogElement.prototype as HTMLDialogElement & { __polyfilled?: boolean };
  if (typeof proto.showModal === 'function' && !proto.__polyfilled) return;
  proto.showModal = function showModal(this: HTMLDialogElement) {
    this.setAttribute('open', '');
  };
  proto.close = function close(this: HTMLDialogElement) {
    if (!this.hasAttribute('open')) return;
    this.removeAttribute('open');
    this.dispatchEvent(new Event('close'));
  };
  proto.__polyfilled = true;
}
