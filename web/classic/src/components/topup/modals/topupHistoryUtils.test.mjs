import assert from 'node:assert/strict';
import {
  getRemainingRefundMoney,
  isAlipayOfficialRefundable,
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

console.log('topupHistoryUtils tests passed');
