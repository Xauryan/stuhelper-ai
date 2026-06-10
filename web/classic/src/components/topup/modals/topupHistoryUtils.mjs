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
    tradeNo.startsWith('WXSUB') ||
    tradeNo.startsWith('SSSUB') ||
    tradeNo.startsWith('BALANCE__')
  );
};

export const isOfficialPaymentTopup = (record) =>
  record?.payment_provider === 'alipay_official' ||
  record?.payment_method === 'alipay_official' ||
  record?.payment_provider === 'wxpay_official' ||
  record?.payment_method === 'wxpay_official';

export const isSelfServeTopup = (record) =>
  record?.payment_provider === 'self_serve' ||
  record?.payment_method === 'alipay_self_serve' ||
  record?.payment_method === 'wxpay_self_serve';

export const isBalanceTopup = (record) =>
  record?.payment_provider === 'balance' ||
  record?.payment_method === 'balance';

export const isBalanceSubscriptionRefundable = (record) => {
  if (!record) {
    return false;
  }
  const statusAllowsRefund =
    record.status === 'success' || record.status === 'partial_refunded';
  return (
    isBalanceTopup(record) &&
    isSubscriptionTopup(record) &&
    statusAllowsRefund &&
    getRemainingRefundMoney(record) > 0
  );
};

export const ADMIN_TOPUP_PAYMENT_METHODS = [
  'admin_add',
  '管理员增加',
  '管理员充值',
];

export const isAdminManagedTopup = (record) =>
  record?.payment_provider === 'admin' ||
  ADMIN_TOPUP_PAYMENT_METHODS.includes(record?.payment_method);

export const getRemainingAdminRefundQuota = (record) => {
  const amount = Number(record?.amount || 0);
  const refundedQuota = Number(record?.refunded_quota || 0);
  return Math.max(0, Math.round(amount - refundedQuota));
};

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

export const isRefundRequestable = (record) => {
  if (!record) {
    return false;
  }
  const statusAllowsRefund =
    record.status === 'success' || record.status === 'partial_refunded';
  return (
    (isOfficialPaymentTopup(record) ||
      isSelfServeRefundable(record) ||
      isBalanceSubscriptionRefundable(record)) &&
    statusAllowsRefund &&
    getRemainingRefundMoney(record) > 0
  );
};

export const isSelfServeRefundable = (record) => {
  if (!record) {
    return false;
  }
  const statusAllowsRefund =
    record.status === 'success' || record.status === 'partial_refunded';
  return (
    isSelfServeTopup(record) &&
    record.audit_status === 'approved' &&
    statusAllowsRefund &&
    getRemainingRefundMoney(record) > 0
  );
};

export const isAdminMoneyRefundable = (record) =>
  isOfficialRefundable(record) ||
  isSelfServeRefundable(record) ||
  isBalanceSubscriptionRefundable(record);

export const isAdminManagedTopupRefundable = (record) => {
  if (!record) {
    return false;
  }
  const statusAllowsRefund =
    record.status === 'success' || record.status === 'partial_refunded';
  return (
    isAdminManagedTopup(record) &&
    statusAllowsRefund &&
    getRemainingAdminRefundQuota(record) > 0
  );
};

export const formatCurrency = (value) => Number(value || 0).toFixed(2);

export const TOPUP_PAYMENT_METHOD_LABELS = {
  stripe: 'Stripe',
  creem: 'Creem',
  waffo: 'Waffo',
  waffo_pancake: 'Waffo Pancake',
  alipay: '支付宝',
  wxpay: '微信',
  alipay_official: '支付宝',
  wxpay_official: '微信',
  alipay_self_serve: '支付宝自助',
  wxpay_self_serve: '微信自助',
  balance: '余额',
  admin_add: '管理员充值',
  管理员增加: '管理员充值',
  管理员充值: '管理员充值',
};

export const getTopupPaymentMethodLabel = (paymentMethod) =>
  TOPUP_PAYMENT_METHOD_LABELS[paymentMethod] || paymentMethod || '-';

export const BILLING_PAYMENT_METHOD_FILTERS = [
  { value: '', key: '全部' },
  { value: 'alipay_official', key: '支付宝' },
  { value: 'wxpay_official', key: '微信' },
  { value: 'self_serve', key: '自助充值' },
  { value: 'balance', key: '余额' },
  { value: 'admin_add', key: '管理员充值' },
];
