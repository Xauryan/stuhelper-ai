import assert from 'node:assert/strict';
import {
  getRefundLogDetailText,
  isModelBillingLog,
  shouldShowLogIp,
} from './usageLogDisplayRules.mjs';

assert.equal(isModelBillingLog({ type: 2 }), true);
assert.equal(isModelBillingLog({ type: 6 }), false);

assert.equal(shouldShowLogIp({ type: 6, ip: '203.0.113.8' }, true), true);
assert.equal(shouldShowLogIp({ type: 6, ip: '203.0.113.8' }, false), false);
assert.equal(shouldShowLogIp({ type: 6, ip: '' }, true), false);

assert.equal(
  getRefundLogDetailText(
    { type: 6, content: '管理员发起支付宝官方退款成功' },
    '退款',
  ),
  '管理员发起支付宝官方退款成功',
);
assert.equal(
  getRefundLogDetailText({ type: 2, content: 'consume' }, '退款'),
  null,
);

console.log('usageLogDisplayRules tests passed');
