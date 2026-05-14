import assert from 'node:assert/strict';
import {
  getRemainingRefundMoney,
  isAlipayOfficialRefundable,
  isSubscriptionTopup,
} from './topupHistoryUtils.mjs';

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
