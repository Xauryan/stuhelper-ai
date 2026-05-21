import assert from 'node:assert/strict';
import {
  buildOfficialChinaPaymentOptions,
  hasSubmittedOrStoredOfficialChinaPaymentValue,
  normalizeOfficialChinaUnitPrice,
} from './officialChinaPaymentSettings.js';

assert.equal(normalizeOfficialChinaUnitPrice(7.231), '7.231');
assert.equal(normalizeOfficialChinaUnitPrice('7.2'), '7.200');

assert.equal(
  hasSubmittedOrStoredOfficialChinaPaymentValue(
    { AlipayOfficialAppAuthToken: '' },
    { AlipayOfficialAppAuthTokenConfigured: 'true' },
    'AlipayOfficialAppAuthToken',
  ),
  true,
);

assert.equal(
  hasSubmittedOrStoredOfficialChinaPaymentValue(
    { AlipayOfficialPrivateKey: '' },
    { AlipayOfficialPrivateKeyConfigured: 'false' },
    'AlipayOfficialPrivateKey',
  ),
  false,
);

assert.equal(
  hasSubmittedOrStoredOfficialChinaPaymentValue(
    { AlipayOfficialAlipayPublicKey: '' },
    { AlipayOfficialAlipayPublicKey: 'stored-public-key' },
    'AlipayOfficialAlipayPublicKey',
  ),
  true,
);

const retainedOptions = buildOfficialChinaPaymentOptions(
  {
    AlipayOfficialEnabled: true,
    AlipayOfficialSandbox: false,
    AlipayOfficialAppID: 'app-id',
    AlipayOfficialAppAuthToken: '',
    AlipayOfficialPrivateKey: '',
    AlipayOfficialAlipayPublicKey: '',
    AlipayOfficialUnitPrice: 7.231,
    AlipayOfficialMinTopUp: 1,
    AlipayOfficialOrderTimeoutSec: 900,
    WechatPayOfficialEnabled: false,
    WechatPayOfficialUnitPrice: '8',
    WechatPayOfficialMinTopUp: 1,
    WechatPayOfficialOrderTimeoutSec: 600,
  },
  {
    AlipayOfficialAppAuthTokenConfigured: 'true',
    AlipayOfficialPrivateKeyConfigured: 'true',
    AlipayOfficialAlipayPublicKey: 'stored-public-key',
  },
);

assert.equal(
  retainedOptions.some((option) => option.key === 'AlipayOfficialAppAuthToken'),
  false,
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
  '7.231',
);
assert.equal(
  retainedOptions.find((option) => option.key === 'WechatPayOfficialUnitPrice')
    ?.value,
  '8.000',
);
assert.equal(
  retainedOptions.find(
    (option) => option.key === 'AlipayOfficialOrderTimeoutSec',
  )?.value,
  '900',
);
assert.equal(
  retainedOptions.find(
    (option) => option.key === 'WechatPayOfficialOrderTimeoutSec',
  )?.value,
  '600',
);

const sensitiveRetainedOptions = buildOfficialChinaPaymentOptions(
  {
    AlipayOfficialEnabled: true,
    AlipayOfficialAppAuthToken: '',
    AlipayOfficialPrivateKey: '',
    AlipayOfficialAlipayPublicKey: 'public-key',
    AlipayOfficialUnitPrice: 7.231,
    WechatPayOfficialAPIv3Key: '',
    WechatPayOfficialPrivateKey: '',
    WechatPayOfficialPlatformPublicKey: 'wechat-platform-public-key',
    WechatPayOfficialUnitPrice: 8.123,
  },
  {},
);
assert.equal(
  sensitiveRetainedOptions.some(
    (option) => option.key === 'AlipayOfficialAppAuthToken',
  ),
  false,
);
assert.equal(
  sensitiveRetainedOptions.some(
    (option) => option.key === 'AlipayOfficialPrivateKey',
  ),
  false,
);
assert.equal(
  sensitiveRetainedOptions.some(
    (option) => option.key === 'WechatPayOfficialAPIv3Key',
  ),
  false,
);
assert.equal(
  sensitiveRetainedOptions.some(
    (option) => option.key === 'WechatPayOfficialPrivateKey',
  ),
  false,
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
