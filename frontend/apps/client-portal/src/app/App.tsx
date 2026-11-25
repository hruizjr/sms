import { AppShell } from '../components/layout/AppShell';
import { HomePage } from '../features/home/HomePage';
import { AppErrorBoundary } from './AppErrorBoundary';

/**
 * The client portal has one view, at "/" (owner decision F4: no router until Phase 4).
 * Navigation is added when more than one real page exists.
 */
export function App() {
  return (
    <AppShell>
      <AppErrorBoundary>
        <HomePage />
      </AppErrorBoundary>
    </AppShell>
  );
}
