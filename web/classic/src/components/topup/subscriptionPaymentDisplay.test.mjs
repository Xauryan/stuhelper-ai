import assert from 'node:assert/strict';
import {
  calculateSubscriptionPayAmount,
  formatSubscriptionPayAmount,
} from './subscriptionPaymentDisplay.js';

assert.equal(calculateSubscriptionPayAmount(50, 1.006), 50.3);
assert.equal(calculateSubscriptionPayAmount(50, 1.001), 50.05);
assert.equal(
  calculateSubscriptionPayAmount(50, 1, 0.6, 'alipay_official'),
  50.3,
);
assert.equal(calculateSubscriptionPayAmount(50, 1, 0.6, 'stripe'), 50);
assert.equal(calculateSubscriptionPayAmount(50, 'bad'), null);
assert.equal(
  formatSubscriptionPayAmount({
    priceAmount: 50,
    unitPrice: 1,
    serviceFeePercent: 0.6,
    paymentMethod: 'alipay_official',
  }),
  '¥50.30',
);
assert.equal(
  formatSubscriptionPayAmount({
    priceAmount: 50,
    unitPrice: 1.2,
  }),
  '¥60.00',
);
assert.equal(
  formatSubscriptionPayAmount({
    priceAmount: 12.5,
    symbol: '$',
    rate: 7.2,
  }),
  '$90',
);
assert.equal(
  formatSubscriptionPayAmount({
    priceAmount: 12.34,
    symbol: '$',
    rate: 1,
  }),
  '$12.34',
);

console.log('subscriptionPaymentDisplay tests passed');
