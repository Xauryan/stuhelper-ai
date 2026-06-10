import assert from 'node:assert/strict';
import { isLegacyQRCodeImageValue } from './qrCodeUtils.js';

assert.equal(
  isLegacyQRCodeImageValue('https://qr.alipay.com/45t165972y9chxii0fm3fe8'),
  false,
);
assert.equal(isLegacyQRCodeImageValue('https://example.com/code.png'), true);
assert.equal(
  isLegacyQRCodeImageValue('https://example.com/code.jpg?version=1'),
  true,
);
assert.equal(isLegacyQRCodeImageValue('data:image/png;base64,Zm9v'), true);
