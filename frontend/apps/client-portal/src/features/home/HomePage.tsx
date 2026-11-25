import { Card } from '@sms/ui';

/** The portal's only page while the frontend foundation is built (Phase 3). */
export function HomePage() {
  return (
    <div className="space-y-6">
      <div className="max-w-prose space-y-2">
        <h1 className="text-2xl font-semibold sm:text-3xl">Client Portal</h1>
        <p className="text-text-muted">
          This is the foundation of the client portal. Features will appear here once they are
          approved and built.
        </p>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Card heading="Portal foundation">
          <p>
            The application, its shared design system, and its build and test tooling are in place.
          </p>
        </Card>
        <Card heading="Features">
          <p>No portal features are available yet. They are added in later phases.</p>
        </Card>
      </div>
    </div>
  );
}
