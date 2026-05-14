import assert from 'node:assert/strict';
import {
  buildSubscriptionPaymentMethods,
  getEpayMethods,
  getOfficialAlipayMethod,
  getOfficialWechatPayMethod,
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
assert.equal(getOfficialWechatPayMethod(methods)?.type, 'wxpay_official');
assert.equal(getOfficialWechatPayMethod([{ type: 'alipay_official' }]), null);

const subscriptionMethods = buildSubscriptionPaymentMethods({
  plan: {
    stripe_price_id: 'price_123',
    creem_product_id: 'prod_123',
  },
  payMethods: [
    { name: 'Stripe', type: 'stripe' },
    { name: '支付宝', type: 'alipay_official', unit_price: '1.006' },
    { name: '微信', type: 'wxpay_official', unit_price: '1.008' },
  ],
  epayMethods: [
    { name: '易支付支付宝', type: 'alipay', unit_price: '1.2' },
    { name: '易支付微信', type: 'wxpay' },
  ],
  enableOnlineTopUp: true,
  enableStripeTopUp: true,
  enableCreemTopUp: true,
  enableAlipayOfficialTopUp: true,
  enableWechatPayOfficialTopUp: true,
  hasAlipayOfficial: true,
  hasWechatPayOfficial: true,
  epayUnitPrice: 1.006,
});

assert.deepEqual(
  subscriptionMethods.map((method) => method.key),
  [
    'stripe',
    'creem',
    'alipay_official',
    'wxpay_official',
    'epay:alipay',
    'epay:wxpay',
  ],
);
assert.equal(
  subscriptionMethods.find((method) => method.key === 'alipay_official')
    ?.unitPrice,
  1.006,
);
assert.equal(
  subscriptionMethods.find((method) => method.key === 'wxpay_official')
    ?.unitPrice,
  1.008,
);
assert.equal(
  subscriptionMethods.find((method) => method.key === 'epay:alipay')
    ?.unitPrice,
  1.2,
);
assert.equal(
  subscriptionMethods.find((method) => method.key === 'epay:wxpay')?.unitPrice,
  1.006,
);
assert.equal(
  buildSubscriptionPaymentMethods({
    plan: {},
    payMethods: [{ name: '微信', type: 'wxpay_official', unit_price: '1.006' }],
    enableWechatPayOfficialTopUp: true,
    hasWechatPayOfficial: false,
  }).length,
  0,
);

console.log('subscriptionPaymentMethods tests passed');
