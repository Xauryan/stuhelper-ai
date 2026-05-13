export const sensitiveOfficialChinaPaymentFields = new Set([
  'AlipayOfficialPrivateKey',
  'WechatPayOfficialAPIv3Key',
  'WechatPayOfficialPrivateKey',
]);

export const optionalRetainedOfficialChinaPaymentFields = new Set([
  ...sensitiveOfficialChinaPaymentFields,
  'AlipayOfficialAlipayPublicKey',
  'WechatPayOfficialPlatformPublicKey',
]);

export const officialChinaPaymentOptionKeys = [
  'AlipayOfficialEnabled',
  'AlipayOfficialSandbox',
  'AlipayOfficialAppID',
  'AlipayOfficialPrivateKey',
  'AlipayOfficialAlipayPublicKey',
  'AlipayOfficialAppCertSN',
  'AlipayOfficialRootCertSN',
  'AlipayOfficialAlipayCertSN',
  'AlipayOfficialNotifyURL',
  'AlipayOfficialReturnURL',
  'AlipayOfficialUnitPrice',
  'AlipayOfficialMinTopUp',
  'WechatPayOfficialEnabled',
  'WechatPayOfficialAppID',
  'WechatPayOfficialMchID',
  'WechatPayOfficialCertificateSerial',
  'WechatPayOfficialAPIv3Key',
  'WechatPayOfficialPrivateKey',
  'WechatPayOfficialPlatformPublicKey',
  'WechatPayOfficialNotifyURL',
  'WechatPayOfficialReturnURL',
  'WechatPayOfficialUnitPrice',
  'WechatPayOfficialMinTopUp',
];

export function hasStoredOfficialChinaPaymentValue(options, key) {
  return String(options?.[key] || '').trim() !== '';
}

export function hasSubmittedOrStoredOfficialChinaPaymentValue(
  values,
  options,
  key,
) {
  return (
    String(values?.[key] || '').trim() !== '' ||
    hasStoredOfficialChinaPaymentValue(options, key)
  );
}

export function normalizeOfficialChinaUnitPrice(value) {
  const price = Number(value);
  if (!Number.isFinite(price)) return '';
  return price.toFixed(2);
}

export function buildOfficialChinaPaymentOptions(values, options = {}) {
  return officialChinaPaymentOptionKeys
    .map((key) => {
      let value = values[key];
      if (typeof value === 'boolean') value = value ? 'true' : 'false';
      if (key.endsWith('URL')) value = String(value || '').replace(/\/+$/, '');
      if (
        key === 'AlipayOfficialUnitPrice' ||
        key === 'WechatPayOfficialUnitPrice'
      ) {
        value = normalizeOfficialChinaUnitPrice(value);
      }
      if (value === undefined || value === null) value = '';
      return { key, value: String(value) };
    })
    .filter((item) => {
      if (!optionalRetainedOfficialChinaPaymentFields.has(item.key)) {
        return true;
      }
      return (
        item.value.trim() !== '' ||
        !hasStoredOfficialChinaPaymentValue(options, item.key)
      );
    });
}
