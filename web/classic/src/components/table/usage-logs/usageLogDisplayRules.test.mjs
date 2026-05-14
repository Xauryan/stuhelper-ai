import assert from 'node:assert/strict';
import {
  getBusinessLogExpandedDetailText,
  isModelBillingLog,
  shouldShowLogIp,
} from './usageLogDisplayRules.mjs';

assert.equal(isModelBillingLog({ type: 2 }), true);
assert.equal(isModelBillingLog({ type: 6 }), false);

assert.equal(shouldShowLogIp({ type: 6, ip: '203.0.113.8' }, true), true);
assert.equal(shouldShowLogIp({ type: 6, ip: '203.0.113.8' }, false), false);
assert.equal(shouldShowLogIp({ type: 6, ip: '' }, true), false);

assert.equal(
  getBusinessLogExpandedDetailText({
    type: 6,
    content: '管理员发起支付宝官方退款成功',
  }),
  '管理员发起支付宝官方退款成功',
);
assert.equal(getBusinessLogExpandedDetailText({ type: 6, content: '' }), null);
assert.equal(
  getBusinessLogExpandedDetailText({ type: 2, content: 'consume' }),
  null,
);

console.log('usageLogDisplayRules tests passed');
