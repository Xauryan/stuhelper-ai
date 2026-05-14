import assert from 'node:assert/strict';
import {
  getRechargeTabKeys,
  getInitialRechargeTabKey,
} from './rechargeTabs.js';

assert.equal(getInitialRechargeTabKey(), 'topup');
assert.deepEqual(getRechargeTabKeys(true), ['topup', 'subscription']);
assert.deepEqual(getRechargeTabKeys(false), ['topup']);

console.log('rechargeTabs tests passed');
