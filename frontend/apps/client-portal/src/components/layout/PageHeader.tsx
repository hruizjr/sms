import { Icon } from '@sms/ui';
import type { ReactNode } from 'react';
import { Link } from '../../lib/router';

export type Breadcrumb = { label: string; to: string };

export type PageHeaderProps = {
  title: ReactNode;
  /** One sentence on what the page is for. */
  description?: ReactNode;
  /** The page's main actions, right-aligned on wide screens. */
  actions?: ReactNode;
  /** The pages above this one, outermost first; the current page (the title) is added last. */
  breadcrumbs?: Breadcrumb[];
  /** The current page's name in the breadcrumb trail when the title is not plain text. */
  breadcrumbLabel?: string;
  /** Content next to the title, such as a status badge. */
  meta?: ReactNode;
};

/**
 * The same header on every page: an optional breadcrumb trail, the title (the page's h1) with
 * optional meta, a description, and the page's actions.
 */
export function PageHeader({
  title,
  description,
  actions,
  breadcrumbs,
  breadcrumbLabel,
  meta,
}: PageHeaderProps) {
  const current = breadcrumbLabel ?? (typeof title === 'string' ? title : undefined);
  return (
    <div className="mb-6 space-y-3">
      {breadcrumbs !== undefined && breadcrumbs.length > 0 && (
        <nav aria-label="Breadcrumb">
          <ol className="flex flex-wrap items-center gap-1 text-sm text-text-muted">
            {breadcrumbs.map((crumb) => (
              <li key={crumb.to} className="flex items-center gap-1">
                <Link
                  to={crumb.to}
                  className="rounded-control font-medium hover:text-text hover:underline"
                >
                  {crumb.label}
                </Link>
                <Icon name="chevronRight" className="size-3.5" />
              </li>
            ))}
            {current !== undefined && (
              <li aria-current="page" className="max-w-full truncate text-text">
                {current}
              </li>
            )}
          </ol>
        </nav>
      )}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
            <h1 className="text-2xl leading-8 font-semibold tracking-tight text-text">{title}</h1>
            {meta}
          </div>
          {description !== undefined && (
            <div className="max-w-3xl text-sm text-text-muted">{description}</div>
          )}
        </div>
        {actions !== undefined && (
          <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>
        )}
      </div>
    </div>
  );
}
