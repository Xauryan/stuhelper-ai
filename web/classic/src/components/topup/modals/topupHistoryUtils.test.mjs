import assert from 'node:assert/strict';
import {
  BILLING_PAYMENT_METHOD_FILTERS,
  canAdminCompleteTopup,
  getRemainingRefundMoney,
  isAlipayOfficialRefundable,
  isOfficialPaymentTopup,
  isOfficialRefundable,
  isSubscriptionTopup,
} from './topupHistoryUtils.mjs';

assert.deepEqual(BILLING_PAYMENT_METHOD_FILTERS, [
  { value: '', key: '全部' },
  { value: 'alipay_official', key: '支付宝' },
  { value: 'wxpay_official', key: '微信' },
]);
assert.equal(
  getRemainingRefundMoney({ money: 1.01, refunded_money: 0.4 }),
  0.61,
);
assert.equal(getRemainingRefundMoney({ money: 1, refunded_money: 1 }), 0);
assert.equal(
  isAlipayOfficialRefundable({
    payment_provider: 'alipay_official',
    payment_method: 'alipay_official',
    status: 'success',
    money: 1,
    refunded_money: 0,
  }),
  true,
);
assert.equal(
  isAlipayOfficialRefundable({
    payment_provider: 'alipay_official',
    payment_method: 'alipay_official',
    status: 'partial_refunded',
    money: 1,
    refunded_money: 0.5,
  }),
  true,
);
assert.equal(
  isAlipayOfficialRefundable({
    payment_provider: 'alipay_official',
    payment_method: 'alipay_official',
    status: 'refunded',
    money: 1,
    refunded_money: 1,
  }),
  false,
);
assert.equal(
  isAlipayOfficialRefundable({
    payment_provider: 'wxpay_official',
    payment_method: 'wxpay_official',
    status: 'success',
    money: 1,
    refunded_money: 0,
  }),
  false,
);
assert.equal(
  isAlipayOfficialRefundable({
    trade_no: 'ALIPAYSUB_1_1778750000_ABCDEF',
    amount: 0,
    payment_provider: 'alipay_official',
    payment_method: 'alipay_official',
    status: 'success',
    money: 9.99,
    refunded_money: 0,
  }),
  false,
);
assert.equal(
  isOfficialRefundable({
    payment_provider: 'wxpay_official',
    payment_method: 'wxpay_official',
    status: 'success',
    money: 1,
    refunded_money: 0,
  }),
  true,
);
assert.equal(
  isOfficialRefundable({
    trade_no: 'WXSUB_1_ABCDEF1234567890',
    amount: 0,
    payment_provider: 'wxpay_official',
    payment_method: 'wxpay_official',
    status: 'success',
    money: 9.99,
    refunded_money: 0,
  }),
  true,
);
assert.equal(
  isOfficialRefundable({
    trade_no: 'WXSUB_1_ABCDEF1234567890',
    amount: 0,
    payment_provider: 'wxpay_official',
    payment_method: 'wxpay_official',
    status: 'partial_refunded',
    money: 9.99,
    refunded_money: 4,
  }),
  true,
);
assert.equal(
  isOfficialRefundable({
    payment_provider: 'stripe',
    payment_method: 'stripe',
    status: 'success',
    money: 1,
    refunded_money: 0,
  }),
  false,
);
assert.equal(
  isOfficialPaymentTopup({
    payment_provider: 'wxpay_official',
    payment_method: 'wxpay_official',
  }),
  true,
);
assert.equal(
  isOfficialPaymentTopup({
    payment_provider: 'stripe',
    payment_method: 'stripe',
  }),
  false,
);
assert.equal(
  canAdminCompleteTopup({
    trade_no: 'ALIPAY_1_1778750000_ABCDEF',
    amount: 10,
    payment_provider: 'alipay_official',
    payment_method: 'alipay_official',
    status: 'pending',
  }),
  true,
);
assert.equal(
  canAdminCompleteTopup({
    trade_no: 'WXSUB_1_ABCDEF1234567890',
    amount: 0,
    payment_provider: 'wxpay_official',
    payment_method: 'wxpay_official',
    status: 'expired',
  }),
  true,
);
assert.equal(
  canAdminCompleteTopup({
    trade_no: 'SUB_1_1778750000',
    amount: 0,
    payment_provider: 'stripe',
    payment_method: 'stripe',
    status: 'expired',
  }),
  false,
);
assert.equal(
  canAdminCompleteTopup({
    trade_no: 'SUB_1_1778750000',
    amount: 0,
    payment_provider: 'stripe',
    payment_method: 'stripe',
    status: 'pending',
  }),
  false,
);
assert.equal(
  isSubscriptionTopup({
    trade_no: 'SUB_1_1778750000',
    amount: 0,
  }),
  true,
);
assert.equal(
  isSubscriptionTopup({
    trade_no: 'ALIPAYSUB_1_1778750000_ABCDEF',
    amount: 0,
  }),
  true,
);
assert.equal(
  isSubscriptionTopup({
    trade_no: 'WXSUB_1_ABCDEF1234567890',
    amount: 0,
  }),
  true,
);
assert.equal(
  isSubscriptionTopup({
    trade_no: 'ALIPAY_1_1778750000_ABCDEF',
    amount: 0,
  }),
  false,
);
assert.equal(
  isSubscriptionTopup({
    trade_no: 'WXSUB_1_ABCDEF1234567890',
    amount: 10,
  }),
  false,
);

console.log('topupHistoryUtils tests passed');
