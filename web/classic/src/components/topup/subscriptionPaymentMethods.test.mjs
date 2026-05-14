import assert from 'node:assert/strict';
import {
  getEpayMethods,
  getOfficialAlipayMethod,
} from './subscriptionPaymentMethods.js';

const methods = [
  { name: 'Alipay', type: 'alipay' },
  { name: 'WeChat', type: 'wxpay' },
  { name: 'Stripe', type: 'stripe' },
  { name: 'Alipay Official', type: 'alipay_official' },
  { name: 'WeChat Official', type: 'wxpay_official' },
];

assert.deepEqual(
  getEpayMethods(methods).map((method) => method.type),
  ['alipay', 'wxpay'],
);
assert.equal(getOfficialAlipayMethod(methods)?.type, 'alipay_official');
assert.equal(getOfficialAlipayMethod([{ type: 'wxpay_official' }]), null);

console.log('subscriptionPaymentMethods tests passed');
