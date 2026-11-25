import '@testing-library/jest-dom/vitest';
import { installNetworkGuard } from '@sms/ui/testing';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// Any attempt by rendered code to call the network fails the test (see @sms/ui/testing).
installNetworkGuard();

afterEach(() => {
  cleanup();
});
