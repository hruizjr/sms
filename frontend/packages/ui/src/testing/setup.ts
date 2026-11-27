import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';
import { installDialogPolyfill } from './dialog-polyfill';
import { installNetworkGuard } from './network-guard';

installNetworkGuard();
installDialogPolyfill();

afterEach(() => {
  cleanup();
});
