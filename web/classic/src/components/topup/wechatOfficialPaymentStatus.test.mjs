import assert from 'node:assert/strict';
import {
  formatOfficialPaymentOrderValidity,
  formatWechatQrCountdown,
  getOfficialWechatStatus,
  getTopupStatusFromPage,
  getWechatOfficialQrPaymentHint,
  normalizeOfficialPaymentOrderTimeoutSeconds,
  shouldBlockOfficialWechatMobilePayment,
} from './wechatOfficialPaymentStatus.mjs';

assert.equal(
  getTopupStatusFromPage(
    {
      data: {
        items: [
          { trade_no: 'WX_1_pending', status: 'pending' },
          { trade_no: 'WX_1_success', status: 'success' },
        ],
      },
    },
    'WX_1_success',
  ),
  'success',
);
assert.equal(
  getTopupStatusFromPage({ data: { items: [] } }, 'WX_1_missing'),
  '',
);
assert.equal(
  getOfficialWechatStatus({
    data: {
      status: 'success',
      wechat_state: 'SUCCESS',
    },
  }),
  'success',
);
assert.equal(
  getWechatOfficialQrPaymentHint('native'),
  '当前未使用微信 H5，已切换为扫码支付',
);
assert.equal(getWechatOfficialQrPaymentHint(''), '请使用微信扫码完成支付');
assert.equal(normalizeOfficialPaymentOrderTimeoutSeconds(undefined), 600);
assert.equal(normalizeOfficialPaymentOrderTimeoutSeconds(0), 600);
assert.equal(normalizeOfficialPaymentOrderTimeoutSeconds('90.9'), 90);
assert.equal(formatOfficialPaymentOrderValidity(600), '10 分钟');
assert.equal(formatOfficialPaymentOrderValidity(90), '1 分钟 30 秒');
assert.equal(formatOfficialPaymentOrderValidity(3661), '1 小时 1 分钟 1 秒');
assert.equal(formatWechatQrCountdown(600), '10 分 00 秒');
assert.equal(formatWechatQrCountdown(598), '09 分 58 秒');
assert.equal(formatWechatQrCountdown(3661), '01 小时 01 分 01 秒');
assert.equal(
  shouldBlockOfficialWechatMobilePayment('wxpay_official', true),
  true,
);
assert.equal(
  shouldBlockOfficialWechatMobilePayment('alipay_official', true),
  false,
);
assert.equal(
  shouldBlockOfficialWechatMobilePayment('wxpay_official', false),
  false,
);

console.log('wechatOfficialPaymentStatus tests passed');
