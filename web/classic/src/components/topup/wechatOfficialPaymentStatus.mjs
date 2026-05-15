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
