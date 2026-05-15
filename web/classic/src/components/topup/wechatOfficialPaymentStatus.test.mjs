import assert from 'node:assert/strict';
import {
  getOfficialWechatStatus,
  getTopupStatusFromPage,
  getWechatOfficialQrPaymentHint,
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

console.log('wechatOfficialPaymentStatus tests passed');
