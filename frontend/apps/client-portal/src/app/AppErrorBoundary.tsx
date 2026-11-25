import { Button } from '@sms/ui';
import { Component, type ReactNode } from 'react';

type Props = { children: ReactNode };
type State = { hasError: boolean };

/**
 * Shows a recovery screen instead of a blank page when a component below it fails while
 * rendering.
 *
 * Scope (React error boundaries): it catches errors thrown while rendering, in lifecycle methods,
 * and in constructors of the components below it. It does NOT catch errors in event handlers,
 * asynchronous code (promises, timers), errors thrown in this boundary itself, or errors outside
 * the tree it wraps (such as the application shell). Those need handling where they occur.
 *
 * Error details are never shown to the user. React reports the caught error to the browser
 * console; there is no error reporting service yet.
 */
export class AppErrorBoundary extends Component<Props, State> {
  override state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  override render(): ReactNode {
    if (!this.state.hasError) {
      return this.props.children;
    }
    return (
      <div role="alert" className="max-w-prose space-y-4">
        <h1 className="text-2xl font-semibold text-text">Something went wrong</h1>
        <p className="text-text-muted">
          This page could not be displayed. Reload the page to try again.
        </p>
        <Button
          variant="primary"
          onClick={() => {
            window.location.reload();
          }}
        >
          Reload page
        </Button>
      </div>
    );
  }
}
