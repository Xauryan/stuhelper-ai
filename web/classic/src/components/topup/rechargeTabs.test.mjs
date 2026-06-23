import assert from 'node:assert/strict';
import {
  getRechargeTabKeys,
  getInitialRechargeTabKey,
  shouldShowSubscriptionTab,
} from './rechargeTabs.js';

assert.equal(getInitialRechargeTabKey(), 'topup');
assert.deepEqual(getRechargeTabKeys(true), ['topup', 'subscription']);
assert.deepEqual(getRechargeTabKeys(false), ['topup']);
assert.equal(
  shouldShowSubscriptionTab({
    loading: false,
    plans: [{ plan: { id: 1 } }],
  }),
  true,
);
assert.equal(
  shouldShowSubscriptionTab({
    loading: true,
    plans: [],
    allSubscriptions: [],
  }),
  false,
);
assert.equal(
  shouldShowSubscriptionTab({
    loading: true,
    plans: [],
    allSubscriptions: [{ subscription: { id: 1 } }],
  }),
  true,
);
assert.equal(
  shouldShowSubscriptionTab({
    loading: false,
    plans: [],
    activeSubscriptions: [{ subscription: { id: 1 } }],
  }),
  true,
);

console.log('rechargeTabs tests passed');
