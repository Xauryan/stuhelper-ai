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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Badge,
  Banner,
  Button,
  Card,
  Divider,
  Modal,
  Select,
  Skeleton,
  Space,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  API,
  getQuotaPerUnit,
  getSubscriptionModelLimits,
  showError,
  showSuccess,
  renderQuota,
} from '../../helpers';
import {
  getCurrencyConfig,
  normalizeHttpsImageUrl,
} from '../../helpers/render';
import { CreditCard, RefreshCw, Sparkles } from 'lucide-react';
import { SiAlipay, SiStripe, SiWechat } from 'react-icons/si';
import SubscriptionPurchaseModal from './modals/SubscriptionPurchaseModal';
import WechatOfficialQrPaymentModal from './modals/WechatOfficialQrPaymentModal';
import SelfServeSubscriptionModal from './modals/SelfServeSubscriptionModal';
import {
  formatSubscriptionDuration,
  formatSubscriptionResetPeriod,
} from '../../helpers/subscriptionFormat';
import {
  buildSubscriptionPaymentMethods,
  getEpayMethods,
  getOfficialAlipayMethod,
  getOfficialWechatPayMethod,
} from './subscriptionPaymentMethods';
import { shouldHighlightSubscriptionPlan } from './subscriptionPlanDisplay';
import {
  calculateSubscriptionPayAmount,
  formatSubscriptionPayAmount,
} from './subscriptionPaymentDisplay';
import {
  getOfficialWechatStatus,
  getTopupStatusFromPage,
  normalizeOfficialPaymentOrderTimeoutSeconds,
  shouldBlockOfficialWechatMobilePayment,
} from './wechatOfficialPaymentStatus.mjs';

const { Text } = Typography;

const isApiSuccess = (data) =>
  data?.success === true || data?.message === 'success';

// 提交易支付表单
function submitEpayForm({ url, params }) {
  const form = document.createElement('form');
  form.action = url;
  form.method = 'POST';
  const isSafari =
    navigator.userAgent.indexOf('Safari') > -1 &&
    navigator.userAgent.indexOf('Chrome') < 1;
  if (!isSafari) form.target = '_blank';
  Object.keys(params || {}).forEach((key) => {
    const input = document.createElement('input');
    input.type = 'hidden';
    input.name = key;
    input.value = params[key];
    form.appendChild(input);
  });
  document.body.appendChild(form);
  form.submit();
  document.body.removeChild(form);
}

