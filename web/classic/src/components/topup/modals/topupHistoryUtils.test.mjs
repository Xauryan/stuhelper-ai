import assert from 'node:assert/strict';
import {
  BILLING_PAYMENT_METHOD_FILTERS,
  canAdminCompleteTopup,
  getTopupPaymentMethodLabel,
  getRemainingAdminRefundQuota,
  getRemainingRefundMoney,
  isAlipayOfficialRefundable,
  isAdminManagedTopup,
  isAdminManagedTopupRefundable,
  isOfficialPaymentTopup,
  isOfficialRefundable,
  isSelfServeTopup,
  isSubscriptionTopup,
} from './topupHistoryUtils.mjs';

assert.deepEqual(BILLING_PAYMENT_METHOD_FILTERS, [
  { value: '', key: '全部' },
  { value: 'alipay_official', key: '支付宝' },
  { value: 'wxpay_official', key: '微信' },
  { value: 'self_serve', key: '自助充值' },
  { value: 'admin_add', key: '管理员充值' },
]);
assert.equal(
  getRemainingRefundMoney({ money: 1.01, refunded_money: 0.4 }),
  0.61,
);
assert.equal(
  getRemainingRefundMoney({ money: 20, fee: 0.12, refunded_money: 0 }),
  20,
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
assert.equal(
  isAdminManagedTopup({
    payment_provider: 'admin',
    payment_method: 'admin_add',
  }),
  true,
);
assert.equal(
  isAdminManagedTopup({
    payment_method: '管理员增加',
  }),
  true,
);
assert.equal(getTopupPaymentMethodLabel('管理员增加'), '管理员充值');
assert.equal(getTopupPaymentMethodLabel('alipay_self_serve'), '支付宝自助');
assert.equal(getTopupPaymentMethodLabel('wxpay_self_serve'), '微信自助');
assert.equal(
  isSelfServeTopup({
    payment_provider: 'self_serve',
    payment_method: 'alipay_self_serve',
  }),
  true,
);
assert.equal(
  isSelfServeTopup({
    payment_provider: 'alipay_official',
    payment_method: 'alipay_official',
  }),
  false,
);
assert.equal(
  getRemainingAdminRefundQuota({
    amount: 1000,
    refunded_quota: 250,
  }),
  750,
);
assert.equal(
  isAdminManagedTopupRefundable({
    status: 'partial_refunded',
    payment_method: 'admin_add',
    amount: 1000,
    refunded_quota: 250,
  }),
  true,
);
assert.equal(
  isAdminManagedTopupRefundable({
    status: 'refunded',
    payment_method: 'admin_add',
    amount: 1000,
    refunded_quota: 1000,
  }),
  false,
);

console.log('topupHistoryUtils tests passed');
