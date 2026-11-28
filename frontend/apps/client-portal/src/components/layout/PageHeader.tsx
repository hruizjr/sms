import { Icon } from '@sms/ui';
import type { ReactNode } from 'react';
import { Link } from '../../lib/router';

export type PageHeaderProps = {
  title: ReactNode;
  /** One sentence on what the page is for. */
  description?: ReactNode;
  /** The page's main actions, right-aligned on wide screens. */
  actions?: ReactNode;
  /** A link back to the parent page. */
  back?: { to: string; label: string };
  /** Content next to the title, such as a status badge. */
  meta?: ReactNode;
};

/** The same header on every page: optional back link, title (the page's h1), description, actions. */
export function PageHeader({ title, description, actions, back, meta }: PageHeaderProps) {
  return (
    <div className="mb-6 space-y-3">
      {back !== undefined && (
        <Link
          to={back.to}
          className="inline-flex items-center gap-1.5 rounded-control text-sm font-medium text-text-muted hover:text-text"
        >
          <Icon name="arrowLeft" />
          {back.label}
        </Link>
      )}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <h1 className="text-xl font-semibold tracking-tight text-text sm:text-2xl">{title}</h1>
            {meta}
          </div>
          {description !== undefined && (
            <p className="max-w-2xl text-sm text-text-muted">{description}</p>
          )}
        </div>
        {actions !== undefined && <div className="flex shrink-0 flex-wrap gap-2">{actions}</div>}
      </div>
    </div>
  );
}
