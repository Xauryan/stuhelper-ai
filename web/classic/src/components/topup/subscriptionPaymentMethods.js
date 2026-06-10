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
export function getEpayMethods(payMethods = []) {
  return (payMethods || []).filter(
    (method) =>
      method?.type &&
      method.type !== 'stripe' &&
      method.type !== 'creem' &&
      method.type !== 'alipay_official' &&
      method.type !== 'wxpay_official' &&
      method.type !== 'alipay_self_serve' &&
      method.type !== 'wxpay_self_serve',
  );
}

export function getOfficialAlipayMethod(payMethods = []) {
  return (
    (payMethods || []).find((method) => method?.type === 'alipay_official') ||
    null
  );
}

export function getOfficialWechatPayMethod(payMethods = []) {
  return (
    (payMethods || []).find((method) => method?.type === 'wxpay_official') ||
    null
  );
}

function normalizeUnitPrice(unitPrice) {
  const value = Number(unitPrice);
  return Number.isFinite(value) && value > 0 ? value : null;
}

function normalizeServiceFeePercent(percent) {
  const value = Number(percent);
  return Number.isFinite(value) && value > 0 ? value : 0;
}

function getSelfServeMethod(payMethods, type) {
  return (payMethods || []).find((method) => method?.type === type) || null;
}

export function getSelfServeMethods({
  payMethods = [],
  selfServeQrCodes = {},
  selfServeUnitPrice,
} = {}) {
  const configuredUnitPrice = normalizeUnitPrice(selfServeUnitPrice);

  const candidates = [
    {
      type: 'alipay_self_serve',
      name: '支付宝自助',
      color: 'rgba(var(--semi-blue-5), 1)',
    },
    {
      type: 'wxpay_self_serve',
      name: '微信自助',
      color: 'rgba(var(--semi-green-5), 1)',
    },
  ];

  return candidates
    .map((candidate) => {
      const method = getSelfServeMethod(payMethods, candidate.type);
      const unitPrice =
        configuredUnitPrice ?? normalizeUnitPrice(method?.unit_price);
      const qrCode = selfServeQrCodes?.[candidate.type] || '';
      if (!qrCode) {
        return null;
      }
      return {
        key: `self_serve:${candidate.type}`,
        type: candidate.type,
        provider: 'self_serve',
        name: method?.name || candidate.name,
        unitPrice,
        service_fee_percent: 0,
        icon: method?.icon,
        color: method?.color || candidate.color,
        qrCode,
        raw: method,
      };
    })
    .filter(Boolean);
}

export function buildSubscriptionPaymentMethods({
  plan,
  payMethods = [],
  epayMethods = [],
  epayUnitPrice,
  selfServeQrCodes = {},
  selfServeUnitPrice,
  enableOnlineTopUp = false,
  enableStripeTopUp = false,
  enableCreemTopUp = false,
  enableAlipayOfficialTopUp = false,
  enableWechatPayOfficialTopUp = false,
  enableSelfServeTopUp = false,
  hasAlipayOfficial = false,
  hasWechatPayOfficial = false,
} = {}) {
  const methods = [];

  if (enableStripeTopUp && plan?.stripe_price_id) {
    const stripeMethod = (payMethods || []).find(
      (method) => method?.type === 'stripe',
    );
    methods.push({
      key: 'stripe',
      type: 'stripe',
      provider: 'stripe',
      name: stripeMethod?.name || 'Stripe',
      unitPrice: normalizeUnitPrice(stripeMethod?.unit_price),
      service_fee_percent: normalizeServiceFeePercent(
        stripeMethod?.service_fee_percent ?? stripeMethod?.serviceFeePercent,
      ),
      icon: stripeMethod?.icon,
      color: stripeMethod?.color,
    });
  }

  if (enableCreemTopUp && plan?.creem_product_id) {
    const creemMethod = (payMethods || []).find(
      (method) => method?.type === 'creem',
    );
    methods.push({
      key: 'creem',
      type: 'creem',
      provider: 'creem',
      name: creemMethod?.name || 'Creem',
      unitPrice: normalizeUnitPrice(creemMethod?.unit_price),
      service_fee_percent: normalizeServiceFeePercent(
        creemMethod?.service_fee_percent ?? creemMethod?.serviceFeePercent,
      ),
      icon: creemMethod?.icon,
      color: creemMethod?.color,
    });
  }

  if (enableAlipayOfficialTopUp && hasAlipayOfficial) {
    const alipayMethod = getOfficialAlipayMethod(payMethods);
    methods.push({
      key: 'alipay_official',
      type: 'alipay_official',
      provider: 'alipay_official',
      name: alipayMethod?.name || '支付宝',
      unitPrice: normalizeUnitPrice(alipayMethod?.unit_price),
      service_fee_percent: normalizeServiceFeePercent(
        alipayMethod?.service_fee_percent ?? alipayMethod?.serviceFeePercent,
      ),
      icon: alipayMethod?.icon,
      color: alipayMethod?.color,
    });
  }

  if (enableWechatPayOfficialTopUp && hasWechatPayOfficial) {
    const wechatMethod = getOfficialWechatPayMethod(payMethods);
    methods.push({
      key: 'wxpay_official',
      type: 'wxpay_official',
      provider: 'wxpay_official',
      name: wechatMethod?.name || '微信',
      unitPrice: normalizeUnitPrice(wechatMethod?.unit_price),
      service_fee_percent: normalizeServiceFeePercent(
        wechatMethod?.service_fee_percent ?? wechatMethod?.serviceFeePercent,
      ),
      icon: wechatMethod?.icon,
      color: wechatMethod?.color,
    });
  }

  if (enableOnlineTopUp) {
    (epayMethods || []).forEach((method) => {
      if (!method?.type) return;
      methods.push({
        key: `epay:${method.type}`,
        type: method.type,
        provider: 'epay',
        name: method.name || method.type,
        unitPrice: normalizeUnitPrice(method.unit_price ?? epayUnitPrice),
        service_fee_percent: normalizeServiceFeePercent(
          method.service_fee_percent ?? method.serviceFeePercent,
        ),
        icon: method.icon,
        color: method.color,
        raw: method,
      });
    });
  }

  if (enableSelfServeTopUp) {
    methods.push(
      ...getSelfServeMethods({
        payMethods,
        selfServeQrCodes,
        selfServeUnitPrice,
      }),
    );
  }

  return methods;
}
