import assert from 'node:assert/strict';
import {
  buildOfficialChinaPaymentOptions,
  hasSubmittedOrStoredOfficialChinaPaymentValue,
  normalizeOfficialChinaUnitPrice,
} from './officialChinaPaymentSettings.js';

assert.equal(normalizeOfficialChinaUnitPrice(7.23), '7.23');
assert.equal(normalizeOfficialChinaUnitPrice('7.2'), '7.20');

assert.equal(
  hasSubmittedOrStoredOfficialChinaPaymentValue(
    { AlipayOfficialAlipayPublicKey: '' },
    { AlipayOfficialAlipayPublicKey: 'stored-public-key' },
    'AlipayOfficialAlipayPublicKey',
  ),
  true,
);

assert.equal(
  hasSubmittedOrStoredOfficialChinaPaymentValue(
    { AlipayOfficialAlipayPublicKey: '' },
    { AlipayOfficialAlipayPublicKey: '' },
    'AlipayOfficialAlipayPublicKey',
  ),
  false,
);

const retainedOptions = buildOfficialChinaPaymentOptions(
  {
    AlipayOfficialEnabled: true,
    AlipayOfficialSandbox: false,
    AlipayOfficialAppID: 'app-id',
    AlipayOfficialPrivateKey: '',
    AlipayOfficialAlipayPublicKey: '',
    AlipayOfficialUnitPrice: 7.23,
    AlipayOfficialMinTopUp: 1,
    WechatPayOfficialEnabled: false,
    WechatPayOfficialUnitPrice: '8',
    WechatPayOfficialMinTopUp: 1,
  },
  {
    AlipayOfficialPrivateKey: 'stored-private-key',
    AlipayOfficialAlipayPublicKey: 'stored-public-key',
  },
);

assert.equal(
  retainedOptions.some((option) => option.key === 'AlipayOfficialPrivateKey'),
  false,
);
assert.equal(
  retainedOptions.some(
    (option) => option.key === 'AlipayOfficialAlipayPublicKey',
  ),
  false,
);
assert.equal(
  retainedOptions.find((option) => option.key === 'AlipayOfficialUnitPrice')
    ?.value,
  '7.23',
);
assert.equal(
  retainedOptions.find((option) => option.key === 'WechatPayOfficialUnitPrice')
    ?.value,
  '8.00',
);

const clearingOptions = buildOfficialChinaPaymentOptions(
  {
    AlipayOfficialEnabled: true,
    AlipayOfficialAlipayPublicKey: '',
    AlipayOfficialUnitPrice: 7,
    WechatPayOfficialUnitPrice: 8,
  },
  {},
);
assert.equal(
  clearingOptions.find(
    (option) => option.key === 'AlipayOfficialAlipayPublicKey',
  )?.value,
  '',
);
