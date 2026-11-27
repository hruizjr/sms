import { render } from '@testing-library/react';
import { App } from '../app/App';
import { fakeTransport, type Handler } from './fake-api';

/** Renders the whole portal at path with a fake API; returns the recorded requests. */
export function renderApp(path: string, handler: Handler, newIdempotencyKey?: () => string) {
  window.history.replaceState(null, '', path);
  const transport = fakeTransport(handler);
  const result = render(
    <App transport={transport} {...(newIdempotencyKey ? { newIdempotencyKey } : {})} />,
  );
  return { ...result, transport };
}
