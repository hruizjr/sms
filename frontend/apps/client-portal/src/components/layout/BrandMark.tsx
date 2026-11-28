import { Icon } from '@sms/ui';

/** The product name with a simple mark (no logo or brand has been decided). */
export function BrandMark() {
  return (
    <p className="flex items-center gap-2.5 text-sm font-semibold text-text">
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
