export function getEpayMethods(payMethods = []) {
  return (payMethods || []).filter(
    (method) =>
      method?.type &&
      method.type !== 'stripe' &&
      method.type !== 'creem' &&
      method.type !== 'alipay_official' &&
      method.type !== 'wxpay_official',
  );
}

export function getOfficialAlipayMethod(payMethods = []) {
  return (
    (payMethods || []).find((method) => method?.type === 'alipay_official') ||
    null
  );
}