const SubscriptionPlansCard = ({
  t,
  loading = false,
  plans = [],
  payMethods = [],
  enableOnlineTopUp = false,
  enableStripeTopUp = false,
  enableCreemTopUp = false,
  enableAlipayOfficialTopUp = false,
  enableWechatPayOfficialTopUp = false,
  enableSelfServeTopUp = false,
  selfServeQrCodes = {},
  selfServeLimits = {},
  priceRatio,
  getPaymentServiceFeePercent,
  billingPreference,
  onChangeBillingPreference,
  activeSubscriptions = [],
  allSubscriptions = [],
  reloadSubscriptionSelf,
  userQuota = 0,
  reloadUserQuota,
  getPaymentOrderTimeoutSeconds,
  withCard = true,
}) => {
  const [open, setOpen] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState(null);
  const [paying, setPaying] = useState(false);
  const [selectedPaymentKey, setSelectedPaymentKey] = useState('');
  const [refreshing, setRefreshing] = useState(false);
  const [wechatQrOpen, setWechatQrOpen] = useState(false);
  const [wechatQrCodeUrl, setWechatQrCodeUrl] = useState('');
  const [wechatQrOrderId, setWechatQrOrderId] = useState('');
  const [wechatQrFallback, setWechatQrFallback] = useState('');
  const [wechatQrCreatedAt, setWechatQrCreatedAt] = useState(0);
  const [wechatQrOrderTimeoutSeconds, setWechatQrOrderTimeoutSeconds] =
    useState(600);
  const [selfServeOpen, setSelfServeOpen] = useState(false);
  const [selfServeTransactionNo, setSelfServeTransactionNo] = useState('');
  const [selfServeConfirmed, setSelfServeConfirmed] = useState(false);
  const [selfServeSubmitting, setSelfServeSubmitting] = useState(false);
  const wechatQrPollTimerRef = useRef(null);

  const epayMethods = useMemo(() => getEpayMethods(payMethods), [payMethods]);
  const alipayOfficialMethod = useMemo(
    () => getOfficialAlipayMethod(payMethods),
    [payMethods],
  );
  const hasAlipayOfficial = enableAlipayOfficialTopUp && !!alipayOfficialMethod;
  const wechatPayOfficialMethod = useMemo(
    () => getOfficialWechatPayMethod(payMethods),
    [payMethods],
  );
  const hasWechatPayOfficial =
    enableWechatPayOfficialTopUp && !!wechatPayOfficialMethod;

  const selectPlan = (p) => {
    setSelectedPlan(p);
  };

  const closeBuy = () => {
    setOpen(false);
    setPaying(false);
  };

  const closeWechatQrModal = () => {
    setWechatQrOpen(false);
    setWechatQrCodeUrl('');
    setWechatQrOrderId('');
    setWechatQrFallback('');
    setWechatQrCreatedAt(0);
  };

  const closeSelfServeModal = () => {
    setSelfServeOpen(false);
    setSelfServeTransactionNo('');
    setSelfServeConfirmed(false);
  };

  const checkWechatQrOrderStatus = async (orderId) => {
    if (!orderId) {
      return false;
    }
    try {
      const officialRes = await API.post(
        '/api/user/wechat-pay/official/status',
        {
          trade_no: orderId,
        },
      );
      let status = getOfficialWechatStatus(officialRes?.data);
      if (!status) {
        const res = await API.get('/api/user/topup/self', {
          params: {
            p: 1,
            page_size: 1,
            keyword: orderId,
          },
        });
        status = getTopupStatusFromPage(res?.data, orderId);
      }
      if (status === 'success') {
        showSuccess(t('支付成功'));
        closeWechatQrModal();
        reloadSubscriptionSelf?.();
        return true;
      }
      if (
        status === 'failed' ||
        status === 'expired' ||
        status === 'refunded'
      ) {
        showError(t('订单未完成，请重新发起支付'));
        closeWechatQrModal();
        return true;
      }
    } catch (e) {
      // 轮询失败时保持二维码，等待下一次查询。
    }
    return false;
  };

  const subscriptionPaymentMethods = useMemo(() => {
    const methods = [];
    if (selectedPlan?.plan?.allow_balance_pay !== false) {
      methods.push({
        key: 'balance',
        type: 'balance',
        provider: 'balance',
        name: t('余额支付'),
      });
    }
    methods.push(
      ...buildSubscriptionPaymentMethods({
        plan: selectedPlan?.plan,
        payMethods,
        epayMethods,
        epayUnitPrice: priceRatio,
        enableOnlineTopUp,
        enableStripeTopUp,
        enableCreemTopUp,
        enableAlipayOfficialTopUp,
        enableWechatPayOfficialTopUp,
        enableSelfServeTopUp,
        hasAlipayOfficial,
        hasWechatPayOfficial,
        selfServeQrCodes,
        selfServeUnitPrice: selfServeLimits?.unit_price,
      }),
    );
    return methods;
  }, [
    selectedPlan?.plan,
    payMethods,
    epayMethods,
    priceRatio,
    enableOnlineTopUp,
    enableStripeTopUp,
    enableCreemTopUp,
    enableAlipayOfficialTopUp,
    enableWechatPayOfficialTopUp,
    enableSelfServeTopUp,
    hasAlipayOfficial,
    hasWechatPayOfficial,
    selfServeQrCodes,
    selfServeLimits?.unit_price,
    t,
  ]);
  const selectedPaymentMethod = useMemo(
    () =>
      subscriptionPaymentMethods.find(
        (method) => method.key === selectedPaymentKey,
      ) || null,
    [subscriptionPaymentMethods, selectedPaymentKey],
  );
  const quotaPerUnit = Number(getQuotaPerUnit()) || 500000;
  const balanceCost = Math.max(
    0,
    Math.ceil(Number(selectedPlan?.plan?.price_amount || 0) * quotaPerUnit),
  );
  const availableBalance = Math.max(0, Number(userQuota || 0));

  React.useEffect(() => {
    if (!selectedPlan?.plan) {
      setSelectedPaymentKey('');
      return;
    }
    if (
      subscriptionPaymentMethods.some(
        (method) => method.key === selectedPaymentKey,
      )
    ) {
      return;
    }
    setSelectedPaymentKey(subscriptionPaymentMethods[0]?.key || '');
  }, [selectedPlan?.plan, selectedPaymentKey, subscriptionPaymentMethods]);

  const getPlanPriceDisplay = (plan) => {
    const { symbol, rate } = getCurrencyConfig();
    const price = Number(plan?.price_amount || 0);
    const convertedPrice = price * rate;
    const displayPrice = convertedPrice.toFixed(
      Number.isInteger(convertedPrice) ? 0 : 2,
    );
    return { symbol, displayPrice };
  };

  const currencyConfig = getCurrencyConfig();
  const selectedPayAmount = formatSubscriptionPayAmount({
    priceAmount: selectedPlan?.plan?.price_amount || 0,
    symbol: currencyConfig.symbol,
    rate: currencyConfig.rate,
    unitPrice: selectedPaymentMethod?.unitPrice,
    serviceFeePercent:
      selectedPaymentMethod?.service_fee_percent ??
      selectedPaymentMethod?.serviceFeePercent ??
      getPaymentServiceFeePercent?.(selectedPaymentMethod?.type) ??
      0,
    paymentMethod: selectedPaymentMethod?.type,
  });
  const selectedSelfServeExpectedMoney =
    selectedPaymentMethod?.provider === 'self_serve'
      ? calculateSubscriptionPayAmount(
          selectedPlan?.plan?.price_amount || 0,
          selectedPaymentMethod?.unitPrice,
          0,
          selectedPaymentMethod?.type,
        )
      : null;
  const selectedPayAmountDisplay =
    selectedPaymentMethod?.provider === 'balance'
      ? renderQuota(balanceCost)
      : selectedPaymentMethod?.provider === 'self_serve'
        ? selectedSelfServeExpectedMoney !== null
          ? `¥${selectedSelfServeExpectedMoney.toFixed(2)}`
          : t('请先配置自助充值价格')
        : selectedPayAmount;

  const renderPaymentIcon = (method) => {
    if (
      method?.type === 'alipay' ||
      method?.type === 'alipay_official' ||
      method?.type === 'alipay_self_serve'
    ) {
      return <SiAlipay size={18} color='#1677FF' />;
    }
    if (
      method?.type === 'wxpay' ||
      method?.type === 'wxpay_official' ||
      method?.type === 'wxpay_self_serve'
    ) {
      return <SiWechat size={18} color='#07C160' />;
    }
    if (method?.type === 'stripe') {
      return <SiStripe size={18} color='#635BFF' />;
    }
    if (method?.type === 'balance') {
      return <CreditCard size={18} color='var(--semi-color-primary)' />;
    }
    const iconUrl = normalizeHttpsImageUrl(method?.icon);
    if (iconUrl) {
      return (
        <img
          src={iconUrl}
          alt={method.name}
          style={{ width: 18, height: 18, objectFit: 'contain' }}
        />
      );
    }
    return (
      <CreditCard
        size={18}
        color={method?.color || 'var(--semi-color-text-2)'}
      />
    );
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await reloadSubscriptionSelf?.();
    } finally {
      setRefreshing(false);
    }
  };

  const payStripe = async () => {
    if (!selectedPlan?.plan?.stripe_price_id) {
      showError(t('该套餐未配置 Stripe'));
      return;
    }
    setPaying(true);
    try {
      const res = await API.post('/api/subscription/stripe/pay', {
        plan_id: selectedPlan.plan.id,
      });
      if (isApiSuccess(res.data)) {
        window.open(res.data.data?.pay_link, '_blank');
        showSuccess(t('已打开支付页面'));
        closeBuy();
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('支付失败');
        showError(errorMsg);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaying(false);
    }
  };

  const payCreem = async () => {
    if (!selectedPlan?.plan?.creem_product_id) {
      showError(t('该套餐未配置 Creem'));
      return;
    }
    setPaying(true);
    try {
      const res = await API.post('/api/subscription/creem/pay', {
        plan_id: selectedPlan.plan.id,
      });
      if (isApiSuccess(res.data)) {
        window.open(res.data.data?.checkout_url, '_blank');
        showSuccess(t('已打开支付页面'));
        closeBuy();
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('支付失败');
        showError(errorMsg);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaying(false);
    }
  };

  const payEpay = async () => {
    if (!selectedPaymentMethod?.type) {
      showError(t('请选择支付方式'));
      return;
    }
    setPaying(true);
    try {
      const res = await API.post('/api/subscription/epay/pay', {
        plan_id: selectedPlan.plan.id,
        payment_method: selectedPaymentMethod.type,
      });
      if (isApiSuccess(res.data)) {
        submitEpayForm({ url: res.data.url, params: res.data.data });
        showSuccess(t('已发起支付'));
        closeBuy();
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('支付失败');
        showError(errorMsg);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaying(false);
    }
  };

  const payBalance = async () => {
    if (!selectedPlan?.plan?.id) {
      showError(t('请选择订阅套餐'));
      return;
    }
    setPaying(true);
    try {
      const res = await API.post('/api/subscription/balance/pay', {
        plan_id: selectedPlan.plan.id,
      });
      if (isApiSuccess(res.data)) {
        showSuccess(t('订阅购买成功'));
        closeBuy();
        await Promise.allSettled([
          reloadSubscriptionSelf?.(),
          reloadUserQuota?.(),
        ]);
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('支付失败');
        showError(errorMsg);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaying(false);
    }
  };

  const isMobilePaymentScene = () => {
    const userAgent = navigator.userAgent || '';
    return (
      /Mobi|Android|iPhone|iPad|iPod|Windows Phone/i.test(userAgent) ||
      window.innerWidth < 768
    );
  };

  const submitOfficialAlipayForm = (formHtml, payWindow) => {
    if (!formHtml) {
      showError(t('支付请求失败'));
      payWindow?.close?.();
      return false;
    }
    const targetDocument = payWindow?.document || window.document;
    targetDocument.open();
    targetDocument.write(formHtml);
    targetDocument.close();
    return true;
  };

  const openWechatOfficialPayment = (data) => {
    if (data?.payment_type === 'redirect' && data?.payment_url) {
      window.location.href = data.payment_url;
      return true;
    }
    if (data?.payment_type === 'qrcode' && data?.code_url) {
      setWechatQrCodeUrl(data.code_url);
      setWechatQrOrderId(data.order_id || '');
      setWechatQrFallback(data.fallback || '');
      setWechatQrCreatedAt(Date.now());
      setWechatQrOrderTimeoutSeconds(
        normalizeOfficialPaymentOrderTimeoutSeconds(
          data.order_timeout_seconds ||
            getPaymentOrderTimeoutSeconds?.('wxpay_official'),
        ),
      );
      setWechatQrOpen(true);
      return true;
    }
    return false;
  };

  const payAlipayOfficial = async () => {
    if (!hasAlipayOfficial) {
      showError(t('管理员未开启支付宝官方支付充值！'));
      return;
    }
    setPaying(true);
    const scene = isMobilePaymentScene() ? 'h5' : 'pc';
    const alipayWindow = scene === 'pc' ? window.open('', '_blank') : null;
    try {
      const res = await API.post('/api/subscription/alipay-official/pay', {
        plan_id: selectedPlan.plan.id,
        scene,
      });
      if (isApiSuccess(res.data)) {
        if (
          submitOfficialAlipayForm(res.data.data?.form_html || '', alipayWindow)
        ) {
          showSuccess(t('已发起支付'));
          closeBuy();
        }
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('支付失败');
        showError(errorMsg);
        alipayWindow?.close?.();
      }
    } catch (e) {
      showError(t('支付请求失败'));
      alipayWindow?.close?.();
    } finally {
      setPaying(false);
    }
  };

  const payWechatPayOfficial = async () => {
    if (!hasWechatPayOfficial) {
      showError(t('管理员未开启微信支付官方充值！'));
      return;
    }
    const mobileScene = isMobilePaymentScene();
    if (shouldBlockOfficialWechatMobilePayment('wxpay_official', mobileScene)) {
      showError(
        t('当前移动端不支持使用微信支付，请使用电脑端或选择其他支付方式'),
      );
      return;
    }
    setPaying(true);
    const scene = mobileScene ? 'h5' : 'pc';
    try {
      const res = await API.post('/api/subscription/wechat-pay-official/pay', {
        plan_id: selectedPlan.plan.id,
        scene,
      });
      if (isApiSuccess(res.data)) {
        if (openWechatOfficialPayment(res.data.data)) {
          showSuccess(t('已发起支付'));
          closeBuy();
        } else {
          showError(t('支付请求失败'));
        }
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('支付失败');
        showError(errorMsg);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaying(false);
    }
  };

  // 当前订阅信息 - 支持多个订阅
  const hasActiveSubscription = activeSubscriptions.length > 0;
  const hasAnySubscription = allSubscriptions.length > 0;
  const disableSubscriptionPreference = !hasActiveSubscription;
  const subscriptionStatusCounts = useMemo(() => {
    const now = Date.now() / 1000;
    return (allSubscriptions || []).reduce(
      (counts, sub) => {
        const subscription = sub?.subscription;
        if (!subscription) {
          return counts;
        }
        const status = subscription.status;
        const startTime = Number(subscription.start_time || 0);
        const endTime = Number(subscription.end_time || 0);
        if (status === 'active' && startTime > now && endTime > now) {
          counts.pendingStart += 1;
        } else if (status !== 'active' || endTime <= now) {
          counts.inactive += 1;
        }
        return counts;
      },
      { pendingStart: 0, inactive: 0 },
    );
  }, [allSubscriptions]);
  const isSubscriptionPreference =
    billingPreference === 'subscription_first' ||
    billingPreference === 'subscription_only';
  const displayBillingPreference =
    disableSubscriptionPreference && isSubscriptionPreference
      ? 'wallet_first'
      : billingPreference;
  const subscriptionPreferenceLabel =
    billingPreference === 'subscription_only' ? t('仅用订阅') : t('优先订阅');

  useEffect(() => {
    if (!wechatQrOpen || !wechatQrOrderId) {
      if (wechatQrPollTimerRef.current) {
        clearInterval(wechatQrPollTimerRef.current);
        wechatQrPollTimerRef.current = null;
      }
      return;
    }

    let stopped = false;
    const poll = async () => {
      const done = await checkWechatQrOrderStatus(wechatQrOrderId);
      if (done && wechatQrPollTimerRef.current) {
        clearInterval(wechatQrPollTimerRef.current);
        wechatQrPollTimerRef.current = null;
        stopped = true;
      }
    };

    poll();
    wechatQrPollTimerRef.current = setInterval(() => {
      if (!stopped) {
        poll();
      }
    }, 3000);

    return () => {
      stopped = true;
      if (wechatQrPollTimerRef.current) {
        clearInterval(wechatQrPollTimerRef.current);
        wechatQrPollTimerRef.current = null;
      }
    };
  }, [wechatQrOpen, wechatQrOrderId]);

  const planPurchaseCountMap = useMemo(() => {
    const map = new Map();
    (allSubscriptions || []).forEach((sub) => {
      const planId = sub?.subscription?.plan_id;
      if (!planId) return;
      map.set(planId, (map.get(planId) || 0) + 1);
    });
    return map;
  }, [allSubscriptions]);

  const planTitleMap = useMemo(() => {
    const map = new Map();
    (plans || []).forEach((p) => {
      const plan = p?.plan;
      if (!plan?.id) return;
      map.set(plan.id, plan.title || '');
    });
    return map;
  }, [plans]);

  const getPlanPurchaseCount = (planId) =>
    planPurchaseCountMap.get(planId) || 0;

  const selectedPlanPurchaseInfo = selectedPlan?.plan?.id
    ? {
        limit: Number(selectedPlan?.plan?.max_purchase_per_user || 0),
        count: getPlanPurchaseCount(selectedPlan?.plan?.id),
      }
    : null;
  const selectedPlanPurchaseLimit = Number(
    selectedPlanPurchaseInfo?.limit || 0,
  );
  const selectedPlanPurchaseCount = Number(
    selectedPlanPurchaseInfo?.count || 0,
  );
  const selectedPlanPurchaseLimitReached =
    selectedPlanPurchaseLimit > 0 &&
    selectedPlanPurchaseCount >= selectedPlanPurchaseLimit;

  const openBuy = () => {
    if (!selectedPlan?.plan) {
      showError(t('请选择订阅套餐'));
      return;
    }
    if (selectedPlanPurchaseLimitReached) {
      showError(t('已达到购买上限'));
      return;
    }
    if (!selectedPaymentMethod) {
      showError(t('请选择支付方式'));
      return;
    }
    if (
      shouldBlockOfficialWechatMobilePayment(
        selectedPaymentMethod.provider,
        isMobilePaymentScene(),
      )
    ) {
      showError(
        t('当前移动端不支持使用微信支付，请使用电脑端或选择其他支付方式'),
      );
      return;
    }
    if (selectedPaymentMethod.provider === 'self_serve') {
      if (!selectedPaymentMethod.qrCode) {
        showError(t('管理员未配置收款码'));
        return;
      }
      if (selectedSelfServeExpectedMoney === null) {
        showError(t('请先配置自助充值价格'));
        return;
      }
      setSelfServeTransactionNo('');
      setSelfServeConfirmed(false);
      setSelfServeOpen(true);
      return;
    }
    setOpen(true);
  };

  const submitSelfServeSubscription = async () => {
    if (!selectedPlan?.plan?.id || !selectedPaymentMethod?.type) {
      showError(t('请选择订阅套餐'));
      return;
    }
    if (selectedSelfServeExpectedMoney === null) {
      showError(t('请先配置自助充值价格'));
      return;
    }
    if (!selfServeTransactionNo.trim()) {
      showError(t('请输入交易订单号'));
      return;
    }
    if (!selfServeConfirmed) {
      showError(t('请确认已完成付款并承诺信息真实'));
      return;
    }
    setSelfServeSubmitting(true);
    try {
      const res = await API.post('/api/subscription/self-serve/pay', {
        plan_id: selectedPlan.plan.id,
        payment_method: selectedPaymentMethod.type,
        declared_money: selectedSelfServeExpectedMoney,
        transaction_no: selfServeTransactionNo.trim(),
      });
      if (isApiSuccess(res.data)) {
        showSuccess(t('自助订阅已提交，订阅已立即开通'));
        closeSelfServeModal();
        await Promise.allSettled([
          reloadSubscriptionSelf?.(),
          reloadUserQuota?.(),
        ]);
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('提交失败');
        showError(errorMsg);
      }
    } catch (e) {
      showError(t('提交失败'));
    } finally {
      setSelfServeSubmitting(false);
    }
  };

  const confirmSubscriptionPurchase = async () => {
    if (!selectedPaymentMethod) {
      showError(t('请选择支付方式'));
      return;
    }
    if (selectedPaymentMethod.provider === 'balance') {
      await payBalance();
      return;
    }
    if (selectedPaymentMethod.provider === 'stripe') {
      await payStripe();
      return;
    }
    if (selectedPaymentMethod.provider === 'creem') {
      await payCreem();
      return;
    }
    if (selectedPaymentMethod.provider === 'alipay_official') {
      await payAlipayOfficial();
      return;
    }
    if (selectedPaymentMethod.provider === 'wxpay_official') {
      await payWechatPayOfficial();
      return;
    }
    if (selectedPaymentMethod.provider === 'epay') {
      await payEpay();
      return;
    }
    if (selectedPaymentMethod.provider === 'self_serve') {
      await submitSelfServeSubscription();
      return;
    }
    showError(t('支付方式不存在'));
  };

  // 计算单个订阅的剩余天数
  const getRemainingDays = (sub) => {
    if (!sub?.subscription?.end_time) return 0;
    const now = Date.now() / 1000;
    const remaining = sub.subscription.end_time - now;
    return Math.max(0, Math.ceil(remaining / 86400));
  };

  // 计算单个订阅的使用进度
  const getUsagePercent = (sub) => {
    const total = Number(sub?.subscription?.amount_total || 0);
    const used = Number(sub?.subscription?.amount_used || 0);
    if (total <= 0) return 0;
    return Math.min(100, Math.round((used / total) * 100));
  };

  const cardContent = (
    <>
      {/* 卡片头部 */}
      {loading ? (
        <div className='space-y-4'>
          {/* 我的订阅骨架屏 */}
          <Card className='!rounded-xl w-full' bodyStyle={{ padding: '12px' }}>
            <div className='flex items-center justify-between mb-3'>
              <Skeleton.Title active style={{ width: 100, height: 20 }} />
              <Skeleton.Button active style={{ width: 24, height: 24 }} />
            </div>
            <div className='space-y-2'>
              <Skeleton.Paragraph active rows={2} />
            </div>
          </Card>
          {/* 套餐列表骨架屏 */}
          <div className='grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-5 w-full px-1'>
            {[1, 2, 3].map((i) => (
              <Card
                key={i}
                className='!rounded-xl w-full h-full'
                bodyStyle={{ padding: 16 }}
              >
                <Skeleton.Title
                  active
                  style={{ width: '60%', height: 24, marginBottom: 8 }}
                />
                <Skeleton.Paragraph
                  active
                  rows={1}
                  style={{ marginBottom: 12 }}
                />
                <div className='text-center py-4'>
                  <Skeleton.Title
                    active
                    style={{ width: '40%', height: 32, margin: '0 auto' }}
                  />
                </div>
                <Skeleton.Paragraph active rows={3} style={{ marginTop: 12 }} />
                <Skeleton.Button
                  active
                  block
                  style={{ marginTop: 16, height: 32 }}
                />
              </Card>
            ))}
          </div>
        </div>
      ) : (
        <Space vertical style={{ width: '100%' }} spacing={8}>
          {/* 当前订阅状态 */}
          <Card className='!rounded-xl w-full' bodyStyle={{ padding: '12px' }}>
            <div className='flex items-center justify-between mb-2 gap-3'>
              <div className='flex items-center gap-2 flex-1 min-w-0'>
                <Text strong>{t('我的订阅')}</Text>
                {hasActiveSubscription ? (
                  <Tag
                    color='white'
                    size='small'
                    shape='circle'
                    prefixIcon={<Badge dot type='success' />}
                  >
                    {activeSubscriptions.length} {t('个生效中')}
                  </Tag>
                ) : (
                  <Tag color='white' size='small' shape='circle'>
                    {t('无生效')}
                  </Tag>
                )}
                {allSubscriptions.length > activeSubscriptions.length && (
                  <>
                    {subscriptionStatusCounts.pendingStart > 0 && (
                      <Tag color='white' size='small' shape='circle'>
                        {subscriptionStatusCounts.pendingStart} {t('未开始')}
                      </Tag>
                    )}
                    {subscriptionStatusCounts.inactive > 0 && (
                      <Tag color='white' size='small' shape='circle'>
                        {subscriptionStatusCounts.inactive} {t('个已过期')}
                      </Tag>
                    )}
                  </>
                )}
              </div>
              <div className='flex items-center gap-2'>
                <Select
                  value={displayBillingPreference}
                  onChange={onChangeBillingPreference}
                  size='small'
                  optionList={[
                    {
                      value: 'subscription_first',
                      label: disableSubscriptionPreference
                        ? `${t('优先订阅')} (${t('无生效')})`
                        : t('优先订阅'),
                      disabled: disableSubscriptionPreference,
                    },
                    { value: 'wallet_first', label: t('优先钱包') },
                    {
                      value: 'subscription_only',
                      label: disableSubscriptionPreference
                        ? `${t('仅用订阅')} (${t('无生效')})`
                        : t('仅用订阅'),
                      disabled: disableSubscriptionPreference,
                    },
                    { value: 'wallet_only', label: t('仅用钱包') },
                  ]}
                />
                <Button
                  size='small'
                  theme='light'
                  type='tertiary'
                  icon={
                    <RefreshCw
                      size={12}
                      className={refreshing ? 'animate-spin' : ''}
                    />
                  }
                  onClick={handleRefresh}
                  loading={refreshing}
                />
              </div>
            </div>
            {disableSubscriptionPreference && isSubscriptionPreference && (
              <Text type='tertiary' size='small'>
                {t('已保存偏好为')}
                {subscriptionPreferenceLabel}
                {t('，当前无生效订阅，将自动使用钱包')}
              </Text>
            )}

            {hasAnySubscription ? (
              <>
                <Divider margin={8} />
                <div className='max-h-64 overflow-y-auto pr-1 semi-table-body'>
                  {allSubscriptions.map((sub, subIndex) => {
                    const isLast = subIndex === allSubscriptions.length - 1;
                    const subscription = sub.subscription;
                    const totalAmount = Number(subscription?.amount_total || 0);
                    const usedAmount = Number(subscription?.amount_used || 0);
                    const remainAmount =
                      totalAmount > 0
                        ? Math.max(0, totalAmount - usedAmount)
                        : 0;
                    const planTitle =
                      planTitleMap.get(subscription?.plan_id) || '';
                    const remainDays = getRemainingDays(sub);
                    const usagePercent = getUsagePercent(sub);
                    const now = Date.now() / 1000;
                    const hasStarted = (subscription?.start_time || 0) <= now;
                    const isExpired = (subscription?.end_time || 0) <= now;
                    const isCancelled = subscription?.status === 'cancelled';
                    const isActive =
                      subscription?.status === 'active' &&
                      hasStarted &&
                      !isExpired;
                    const isPendingStart =
                      subscription?.status === 'active' &&
                      !hasStarted &&
                      !isExpired;

                    return (
                      <div key={subscription?.id || subIndex}>
                        {/* 订阅概要 */}
                        <div className='flex items-center justify-between text-xs mb-2'>
                          <div className='flex items-center gap-2'>
                            <span className='font-medium'>
                              {planTitle
                                ? `${planTitle} · ${t('订阅')} #${subscription?.id}`
                                : `${t('订阅')} #${subscription?.id}`}
                            </span>
                            {isActive ? (
                              <Tag
                                color='white'
                                size='small'
                                shape='circle'
                                prefixIcon={<Badge dot type='success' />}
                              >
                                {t('生效')}
                              </Tag>
                            ) : isPendingStart ? (
                              <Tag color='white' size='small' shape='circle'>
                                {t('未开始')}
                              </Tag>
                            ) : isCancelled ? (
                              <Tag color='white' size='small' shape='circle'>
                                {t('已作废')}
                              </Tag>
                            ) : (
                              <Tag color='white' size='small' shape='circle'>
                                {t('已过期')}
                              </Tag>
                            )}
                          </div>
                          {isActive && (
                            <span className='text-gray-500'>
                              {t('剩余')} {remainDays} {t('天')}
                            </span>
                          )}
                        </div>
                        <div className='text-xs text-gray-500 mb-2'>
                          {isActive
                            ? t('至')
                            : isPendingStart
                              ? t('未开始')
                              : isCancelled
                                ? t('作废于')
                                : t('过期于')}{' '}
                          {isPendingStart
                            ? `${new Date(
                                (subscription?.start_time || 0) * 1000,
                              ).toLocaleString()} - `
                            : ''}
                          {new Date(
                            (subscription?.end_time || 0) * 1000,
                          ).toLocaleString()}
                        </div>
                        {isActive && subscription?.next_reset_time > 0 && (
                          <div className='text-xs text-gray-500 mb-2'>
                            {t('下一次重置')}:{' '}
                            {new Date(
                              subscription.next_reset_time * 1000,
                            ).toLocaleString()}
                          </div>
                        )}
                        <div className='text-xs text-gray-500 mb-2'>
                          {t('总额度')}:{' '}
                          {totalAmount > 0 ? (
                            <Tooltip
                              content={`${t('原生额度')}：${usedAmount}/${totalAmount} · ${t('剩余')} ${remainAmount}`}
                            >
                              <span>
                                {renderQuota(usedAmount)}/
                                {renderQuota(totalAmount)} · {t('剩余')}{' '}
                                {renderQuota(remainAmount)}
                              </span>
                            </Tooltip>
                          ) : (
                            t('不限')
                          )}
                          {totalAmount > 0 && (
                            <span className='ml-2'>
                              {t('已用')} {usagePercent}%
                            </span>
                          )}
                        </div>
                        {!isLast && <Divider margin={12} />}
                      </div>
                    );
                  })}
                </div>
              </>
            ) : (
              <div className='text-xs text-gray-500'>
                {t('购买套餐后即可享受模型权益')}
              </div>
            )}
          </Card>

          {/* 可购买套餐 - 标准定价卡片 */}
          {plans.length > 0 ? (
            <div className='grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-5 w-full px-1'>
              {plans.map((p) => {
                const plan = p?.plan;
                const totalAmount = Number(plan?.total_amount || 0);
                const { symbol, displayPrice } = getPlanPriceDisplay(plan);
                const isRecommended = shouldHighlightSubscriptionPlan(plan);
                const isSelected = selectedPlan?.plan?.id === plan?.id;
                const limit = Number(plan?.max_purchase_per_user || 0);
                const limitLabel = limit > 0 ? `${t('限购')} ${limit}` : null;
                const totalLabel =
                  totalAmount > 0
                    ? `${t('总额度')}: ${renderQuota(totalAmount)}`
                    : `${t('总额度')}: ${t('不限')}`;
                const upgradeLabel = plan?.upgrade_group
                  ? `${t('升级分组')}: ${plan.upgrade_group}`
                  : null;
                const resetLabel =
                  formatSubscriptionResetPeriod(plan, t) === t('不重置')
                    ? null
                    : `${t('额度重置')}: ${formatSubscriptionResetPeriod(plan, t)}`;
                const modelLimits = getSubscriptionModelLimits(plan);
                const modelLimitsLabel =
                  modelLimits.length > 0
                    ? `${t('可用模型')}: ${modelLimits.length} ${t('个模型')}`
                    : null;
                const planBenefits = [
                  {
                    label: `${t('有效期')}: ${formatSubscriptionDuration(plan, t)}`,
                  },
                  resetLabel ? { label: resetLabel } : null,
                  totalAmount > 0
                    ? {
                        label: totalLabel,
                        tooltip: `${t('原生额度')}：${totalAmount}`,
                      }
                    : { label: totalLabel },
                  limitLabel ? { label: limitLabel } : null,
                  upgradeLabel ? { label: upgradeLabel } : null,
                  modelLimitsLabel
                    ? {
                        label: modelLimitsLabel,
                        tooltip: modelLimits.join(', '),
                      }
                    : null,
                ].filter(Boolean);

                return (
                  <Card
                    key={plan?.id}
                    className={`!rounded-xl transition-colors w-full h-full ${
                      isRecommended ? 'ring-2 ring-purple-500' : ''
                    } ${isSelected ? 'border-primary' : ''}`}
                    style={{
                      border: isSelected
                        ? '2px solid var(--semi-color-primary)'
                        : undefined,
                    }}
                    bodyStyle={{ padding: 0 }}
                  >
                    <div className='p-4 h-full flex flex-col'>
                      {/* 推荐标签 */}
                      {isRecommended && (
                        <div className='mb-2'>
                          <Tag color='purple' shape='circle' size='small'>
                            <Sparkles size={10} className='mr-1' />
                            {t('推荐')}
                          </Tag>
                        </div>
                      )}
                      {/* 套餐名称 */}
                      <div className='mb-3'>
                        <Typography.Title
                          heading={5}
                          ellipsis={{ rows: 1, showTooltip: true }}
                          style={{ margin: 0 }}
                        >
                          {plan?.title || t('订阅套餐')}
                        </Typography.Title>
                        {plan?.subtitle && (
                          <Text
                            type='tertiary'
                            size='small'
                            ellipsis={{ rows: 1, showTooltip: true }}
                            style={{ display: 'block' }}
                          >
                            {plan.subtitle}
                          </Text>
                        )}
                      </div>

                      {/* 价格区域 */}
                      <div className='py-2'>
                        <div className='flex items-baseline justify-start'>
                          <span className='text-xl font-bold text-purple-600'>
                            {symbol}
                          </span>
                          <span className='text-3xl font-bold text-purple-600'>
                            {displayPrice}
                          </span>
                        </div>
                      </div>

                      {/* 套餐权益描述 */}
                      <div className='flex flex-col items-start gap-1 pb-2'>
                        {planBenefits.map((item) => {
                          const content = (
                            <div className='flex items-center gap-2 text-xs text-gray-500'>
                              <Badge dot type='tertiary' />
                              <span>{item.label}</span>
                            </div>
                          );
                          if (!item.tooltip) {
                            return (
                              <div
                                key={item.label}
                                className='w-full flex justify-start'
                              >
                                {content}
                              </div>
                            );
                          }
                          return (
                            <Tooltip key={item.label} content={item.tooltip}>
                              <div className='w-full flex justify-start'>
                                {content}
                              </div>
                            </Tooltip>
                          );
                        })}
                      </div>

                      <div className='mt-auto'>
                        <Divider margin={12} />

                        {/* 购买按钮 */}
                        {(() => {
                          const count = getPlanPurchaseCount(p?.plan?.id);
                          const reached = limit > 0 && count >= limit;
                          const tip = reached
                            ? t('已达到购买上限') + ` (${count}/${limit})`
                            : '';
                          const buttonEl = (
                            <Button
                              theme={isSelected ? 'solid' : 'outline'}
                              type='primary'
                              block
                              disabled={reached}
                              onClick={() => {
                                if (!reached) selectPlan(p);
                              }}
                            >
                              {reached
                                ? t('已达上限')
                                : isSelected
                                  ? t('已选择')
                                  : t('选择订阅套餐')}
                            </Button>
                          );
                          return reached ? (
                            <Tooltip content={tip} position='top'>
                              {buttonEl}
                            </Tooltip>
                          ) : (
                            buttonEl
                          );
                        })()}
                      </div>
                    </div>
                  </Card>
                );
              })}
            </div>
          ) : (
            <div className='text-center text-gray-400 text-sm py-4'>
              {t('暂无可购买套餐')}
            </div>
          )}

          {plans.length > 0 && selectedPlan?.plan && (
            <Card className='!rounded-xl w-full' bodyStyle={{ padding: 16 }}>
              <div className='space-y-4'>
                <div className='flex flex-col gap-1'>
                  <Text strong>{t('选择支付方式')}</Text>
                  <Text type='tertiary' size='small'>
                    {selectedPlan.plan.title || t('订阅套餐')}
                  </Text>
                </div>

                {subscriptionPaymentMethods.length > 0 ? (
                  <>
                    <div className='flex flex-wrap gap-2'>
                      {subscriptionPaymentMethods.map((method) => (
                        <Button
                          key={method.key}
                          theme={
                            selectedPaymentKey === method.key
                              ? 'solid'
                              : 'outline'
                          }
                          type='primary'
                          icon={renderPaymentIcon(method)}
                          onClick={() => setSelectedPaymentKey(method.key)}
                          className='!rounded-lg !px-4 !py-2'
                        >
                          {method.name}
                        </Button>
                      ))}
                    </div>

                    <Divider margin={8} />
                    <div className='flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3'>
                      <div>
                        <Text type='tertiary' size='small'>
                          {t('应付金额')}
                        </Text>
                        <div className='text-2xl font-bold text-purple-600'>
                          {selectedPayAmountDisplay}
                        </div>
                        {selectedPaymentMethod?.provider === 'balance' && (
                          <Text type='tertiary' size='small'>
                            {t('可用余额')}：{renderQuota(availableBalance)}
                          </Text>
                        )}
                      </div>
                      <Button
                        theme='solid'
                        type='primary'
                        loading={paying}
                        disabled={
                          !selectedPaymentMethod ||
                          selectedPlanPurchaseLimitReached
                        }
                        onClick={openBuy}
                      >
                        {t('立即订阅')}
                      </Button>
                    </div>
                    {selectedPlanPurchaseLimitReached && (
                      <Text type='warning' size='small'>
                        {t('已达到购买上限')} ({selectedPlanPurchaseCount}/
                        {selectedPlanPurchaseLimit})
                      </Text>
                    )}
                  </>
                ) : (
                  <Banner
                    type='info'
                    description={t(
                      '管理员未开启在线支付功能，请联系管理员配置。',
                    )}
                    className='!rounded-xl'
                    closeIcon={null}
                  />
                )}
              </div>
            </Card>
          )}
        </Space>
      )}
    </>
  );

  return (
    <>
      {withCard ? (
        <Card className='!rounded-2xl shadow-sm border-0'>{cardContent}</Card>
      ) : (
        <div className='space-y-3'>{cardContent}</div>
      )}

      {/* 购买确认弹窗 */}
      <SubscriptionPurchaseModal
        t={t}
        visible={open}
        onCancel={closeBuy}
        selectedPlan={selectedPlan}
        paying={paying}
        selectedPaymentMethod={selectedPaymentMethod}
        displayPayAmount={selectedPayAmountDisplay}
        purchaseLimitInfo={selectedPlanPurchaseInfo}
        balanceCost={balanceCost}
        availableBalance={availableBalance}
        onConfirm={confirmSubscriptionPurchase}
      />

      <WechatOfficialQrPaymentModal
        t={t}
        visible={wechatQrOpen}
        codeUrl={wechatQrCodeUrl}
        fallback={wechatQrFallback}
        createdAt={wechatQrCreatedAt}
        orderTimeoutSeconds={wechatQrOrderTimeoutSeconds}
        onCancel={closeWechatQrModal}
      />

      <SelfServeSubscriptionModal
        t={t}
        visible={selfServeOpen}
        selectedPlan={selectedPlan}
        paymentMethod={selectedPaymentMethod?.type}
        paymentName={selectedPaymentMethod?.name}
        qrCode={selectedPaymentMethod?.qrCode}
        expectedMoney={selectedSelfServeExpectedMoney || 0}
        transactionNo={selfServeTransactionNo}
        setTransactionNo={setSelfServeTransactionNo}
        confirmed={selfServeConfirmed}
        setConfirmed={setSelfServeConfirmed}
        submitLoading={selfServeSubmitting}
        onSubmit={submitSelfServeSubscription}
        onCancel={closeSelfServeModal}
      />
    </>
  );
};

export default SubscriptionPlansCard;
