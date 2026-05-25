import assert from 'node:assert/strict';
import { buildRechargeAmountDisplay } from './rechargeAmountDisplay.js';

const alipayDisplay = buildRechargeAmountDisplay({
  preset: { value: 10 },
  priceRatio: 1,
  discountConfig: {},
  currencyConfig: { symbol: '$', rate: 1, type: 'USD' },
  usdExchangeRate: 7,
  selectedPaymentMethod: 'alipay_official',
  serviceFeePercent: 0.6,
});

assert.equal(alipayDisplay.displayValue, 10);
assert.equal(alipayDisplay.paymentSymbol, '¥');
assert.equal(alipayDisplay.displayActualPay.toFixed(2), '10.06');
assert.equal(alipayDisplay.fee.toFixed(2), '0.06');
assert.equal(alipayDisplay.showSavings, false);

const discountedAlipayDisplay = buildRechargeAmountDisplay({
  preset: { value: 100 },
  priceRatio: 1,
  discountConfig: { 100: 0.95 },
  currencyConfig: { symbol: '$', rate: 1, type: 'USD' },
  usdExchangeRate: 7,
  selectedPaymentMethod: 'alipay_official',
  serviceFeePercent: 0.6,
});

assert.equal(discountedAlipayDisplay.paymentSymbol, '¥');
assert.equal(discountedAlipayDisplay.displayActualPay.toFixed(2), '95.57');
assert.equal(discountedAlipayDisplay.displaySave.toFixed(2), '5.03');
assert.equal(discountedAlipayDisplay.showSavings, true);

const stripeDisplay = buildRechargeAmountDisplay({
  preset: { value: 10 },
  priceRatio: 7,
  discountConfig: {},
  currencyConfig: { symbol: '$', rate: 1, type: 'USD' },
  usdExchangeRate: 7,
  selectedPaymentMethod: 'stripe',
  serviceFeePercent: 0.6,
});

assert.equal(stripeDisplay.paymentSymbol, '$');
assert.equal(stripeDisplay.displayActualPay.toFixed(2), '10.00');
assert.equal(stripeDisplay.showSavings, false);

const ceilDisplay = buildRechargeAmountDisplay({
  preset: { value: 1 },
  priceRatio: 7.231,
  discountConfig: {},
  currencyConfig: { symbol: '$', rate: 1, type: 'USD' },
  usdExchangeRate: 7,
  selectedPaymentMethod: 'alipay_official',
});

assert.equal(ceilDisplay.displayActualPay.toFixed(2), '7.24');
