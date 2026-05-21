export const getRemainingRefundMoney = (record) => {
  const money = Number(record?.money || 0);
  const refundedMoney = Number(record?.refunded_money || 0);
  return Math.max(0, Math.round((money - refundedMoney) * 100) / 100);
};

export const isSubscriptionTopup = (record) => {
  if (!record) {
    return false;
  }
  if (Number(record.amount || 0) !== 0) {
    return false;
  }
  const tradeNo = String(record.trade_no || '').toUpperCase();
  return (
    tradeNo.startsWith('SUB') ||
    tradeNo.startsWith('ALIPAYSUB') ||
    tradeNo.startsWith('WXSUB')
  );
};

export const isOfficialPaymentTopup = (record) =>
  record?.payment_provider === 'alipay_official' ||
  record?.payment_method === 'alipay_official' ||
  record?.payment_provider === 'wxpay_official' ||
  record?.payment_method === 'wxpay_official';

export const canAdminCompleteTopup = (record) => {
  if (!record) {
    return false;
  }
  if (record.status === 'pending') {
    return !isSubscriptionTopup(record) || isOfficialPaymentTopup(record);
  }
  return record.status === 'expired' && isOfficialPaymentTopup(record);
};

export const isAlipayOfficialRefundable = (record) => {
  if (!record) {
    return false;
  }
  if (isSubscriptionTopup(record)) {
    return false;
  }
  const isAlipayOfficial =
    record.payment_provider === 'alipay_official' ||
    record.payment_method === 'alipay_official';
  const statusAllowsRefund =
    record.status === 'success' || record.status === 'partial_refunded';
  return (
    isAlipayOfficial &&
    statusAllowsRefund &&
    getRemainingRefundMoney(record) > 0
  );
};

export const isOfficialRefundable = (record) => {
  if (!record) {
    return false;
  }
  const statusAllowsRefund =
    record.status === 'success' || record.status === 'partial_refunded';
  return (
    isOfficialPaymentTopup(record) &&
    statusAllowsRefund &&
    getRemainingRefundMoney(record) > 0
  );
};

export const formatCurrency = (value) => Number(value || 0).toFixed(2);

export const BILLING_PAYMENT_METHOD_FILTERS = [
  { value: '', key: '全部' },
  { value: 'alipay_official', key: '支付宝' },
  { value: 'wxpay_official', key: '微信' },
];
