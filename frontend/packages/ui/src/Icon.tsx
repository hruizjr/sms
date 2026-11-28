import { classNames } from './class-names';

/*
 * A small set of line icons drawn on a 24×24 grid with a 2px stroke in the current text colour. They
 * are decorative (aria-hidden): every icon is accompanied by visible text or an accessible name, so
 * meaning never depends on the icon alone (AGENTS.md §8.9). Hand-drawn geometric shapes; no icon
 * library is used.
 */

const PATHS = {
  check: 'M5 12.5l4.5 4.5L19 7.5',
  x: 'M6 6l12 12M18 6L6 18',
  clock: 'M12 3a9 9 0 1 0 0 18a9 9 0 1 0 0-18M12 7v5l3 2',
  alert: 'M12 3L2 20h20L12 3M12 10v4M12 17v.01',
  question:
    'M12 3a9 9 0 1 0 0 18a9 9 0 1 0 0-18M9.5 9.5a2.5 2.5 0 1 1 3.5 2.3c-.6.3-1 .9-1 1.6v.6M12 17v.01',
  info: 'M12 3a9 9 0 1 0 0 18a9 9 0 1 0 0-18M12 11v6M12 7.5v.01',
  menu: 'M4 6h16M4 12h16M4 18h16',
  plus: 'M12 5v14M5 12h14',
  key: 'M8 11a4 4 0 1 0 0 8a4 4 0 1 0 0-8M11 13l9-9M16 8l3 3M14 10l2 2',
  messages: 'M4 5h16v11H9l-5 4V5',
  send: 'M4 12l16-8-6 16-3-7-7-1M11 13l9-9',
  copy: 'M9 9h11v11H9V9M5 15H4V4h11v1',
  arrowLeft: 'M19 12H5M11 6l-6 6 6 6',
  chevronRight: 'M9 6l6 6-6 6',
  logOut: 'M10 4H5v16h5M15 8l4 4-4 4M19 12H9',
  refresh: 'M20 11a8 8 0 0 0-14.5-4.5L4 8M4 4v4h4M4 13a8 8 0 0 0 14.5 4.5L20 16M20 20v-4h-4',
  code: 'M8 7l-5 5 5 5M16 7l5 5-5 5',
  shield: 'M12 3l8 3v6c0 4.5-3.5 8-8 9c-4.5-1-8-4.5-8-9V6l8-3',
  lock: 'M6 11h12v10H6V11M8 11V8a4 4 0 0 1 8 0v3M12 15v2',
  building: 'M4 21h16M6 21V4h9v17M15 9h3v12M9 8h3M9 12h3M9 16h3',
} as const;

export type IconName = keyof typeof PATHS;

export type IconProps = {
  name: IconName;
  className?: string;
};

export function Icon({ name, className }: IconProps) {
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      data-icon={name}
      className={classNames('size-4 shrink-0', className)}
    >
      <path d={PATHS[name]} />
    </svg>
  );
}
