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
export const sensitiveOfficialChinaPaymentFields = new Set([
  'AlipayOfficialAppAuthToken',
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
  'AlipayOfficialAppAuthToken',
  'AlipayOfficialPrivateKey',
  'AlipayOfficialAlipayPublicKey',
  'AlipayOfficialAppCertSN',
  'AlipayOfficialRootCertSN',
  'AlipayOfficialAlipayCertSN',
  'AlipayOfficialNotifyURL',
  'AlipayOfficialReturnURL',
  'AlipayOfficialUnitPrice',
  'AlipayOfficialServiceFeePercent',
  'AlipayOfficialMinTopUp',
  'AlipayOfficialOrderTimeoutSec',
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
  'WechatPayOfficialServiceFeePercent',
  'WechatPayOfficialMinTopUp',
  'WechatPayOfficialOrderTimeoutSec',
];

export function hasStoredOfficialChinaPaymentValue(options, key) {
  if (sensitiveOfficialChinaPaymentFields.has(key)) {
    return String(options?.[`${key}Configured`] || '').trim() === 'true';
  }
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
  return price.toFixed(3);
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
      if (
        key === 'AlipayOfficialServiceFeePercent' ||
        key === 'WechatPayOfficialServiceFeePercent'
      ) {
        value = String(Number(value || 0));
      }
      if (value === undefined || value === null) value = '';
      return { key, value: String(value) };
    })
    .filter((item) => {
      if (!optionalRetainedOfficialChinaPaymentFields.has(item.key)) {
        return true;
      }
      if (sensitiveOfficialChinaPaymentFields.has(item.key)) {
        return item.value.trim() !== '';
      }
      return (
        item.value.trim() !== '' ||
        !hasStoredOfficialChinaPaymentValue(options, item.key)
      );
    });
}
