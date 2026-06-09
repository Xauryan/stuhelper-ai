import assert from 'node:assert/strict';
import {
  buildSubscriptionPaymentMethods,
  getEpayMethods,
  getOfficialAlipayMethod,
  getOfficialWechatPayMethod,
  getSelfServeMethods,
} from './subscriptionPaymentMethods.js';

const methods = [
  { name: 'Alipay', type: 'alipay' },
  { name: 'WeChat', type: 'wxpay' },
  { name: 'Stripe', type: 'stripe' },
  { name: 'Alipay Official', type: 'alipay_official' },
  { name: 'WeChat Official', type: 'wxpay_official' },
  { name: 'Alipay Self Serve', type: 'alipay_self_serve' },
  { name: 'WeChat Self Serve', type: 'wxpay_self_serve' },
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
    { name: '支付宝自助', type: 'alipay_self_serve' },
    { name: '微信自助', type: 'wxpay_self_serve' },
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
  enableSelfServeTopUp: true,
  hasAlipayOfficial: true,
  hasWechatPayOfficial: true,
  epayUnitPrice: 1.006,
  selfServeQrCodes: {
    alipay_self_serve: 'alipay-qr-content',
    wxpay_self_serve: 'wxpay-qr-content',
  },
  selfServeUnitPrice: 1.15,
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
    'self_serve:alipay_self_serve',
    'self_serve:wxpay_self_serve',
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
  subscriptionMethods.find((method) => method.key === 'epay:alipay')?.unitPrice,
  1.2,
);
assert.equal(
  subscriptionMethods.find((method) => method.key === 'epay:wxpay')?.unitPrice,
  1.006,
);
assert.equal(
  subscriptionMethods.find(
    (method) => method.key === 'self_serve:alipay_self_serve',
  )?.unitPrice,
  1.15,
);
assert.equal(
  subscriptionMethods.find(
    (method) => method.key === 'self_serve:wxpay_self_serve',
  )?.qrCode,
  'wxpay-qr-content',
);
assert.deepEqual(
  getSelfServeMethods({
    selfServeQrCodes: { alipay_self_serve: 'alipay-qr-content' },
    selfServeUnitPrice: 1.2,
  }).map((method) => method.key),
  ['self_serve:alipay_self_serve'],
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
