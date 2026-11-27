import '@testing-library/jest-dom/vitest';
import { installNetworkGuard } from '@sms/ui/testing';
import { cleanup, configure } from '@testing-library/react';
import { afterEach } from 'vitest';

// Any attempt by rendered code to call the network fails the test (see @sms/ui/testing).
installNetworkGuard();

// Pages load through an asynchronous fake API; allow for slow CI runners when waiting for them.
configure({ asyncUtilTimeout: 3000 });

afterEach(() => {
  cleanup();
});
