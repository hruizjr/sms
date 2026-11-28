import { Icon } from '@sms/ui';

/**
 * The product name with a simple mark (no logo or brand has been decided). "inverse" is for the dark
 * navigation surface.
 */
export function BrandMark({ tone = 'default' }: { tone?: 'default' | 'inverse' }) {
  return (
    <p
      className={`flex items-center gap-2.5 text-sm font-semibold tracking-tight ${
        tone === 'inverse' ? 'text-nav-active-foreground' : 'text-text'
      }`}
    >
      <span
        aria-hidden="true"
        className="flex size-7 items-center justify-center rounded-control bg-primary text-primary-foreground"
      >
        <Icon name="messages" className="size-4" />
      </span>
      Client Portal
    </p>
  );
}
