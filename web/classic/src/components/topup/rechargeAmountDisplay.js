/*
Copyright (C) 2025 Xauryan

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@xauryan.com
*/

const CNY_PAYMENT_METHODS = new Set([
  'alipay',
  'wxpay',
  'alipay_official',
  'wxpay_official',
]);

function ceilToCents(amount) {
  return Math.ceil((Number(amount) || 0) * 100 - 1e-9) / 100;
}

function applyServiceFee(amount, serviceFeePercent) {
  const feePercent = Number(serviceFeePercent);
  if (!Number.isFinite(feePercent) || feePercent <= 0) {
    return ceilToCents(amount);
  }
  return ceilToCents(ceilToCents(amount) * (1 + feePercent / 100));
}

function shouldApplyServiceFee(paymentMethod) {
  return paymentMethod !== 'stripe' && paymentMethod !== 'creem';
}

function getConfiguredDiscount(preset, discountConfig) {
  if (preset && Object.prototype.hasOwnProperty.call(preset, 'discount')) {
    const presetDiscount = Number(preset.discount);
    if (Number.isFinite(presetDiscount) && presetDiscount > 0) {
      return presetDiscount;
    }
  }

  const discountByAmount = discountConfig || {};
  const amountKey = preset?.value;
  if (
    amountKey !== undefined &&
    Object.prototype.hasOwnProperty.call(discountByAmount, amountKey)
  ) {
    const configuredDiscount = Number(discountByAmount[amountKey]);
    if (Number.isFinite(configuredDiscount) && configuredDiscount > 0) {
      return configuredDiscount;
    }
  }

  return null;
}

export function buildRechargeAmountDisplay({
  preset,
  priceRatio,
  discountConfig,
  currencyConfig,
  usdExchangeRate,
  selectedPaymentMethod,
  serviceFeePercent,
}) {
  const amount = Number(preset?.value) || 0;
  const unitPrice = Number(priceRatio) || 0;
  const originalPrice = amount * unitPrice;
  const configuredDiscount = getConfiguredDiscount(preset, discountConfig);
  const discount = configuredDiscount ?? 1;
  const hasDiscount = configuredDiscount !== null && discount < 1;
  const paymentMethod = selectedPaymentMethod || '';
  const activeServiceFeePercent = shouldApplyServiceFee(paymentMethod)
    ? serviceFeePercent
    : 0;
  const effectivePay = ceilToCents(originalPrice * discount);
  const actualPay = applyServiceFee(effectivePay, activeServiceFeePercent);
  const originalPay = applyServiceFee(originalPrice, activeServiceFeePercent);
  const save = Math.max(0, originalPay - actualPay);
  const fee = Math.max(0, actualPay - effectivePay);
  const displayType = currencyConfig?.type || 'USD';
  const exchangeRate = Number(usdExchangeRate) || 7;
  let displayValue = amount;

  if (displayType === 'CNY') {
    displayValue = amount * exchangeRate;
  } else if (displayType === 'CUSTOM') {
    displayValue = amount * (Number(currencyConfig?.rate) || 1);
  }

  let paymentSymbol = currencyConfig?.symbol || '$';
  let displayActualPay = actualPay;
  let displaySave = save;

  if (CNY_PAYMENT_METHODS.has(paymentMethod)) {
    paymentSymbol = '¥';
  } else if (paymentMethod === 'stripe') {
    paymentSymbol = '$';
    displayActualPay = actualPay / exchangeRate;
    displaySave = save / exchangeRate;
  } else if (displayType === 'USD') {
    paymentSymbol = '$';
    displayActualPay = actualPay / exchangeRate;
    displaySave = save / exchangeRate;
  } else if (displayType === 'CUSTOM') {
    displayActualPay =
      (actualPay / exchangeRate) * (Number(currencyConfig?.rate) || 1);
    displaySave = (save / exchangeRate) * (Number(currencyConfig?.rate) || 1);
  }

  return {
    displayValue,
    paymentSymbol,
    displayActualPay,
    displaySave,
    fee,
    discount,
    hasDiscount,
    showSavings: hasDiscount,
  };
}
