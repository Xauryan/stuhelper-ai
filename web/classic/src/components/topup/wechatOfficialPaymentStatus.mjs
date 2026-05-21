export const getTopupStatusFromPage = (payload, orderId) => {
  const items = payload?.data?.items || payload?.items || [];
  const order = items.find((item) => item?.trade_no === orderId);
  return order?.status || '';
};

export const getOfficialWechatStatus = (payload) => {
  return payload?.data?.status || payload?.status || '';
};

export const getWechatOfficialQrPaymentHint = (fallback) => {
  return fallback === 'native'
    ? '当前未使用微信 H5，已切换为扫码支付'
    : '请使用微信扫码完成支付';
};

export const DEFAULT_OFFICIAL_PAYMENT_ORDER_TIMEOUT_SECONDS = 600;

export const normalizeOfficialPaymentOrderTimeoutSeconds = (value) => {
  const seconds = Number(value);
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return DEFAULT_OFFICIAL_PAYMENT_ORDER_TIMEOUT_SECONDS;
  }
  return Math.floor(seconds);
};

export const formatOfficialPaymentOrderValidity = (value) => {
  const totalSeconds = normalizeOfficialPaymentOrderTimeoutSeconds(value);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const parts = [];
  if (hours > 0) parts.push(`${hours} 小时`);
  if (minutes > 0) parts.push(`${minutes} 分钟`);
  if (seconds > 0 || parts.length === 0) parts.push(`${seconds} 秒`);
  return parts.join(' ');
};

const padTime = (value) => String(value).padStart(2, '0');

export const formatWechatQrCountdown = (value) => {
  const totalSeconds = Math.max(0, Math.floor(Number(value) || 0));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${padTime(hours)} 小时 ${padTime(minutes)} 分 ${padTime(seconds)} 秒`;
  }
  return `${padTime(minutes)} 分 ${padTime(seconds)} 秒`;
};

export const shouldBlockOfficialWechatMobilePayment = (
  payment,
  isMobileScene,
) => payment === 'wxpay_official' && Boolean(isMobileScene);
