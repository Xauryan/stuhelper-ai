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
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  Modal,
  Table,
  Badge,
  Typography,
  Toast,
  Empty,
  Button,
  Input,
  InputNumber,
  Tag,
  Space,
  Checkbox,
  Radio,
  RadioGroup,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Coins, ImageUp } from 'lucide-react';
import { IconEdit, IconSearch } from '@douyinfe/semi-icons';
import { API, timestamp2string, renderQuota } from '../../../helpers';
import { isAdmin } from '../../../helpers/utils';
import CardTable from '../../common/ui/CardTable';
import SelfServeQRCode from '../SelfServeQRCode';
import { decodeQRCodeImage } from '../qrCodeUtils';
import {
  canAdminCompleteTopup,
  formatCurrency,
  getRemainingAdminRefundQuota,
  getRemainingRefundMoney,
  getTopupPaymentMethodLabel,
  isAdminMoneyRefundable,
  isAdminManagedTopup,
  isAdminManagedTopupRefundable,
  isBalanceTopup,
  isRefundRequestable,
  isSelfServeTopup,
  isSubscriptionTopup,
} from './topupHistoryUtils.mjs';
const { Text } = Typography;

// 状态映射配置
const STATUS_CONFIG = {
  success: { type: 'success', key: '成功' },
  pending: { type: 'warning', key: '待支付' },
  failed: { type: 'danger', key: '失败' },
  expired: { type: 'danger', key: '已超时' },
  partial_refunded: { type: 'warning', key: '部分退款' },
  refunded: { type: 'secondary', key: '已退款' },
};

const SELF_SERVE_AUDIT_STATUS_CONFIG = {
  pending: { type: 'warning', key: '待审核' },
  approved: { type: 'success', key: '审核通过' },
  rejected: { type: 'danger', key: '已拒绝' },
};

const REFUNDED_STATUS_KEYS = ['partial_refunded', 'refunded'];

const isAdminDirectMoneyRefundable = (record) =>
  isAdminMoneyRefundable(record) && !record.refund_request_id;

const isUserRefundRequestable = (record) =>
  isRefundRequestable(record) && !record.refund_request_id;

const EMPTY_TOPUP_FILTERS = {};
const QR_MAX_BYTES = 300 * 1024;

const ACTIONABLE_PAYMENT_STATUSES = ['pending', 'expired'];

const isPendingOrExpired = (record) =>
  ACTIONABLE_PAYMENT_STATUSES.includes(record?.status);

const getAdminTopupFeePercent = (record) => {
  const money = Number(record?.money || 0);
  const fee = Number(record?.fee || 0);
  if (!Number.isFinite(money) || !Number.isFinite(fee) || money <= 0) {
    return 0;
  }
  return Math.round((fee / money) * 100000) / 1000;
};

const formatPercent = (value) => {
  const percent = Number(value || 0);
  if (!Number.isFinite(percent)) {
    return '0';
  }
  return percent.toFixed(3).replace(/\.?0+$/, '');
};

const EMPTY_SELF_SERVE_LIMITS = {
  single_max_money: 0,
  daily_max_money: 0,
};

const positiveMoney = (value) => {
  const money = Number(value);
  return Number.isFinite(money) && money > 0 ? money : 0;
};

const normalizeSelfServeLimits = (limits) => ({
  single_max_money: positiveMoney(limits?.single_max_money),
  daily_max_money: positiveMoney(limits?.daily_max_money),
});

const getSelfServeDisplayMoney = (record) => {
  const declaredMoney = Number(record?.declared_money);
  if (Number.isFinite(declaredMoney) && declaredMoney > 0) {
    return declaredMoney;
  }
  const paidMoney = Number(record?.money);
  return Number.isFinite(paidMoney) && paidMoney > 0 ? paidMoney : 0;
};

const TopupBillingTable = ({
  active = true,
  compactMode = false,
  externalFilters,
  externalKeyword,
  hideFilters = false,
  hidePagination = false,
  onPaginationChange,
  onReady,
  pendingRefundOnly = false,
  pendingSelfServeAuditOnly = false,
  t,
  variant = 'embedded',
}) => {
  const userIsAdmin = useMemo(() => isAdmin(), []);
  const [loading, setLoading] = useState(false);
  const [topups, setTopups] = useState([]);
  const [total, setTotal] = useState(0);
  const [totalMoney, setTotalMoney] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [keyword, setKeyword] = useState('');
  const [debouncedKeyword, setDebouncedKeyword] = useState('');
  const [refundVisible, setRefundVisible] = useState(false);
  const [refundRecord, setRefundRecord] = useState(null);
  const [refundAmount, setRefundAmount] = useState(0);
  const [refundQuota, setRefundQuota] = useState(0);
  const [refundReason, setRefundReason] = useState('');
  const [refundLoading, setRefundLoading] = useState(false);
  const [refundPreview, setRefundPreview] = useState(null);
  const [refundMode, setRefundMode] = useState('direct');
  const [refundFull, setRefundFull] = useState(false);
  const [refundQRCode, setRefundQRCode] = useState('');
  const refundQRCodeFileRef = useRef(null);
  const [editVisible, setEditVisible] = useState(false);
  const [editRecord, setEditRecord] = useState(null);
  const [editLoading, setEditLoading] = useState(false);
  const [editForm, setEditForm] = useState({
    operationType: 'recharge',
    amount: 0,
    money: 0,
    fee: 0,
    serviceFeePercent: 0,
    useDefaultMoney: true,
  });
  const [selfServeEditVisible, setSelfServeEditVisible] = useState(false);
  const [selfServeEditRecord, setSelfServeEditRecord] = useState(null);
  const [selfServeEditLoading, setSelfServeEditLoading] = useState(false);
  const [selfServeEditForm, setSelfServeEditForm] = useState({
    declaredMoney: 0,
    transactionNo: '',
    reason: '',
  });
  const [selfServeLimits, setSelfServeLimits] = useState(
    EMPTY_SELF_SERVE_LIMITS,
  );
  const selfServeSingleMax = positiveMoney(selfServeLimits.single_max_money);
  const isPageVariant = variant === 'page';
  const selfServeRefundRecord = isSelfServeTopup(refundRecord);
  const userSelfServeRefundApply =
    selfServeRefundRecord && !userIsAdmin && refundMode === 'direct';
  const adminSelfServeRefundApprove =
    selfServeRefundRecord && userIsAdmin && refundMode === 'approve';
  // hideFilters 模式（独立账单页）下，关键词由父组件 submit 后传入，不需要再做内部 debounce；
  // 否则在弹窗内是逐字输入，加 300ms debounce 避免每按一个字符就打一次 API。
  const effectiveKeyword = hideFilters
    ? externalKeyword || ''
    : debouncedKeyword;
  const effectiveFilters = useMemo(
    () =>
      hideFilters
        ? externalFilters || EMPTY_TOPUP_FILTERS
        : EMPTY_TOPUP_FILTERS,
    [externalFilters, hideFilters],
  );

  useEffect(() => {
    if (hideFilters) {
      return undefined;
    }
    const timer = setTimeout(() => {
      setDebouncedKeyword(keyword);
    }, 300);
    return () => clearTimeout(timer);
  }, [keyword, hideFilters]);

  const loadTopups = useCallback(
    async (currentPage, currentPageSize) => {
      setLoading(true);
      try {
        const base = isAdmin() ? '/api/user/topup' : '/api/user/topup/self';
        const params = new URLSearchParams({
          p: String(currentPage),
          page_size: String(currentPageSize),
        });
        if (effectiveKeyword) {
          params.set('keyword', effectiveKeyword);
        }
        if (effectiveFilters?.user_id) {
          params.set('user_id', effectiveFilters.user_id);
        }
        if (effectiveFilters?.username) {
          params.set('username', effectiveFilters.username);
        }
        if (effectiveFilters?.trade_no) {
          params.set('trade_no', effectiveFilters.trade_no);
        }
        if (effectiveFilters?.payment_method) {
          params.set('payment_method', effectiveFilters.payment_method);
        }
        if (effectiveFilters?.audit_status) {
          params.set('audit_status', effectiveFilters.audit_status);
        }
        if (
          Array.isArray(effectiveFilters?.dateRange) &&
          effectiveFilters.dateRange.length === 2
        ) {
          const startTimestamp = Date.parse(effectiveFilters.dateRange[0]);
          const endTimestamp = Date.parse(effectiveFilters.dateRange[1]);
          if (!Number.isNaN(startTimestamp)) {
            params.set('start_timestamp', String(startTimestamp / 1000));
          }
          if (!Number.isNaN(endTimestamp)) {
            params.set('end_timestamp', String(endTimestamp / 1000));
          }
        }
        if (pendingRefundOnly) {
          params.set('pending_refund', 'true');
        }
        if (pendingSelfServeAuditOnly) {
          params.set('payment_method', 'self_serve');
          params.set('audit_status', 'pending');
        }
        const endpoint = `${base}?${params.toString()}`;
        const res = await API.get(endpoint);
        const { success, message, data } = res.data;
        if (success) {
          setTopups(data.items || []);
          setTotal(data.total || 0);
          setTotalMoney(Number(data.total_money || 0));
          onPaginationChange?.({
            page: data.page || currentPage,
            pageSize: data.page_size || currentPageSize,
            total: data.total || 0,
            totalMoney: Number(data.total_money || 0),
          });
        } else {
          Toast.error({ content: message || t('加载失败') });
        }
      } catch (error) {
        Toast.error({ content: t('加载账单失败') });
      } finally {
        setLoading(false);
      }
    },
    [
      effectiveFilters,
      effectiveKeyword,
      onPaginationChange,
      pendingRefundOnly,
      pendingSelfServeAuditOnly,
      t,
    ],
  );

  useEffect(() => {
    if (active) {
      loadTopups(page, pageSize);
    }
  }, [active, page, pageSize, loadTopups]);

  useEffect(() => {
    onReady?.({ setPage, setPageSize });
  }, [onReady]);

  useEffect(() => {
    if (onPaginationChange) {
      onPaginationChange({ page, pageSize, total, totalMoney });
    }
  }, [onPaginationChange, page, pageSize, total, totalMoney]);

  useEffect(() => {
    if (!userIsAdmin) {
      return;
    }
    let cancelled = false;
    const loadSelfServePolicy = async () => {
      try {
        const res = await API.get('/api/user/topup/info');
        const { success, data } = res.data;
        if (!cancelled && success) {
          setSelfServeLimits(normalizeSelfServeLimits(data?.self_serve_limits));
        }
      } catch (e) {
        // Keep the local default when the policy cannot be loaded.
      }
    };
    loadSelfServePolicy();
    return () => {
      cancelled = true;
    };
  }, [userIsAdmin]);

  const handlePageChange = (currentPage) => {
    setPage(currentPage);
  };

  const handlePageSizeChange = (currentPageSize) => {
    setPageSize(currentPageSize);
    setPage(1);
  };

  const handleKeywordChange = (value) => {
    setKeyword(value);
    setPage(1);
  };

  // 管理员补单
  const handleAdminComplete = useCallback(
    async (tradeNo) => {
      try {
        const res = await API.post('/api/user/topup/complete', {
          trade_no: tradeNo,
        });
        const { success, message } = res.data;
        if (success) {
          Toast.success({ content: t('补单成功') });
          await loadTopups(page, pageSize);
        } else {
          Toast.error({ content: message || t('补单失败') });
        }
      } catch (e) {
        Toast.error({ content: t('补单失败') });
      }
    },
    [loadTopups, page, pageSize, t],
  );

  const confirmAdminComplete = useCallback(
    (tradeNo) => {
      Modal.confirm({
        title: t('确认补单'),
        content: t('是否将该订单标记为成功并为用户入账？'),
        onOk: () => handleAdminComplete(tradeNo),
      });
    },
    [handleAdminComplete, t],
  );

  const openEditModal = useCallback((record) => {
    if (!record) {
      return;
    }
    setEditRecord(record);
    setEditForm({
      operationType: 'recharge',
      amount: Math.round(Number(record.amount || 0)),
      money: Number(record.money || 0),
      fee: Number(record.fee || 0),
      serviceFeePercent: getAdminTopupFeePercent(record),
      useDefaultMoney: true,
    });
    setEditVisible(true);
  }, []);

  const setEditField = useCallback((field, value) => {
    setEditForm((prev) => ({
      ...prev,
      [field]: value,
    }));
  }, []);

  const handleEditAdminTopup = async () => {
    if (!editRecord) {
      return;
    }
    const amount = Math.round(Number(editForm.amount || 0));
    if (!amount || amount <= 0) {
      Toast.error({ content: t('请输入充值额度') });
      return;
    }
    const operationType = editForm.operationType || 'recharge';
    const payload = {
      trade_no: editRecord.trade_no,
      operation_type: operationType,
      amount,
      use_default_money:
        operationType === 'recharge' ? Boolean(editForm.useDefaultMoney) : true,
    };
    if (operationType === 'recharge') {
      const serviceFeePercent = Number(editForm.serviceFeePercent || 0);
      if (!Number.isFinite(serviceFeePercent) || serviceFeePercent < 0) {
        Toast.error({ content: t('支付手续费不能小于 0') });
        return;
      }
      payload.service_fee_percent = serviceFeePercent;
      if (!editForm.useDefaultMoney) {
        const money = Math.round(Number(editForm.money || 0) * 100) / 100;
        const fee = Math.round(Number(editForm.fee || 0) * 100) / 100;
        if (!Number.isFinite(money) || money < 0) {
          Toast.error({ content: t('支付金额不能小于 0') });
          return;
        }
        if (!Number.isFinite(fee) || fee < 0) {
          Toast.error({ content: t('支付手续费不能小于 0') });
          return;
        }
        payload.money = money;
        payload.fee = fee;
      }
    }

    setEditLoading(true);
    try {
      const res = await API.post('/api/user/topup/admin/update', payload);
      const { success, message } = res.data;
      if (success) {
        Toast.success({
          content:
            operationType === 'gift' ? t('已转为管理员赠送') : t('账单已更新'),
        });
        setEditVisible(false);
        await loadTopups(page, pageSize);
      } else {
        Toast.error({ content: message || t('操作失败') });
      }
    } catch (e) {
      Toast.error({ content: t('操作失败') });
    } finally {
      setEditLoading(false);
    }
  };

  const openRefundModal = useCallback(
    async (record) => {
      if (isAdminManagedTopup(record)) {
        setRefundPreview(null);
        setRefundRecord(record);
        setRefundAmount(0);
        setRefundQuota(getRemainingAdminRefundQuota(record));
        setRefundReason('');
        setRefundQRCode('');
        setRefundMode('admin_quota');
        setRefundFull(false);
        setRefundVisible(true);
        return;
      }
      let preview = null;
      let remaining = getRemainingRefundMoney(record);
      try {
        const res = await API.post('/api/user/topup/official/refund/preview', {
          trade_no: record.trade_no,
        });
        const { success, data } = res.data;
        if (success && data) {
          preview = data;
          remaining = Number(data.max_refund_amount || remaining);
        }
      } catch (e) {
        // The admin can still fall back to the local remaining amount.
      }
      setRefundPreview(preview);
      setRefundRecord(record);
      setRefundAmount(remaining);
      setRefundQuota(0);
      setRefundReason('');
      setRefundQRCode(record?.refund_request_qrcode || '');
      setRefundMode(
        record.refund_request_id && userIsAdmin ? 'approve' : 'direct',
      );
      setRefundFull(false);
      setRefundVisible(true);
    },
    [userIsAdmin],
  );

  const handleRefund = async () => {
    if (!refundRecord) {
      return;
    }
    if (refundMode === 'admin_quota') {
      const maxQuota = getRemainingAdminRefundQuota(refundRecord);
      const normalizedQuota = Math.round(Number(refundQuota || 0));
      if (!normalizedQuota || normalizedQuota <= 0) {
        Toast.error({ content: t('请输入退款额度') });
        return;
      }
      if (normalizedQuota > maxQuota) {
        Toast.error({ content: t('退款额度不能超过可退额度') });
        return;
      }
      setRefundLoading(true);
      try {
        const res = await API.post('/api/user/topup/admin/refund', {
          trade_no: refundRecord.trade_no,
          refund_quota: normalizedQuota,
          reason: refundReason,
          full_refund: refundFull,
        });
        const { success, message } = res.data;
        if (success) {
          Toast.success({ content: t('退款成功') });
          setRefundVisible(false);
          await loadTopups(page, pageSize);
        } else {
          Toast.error({ content: message || t('退款失败') });
        }
      } catch (e) {
        Toast.error({ content: t('退款失败') });
      } finally {
        setRefundLoading(false);
      }
      return;
    }
    const maxAmount = refundFull
      ? getRemainingRefundMoney(refundRecord)
      : (refundPreview?.max_refund_amount ??
        getRemainingRefundMoney(refundRecord));
    const normalizedAmount = Math.round(Number(refundAmount || 0) * 100) / 100;
    if (!normalizedAmount || normalizedAmount <= 0) {
      Toast.error({ content: t('请输入退款金额') });
      return;
    }
    if (normalizedAmount > maxAmount) {
      Toast.error({ content: t('退款金额不能超过可退金额') });
      return;
    }
    const normalizedRefundQRCode = refundQRCode.trim();
    if (userSelfServeRefundApply && !normalizedRefundQRCode) {
      Toast.error({ content: t('请上传或填写退款收款码') });
      return;
    }
    setRefundLoading(true);
    try {
      let res;
      if (refundMode === 'approve') {
        res = await API.post(
          '/api/user/topup/official/refund-request/approve',
          {
            request_id: refundRecord.refund_request_id,
            refund_amount: normalizedAmount,
            reason: refundReason,
            full_refund: refundFull,
          },
        );
      } else {
        const isWechatOfficial =
          refundRecord.payment_provider === 'wxpay_official' ||
          refundRecord.payment_method === 'wxpay_official';
        const isBalancePayment = isBalanceTopup(refundRecord);
        const isSelfServePayment = isSelfServeTopup(refundRecord);
        const endpoint = userIsAdmin
          ? isBalancePayment
            ? '/api/user/topup/balance/refund'
            : isSelfServePayment
              ? '/api/user/topup/self-serve/refund'
              : isWechatOfficial
                ? '/api/user/topup/wechat-pay-official/refund'
                : '/api/user/topup/alipay-official/refund'
          : '/api/user/topup/official/refund/apply';
        res = await API.post(endpoint, {
          trade_no: refundRecord.trade_no,
          refund_amount: normalizedAmount,
          reason: refundReason,
          full_refund: refundFull,
          refund_qrcode: userSelfServeRefundApply
            ? normalizedRefundQRCode
            : undefined,
        });
      }
      const { success, message } = res.data;
      if (success) {
        Toast.success({
          content:
            userIsAdmin || refundMode === 'approve'
              ? t('退款成功')
              : t('退款申请已提交'),
        });
        setRefundVisible(false);
        await loadTopups(page, pageSize);
      } else {
        Toast.error({ content: message || t('退款失败') });
      }
    } catch (e) {
      Toast.error({ content: t('退款失败') });
    } finally {
      setRefundLoading(false);
    }
  };

  const handleRefundQRCodeFile = async (event) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    if (file.size > QR_MAX_BYTES) {
      Toast.error({ content: t('二维码图片不能超过 300KB') });
      return;
    }
    try {
      const decoded = await decodeQRCodeImage(file);
      setRefundQRCode(decoded);
      Toast.success({ content: t('二维码已解码并填入') });
    } catch (error) {
      Toast.error({ content: t('未能识别二维码，请上传清晰的收款码图片') });
    }
  };

  const handleRejectRefundRequest = useCallback(
    (record) => {
      let rejectReason = '';
      Modal.confirm({
        title: t('拒绝退款申请'),
        content: (
          <Input
            placeholder={t('拒绝原因，可留空')}
            onChange={(value) => {
              rejectReason = value;
            }}
            showClear
          />
        ),
        onOk: async () => {
          try {
            const res = await API.post(
              '/api/user/topup/official/refund-request/reject',
              {
                request_id: record.refund_request_id,
                reason: rejectReason,
              },
            );
            const { success, message } = res.data;
            if (success) {
              Toast.success({ content: t('已拒绝退款申请') });
              await loadTopups(page, pageSize);
            } else {
              Toast.error({ content: message || t('操作失败') });
            }
          } catch (e) {
            Toast.error({ content: t('操作失败') });
          }
        },
      });
    },
    [loadTopups, page, pageSize, t],
  );

  const openSelfServeEditModal = useCallback((record) => {
    if (!record) {
      return;
    }
    setSelfServeEditRecord(record);
    setSelfServeEditForm({
      declaredMoney: Number(record.declared_money || record.money || 0),
      transactionNo: record.transaction_no || '',
      reason: record.audit_admin_reason || '',
    });
    setSelfServeEditVisible(true);
  }, []);

  const setSelfServeEditField = useCallback((field, value) => {
    setSelfServeEditForm((prev) => ({
      ...prev,
      [field]: value,
    }));
  }, []);

  const handleSelfServeApprove = useCallback(
    (record) => {
      let auditReason = '';
      Modal.confirm({
        title: t('通过自助充值审核'),
        content: (
          <div className='space-y-3'>
            <div>
              <Text type='secondary'>
                {t('确认该交易订单号已真实到账后再通过审核。')}
              </Text>
            </div>
            <Input
              placeholder={t('审核备注，可留空')}
              onChange={(value) => {
                auditReason = value;
              }}
              showClear
            />
          </div>
        ),
        okText: t('通过'),
        cancelText: t('取消'),
        onOk: async () => {
          try {
            const res = await API.post('/api/user/topup/self-serve/approve', {
              trade_no: record.trade_no,
              reason: auditReason,
            });
            const { success, message } = res.data;
            if (success) {
              Toast.success({ content: t('审核已通过') });
              await loadTopups(page, pageSize);
            } else {
              Toast.error({ content: message || t('操作失败') });
            }
          } catch (e) {
            Toast.error({ content: t('操作失败') });
          }
        },
      });
    },
    [loadTopups, page, pageSize, t],
  );

  const handleSelfServeReject = useCallback(
    (record) => {
      let rejectReason = '';
      let banUser = false;
      Modal.confirm({
        title: t('拒绝自助充值审核'),
        content: (
          <div className='space-y-3'>
            <Text type='warning'>
              {t('拒绝后会立即扣回该订单已到账余额，且不会退款。')}
            </Text>
            <Input
              placeholder={t('拒绝原因，可留空')}
              onChange={(value) => {
                rejectReason = value;
              }}
              showClear
            />
            <Checkbox
              onChange={(event) => {
                banUser = event.target.checked;
              }}
            >
              {t('同时封禁该用户')}
            </Checkbox>
          </div>
        ),
        okText: t('确认拒绝'),
        cancelText: t('取消'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            const res = await API.post('/api/user/topup/self-serve/reject', {
              trade_no: record.trade_no,
              reason: rejectReason,
              ban_user: banUser,
            });
            const { success, message } = res.data;
            if (success) {
              Toast.success({
                content: banUser ? t('已拒绝并封禁用户') : t('已拒绝审核'),
              });
              await loadTopups(page, pageSize);
            } else {
              Toast.error({ content: message || t('操作失败') });
            }
          } catch (e) {
            Toast.error({ content: t('操作失败') });
          }
        },
      });
    },
    [loadTopups, page, pageSize, t],
  );

  const handleSelfServeUpdate = async () => {
    if (!selfServeEditRecord) {
      return;
    }
    const declaredMoney =
      Math.round(Number(selfServeEditForm.declaredMoney || 0) * 100) / 100;
    const transactionNo = selfServeEditForm.transactionNo.trim();
    if (!declaredMoney || declaredMoney <= 0) {
      Toast.error({ content: t('请输入充值金额') });
      return;
    }
    if (!selfServeSingleMax) {
      Toast.error({ content: t('请先配置自助充值限额') });
      return;
    }
    if (declaredMoney > selfServeSingleMax) {
      Toast.error({
        content: t('单笔自助充值金额不能超过 {{amount}} 元', {
          amount: selfServeSingleMax.toFixed(2),
        }),
      });
      return;
    }
    if (!transactionNo) {
      Toast.error({ content: t('请输入交易订单号') });
      return;
    }
    setSelfServeEditLoading(true);
    try {
      const res = await API.post('/api/user/topup/self-serve/update', {
        trade_no: selfServeEditRecord.trade_no,
        declared_money: declaredMoney,
        transaction_no: transactionNo,
        reason: selfServeEditForm.reason,
      });
      const { success, message } = res.data;
      if (success) {
        Toast.success({ content: t('自助充值订单已更新') });
        setSelfServeEditVisible(false);
        await loadTopups(page, pageSize);
      } else {
        Toast.error({ content: message || t('操作失败') });
      }
    } catch (e) {
      Toast.error({ content: t('操作失败') });
    } finally {
      setSelfServeEditLoading(false);
    }
  };

  const handleQueryAlipayOfficial = useCallback(
    async (tradeNo) => {
      try {
        const res = await API.post('/api/user/topup/alipay-official/query', {
          trade_no: tradeNo,
        });
        const { success, message, data } = res.data;
        if (success) {
          Toast.success({
            content: data?.trade_status
              ? `${t('查询成功')}：${data.trade_status}`
              : t('查询成功'),
          });
          await loadTopups(page, pageSize);
        } else {
          Toast.error({ content: message || t('查询失败') });
        }
      } catch (e) {
        Toast.error({ content: t('查询失败') });
      }
    },
    [loadTopups, page, pageSize, t],
  );

  const handleQueryWechatPayOfficial = useCallback(
    async (tradeNo) => {
      try {
        const res = await API.post(
          '/api/user/topup/wechat-pay-official/query',
          {
            trade_no: tradeNo,
          },
        );
        const { success, message, data } = res.data;
        if (success) {
          Toast.success({
            content: data?.trade_state
              ? `${t('查询成功')}：${data.trade_state}`
              : t('查询成功'),
          });
          await loadTopups(page, pageSize);
        } else {
          Toast.error({ content: message || t('查询失败') });
        }
      } catch (e) {
        Toast.error({ content: t('查询失败') });
      }
    },
    [loadTopups, page, pageSize, t],
  );

  const confirmCloseAlipayOfficial = useCallback(
    (tradeNo) => {
      Modal.confirm({
        title: t('关闭订单'),
        content: t('确认关闭该支付宝官方待支付订单？'),
        onOk: async () => {
          try {
            const res = await API.post(
              '/api/user/topup/alipay-official/close',
              {
                trade_no: tradeNo,
              },
            );
            const { success, message } = res.data;
            if (success) {
              Toast.success({ content: t('订单已关闭') });
              await loadTopups(page, pageSize);
            } else {
              Toast.error({ content: message || t('关闭订单失败') });
            }
          } catch (e) {
            Toast.error({ content: t('关闭订单失败') });
          }
        },
      });
    },
    [loadTopups, page, pageSize, t],
  );

  const confirmCloseWechatPayOfficial = useCallback(
    (tradeNo) => {
      Modal.confirm({
        title: t('关闭订单'),
        content: t('确认关闭该微信待支付订单？'),
        onOk: async () => {
          try {
            const res = await API.post(
              '/api/user/topup/wechat-pay-official/close',
              {
                trade_no: tradeNo,
              },
            );
            const { success, message } = res.data;
            if (success) {
              Toast.success({ content: t('订单已关闭') });
              await loadTopups(page, pageSize);
            } else {
              Toast.error({ content: message || t('关闭订单失败') });
            }
          } catch (e) {
            Toast.error({ content: t('关闭订单失败') });
          }
        },
      });
    },
    [loadTopups, page, pageSize, t],
  );

  // 渲染状态徽章
  const renderStatusBadge = (status, record) => {
    let config = STATUS_CONFIG[status] || { type: 'primary', key: status };

    if (isSelfServeTopup(record) && record.audit_status) {
      if (record.audit_status === 'rejected') {
        config = SELF_SERVE_AUDIT_STATUS_CONFIG.rejected;
      } else if (!REFUNDED_STATUS_KEYS.includes(status)) {
        config = SELF_SERVE_AUDIT_STATUS_CONFIG[record.audit_status] || {
          type: 'secondary',
          key: record.audit_status,
        };
      }
    }

    return (
      <span className='flex items-center gap-2'>
        <Badge dot type={config.type} />
        <span>{t(config.key)}</span>
      </span>
    );
  };

  // 渲染支付方式
  const renderPaymentMethod = (pm) => {
    const displayName = getTopupPaymentMethodLabel(pm);
    return <Text>{displayName === '-' ? displayName : t(displayName)}</Text>;
  };

  const renderQRCodePreview = (value, alt) => {
    const text = String(value || '').trim();
    if (!text) {
      return null;
    }
    return (
      <div className='inline-flex rounded-lg border border-[var(--semi-color-border)] bg-white p-2'>
        <SelfServeQRCode value={text} alt={alt} size={128} />
      </div>
    );
  };

  const columns = useMemo(() => {
    const baseColumns = [
      ...(userIsAdmin
        ? [
            {
              title: t('用户ID'),
              dataIndex: 'user_id',
              key: 'user_id',
              render: (userId) => <Text>{userId ?? '-'}</Text>,
            },
            {
              title: t('用户名'),
              dataIndex: 'username',
              key: 'username',
              render: (username) => <Text>{username || '-'}</Text>,
            },
          ]
        : []),
      {
        title: t('订单号'),
        dataIndex: 'trade_no',
        key: 'trade_no',
        render: (text) => <Text copyable>{text}</Text>,
      },
      {
        title: t('支付方式'),
        dataIndex: 'payment_method',
        key: 'payment_method',
        render: renderPaymentMethod,
      },
      {
        title: t('交易订单号'),
        dataIndex: 'transaction_no',
        key: 'transaction_no',
        render: (transactionNo, record) =>
          isSelfServeTopup(record) && transactionNo ? (
            <Text copyable>{transactionNo}</Text>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
      {
        title: t('充值额度'),
        dataIndex: 'amount',
        key: 'amount',
        render: (amount, record) => {
          if (isSubscriptionTopup(record)) {
            return (
              <Tag color='purple' shape='circle' size='small'>
                {t('订阅套餐')}
              </Tag>
            );
          }
          if (isSelfServeTopup(record)) {
            return (
              <Text type='danger'>
                ¥{formatCurrency(getSelfServeDisplayMoney(record))}
              </Text>
            );
          }
          if (isAdminManagedTopup(record)) {
            return (
              <span className='flex items-center gap-1'>
                <Coins size={16} />
                <Text>{renderQuota(amount)}</Text>
              </span>
            );
          }
          return (
            <span className='flex items-center gap-1'>
              <Coins size={16} />
              <Text>{amount}</Text>
            </span>
          );
        },
      },
      {
        title: t('支付金额'),
        dataIndex: 'money',
        key: 'money',
        render: (money) => <Text type='danger'>¥{formatCurrency(money)}</Text>,
      },
      {
        title: t('手续费'),
        dataIndex: 'fee',
        key: 'fee',
        render: (fee) =>
          Number(fee || 0) > 0 ? (
            <Text type='secondary'>¥{formatCurrency(fee)}</Text>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
      {
        title: t('已退款'),
        dataIndex: 'refunded_money',
        key: 'refunded_money',
        render: (money, record) =>
          isAdminManagedTopup(record) &&
          Number(record?.refunded_quota || 0) > 0 ? (
            <Text type='warning'>{renderQuota(record.refunded_quota)}</Text>
          ) : Number(money || 0) > 0 ? (
            <Text type='warning'>¥{formatCurrency(money)}</Text>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        key: 'status',
        render: renderStatusBadge,
      },
    ];

    baseColumns.push({
      title: userIsAdmin ? t('操作') : t('退款'),
      key: 'action',
      render: (_, record) => {
        const actions = [];
        const subscriptionTopup = isSubscriptionTopup(record);
        const isActionablePayment = isPendingOrExpired(record);
        if (userIsAdmin && canAdminCompleteTopup(record)) {
          actions.push(
            <Button
              key='complete'
              size='small'
              type='primary'
              theme='outline'
              onClick={() => confirmAdminComplete(record.trade_no)}
            >
              {t('补单')}
            </Button>,
          );
        }
        if (
          userIsAdmin &&
          isActionablePayment &&
          record.payment_provider === 'alipay_official'
        ) {
          actions.push(
            <Button
              key='query'
              size='small'
              theme='outline'
              onClick={() => handleQueryAlipayOfficial(record.trade_no)}
            >
              {t('查询')}
            </Button>,
          );
          if (record.status === 'pending' && !subscriptionTopup) {
            actions.push(
              <Button
                key='close'
                size='small'
                type='danger'
                theme='outline'
                onClick={() => confirmCloseAlipayOfficial(record.trade_no)}
              >
                {t('关闭')}
              </Button>,
            );
          }
        }
        if (
          userIsAdmin &&
          isActionablePayment &&
          record.payment_provider === 'wxpay_official'
        ) {
          actions.push(
            <Button
              key='wx-query'
              size='small'
              theme='outline'
              onClick={() => handleQueryWechatPayOfficial(record.trade_no)}
            >
              {t('查询')}
            </Button>,
          );
          if (record.status === 'pending' && !subscriptionTopup) {
            actions.push(
              <Button
                key='wx-close'
                size='small'
                type='danger'
                theme='outline'
                onClick={() => confirmCloseWechatPayOfficial(record.trade_no)}
              >
                {t('关闭')}
              </Button>,
            );
          }
        }
        if (
          userIsAdmin &&
          isAdminDirectMoneyRefundable(record) &&
          record.refund_request_status !== 'pending'
        ) {
          actions.push(
            <Button
              key='refund'
              size='small'
              type='warning'
              theme='outline'
              onClick={() => openRefundModal(record)}
            >
              {t('退款')}
            </Button>,
          );
        }
        if (
          !userIsAdmin &&
          isUserRefundRequestable(record) &&
          record.refund_request_status !== 'pending'
        ) {
          actions.push(
            <Button
              key='refund-apply'
              size='small'
              type='warning'
              theme='outline'
              onClick={() => openRefundModal(record)}
            >
              {t('申请退款')}
            </Button>,
          );
        }
        if (!userIsAdmin && record.refund_request_status === 'pending') {
          actions.push(
            <span key='refund-pending' className='flex items-center gap-2'>
              <Badge dot type='warning' />
              <span>{t('退款审核中')}</span>
            </span>,
          );
        }
        if (userIsAdmin && isAdminManagedTopup(record)) {
          actions.push(
            <Button
              key='admin-edit'
              size='small'
              theme='outline'
              icon={<IconEdit />}
              onClick={() => openEditModal(record)}
            >
              {t('编辑')}
            </Button>,
          );
        }
        if (userIsAdmin && isAdminManagedTopupRefundable(record)) {
          actions.push(
            <Button
              key='admin-refund'
              size='small'
              type='warning'
              theme='outline'
              onClick={() => openRefundModal(record)}
            >
              {t('退款')}
            </Button>,
          );
        }
        if (
          userIsAdmin &&
          isSelfServeTopup(record) &&
          record.audit_status === 'pending'
        ) {
          actions.push(
            <Button
              key='self-serve-approve'
              size='small'
              type='primary'
              theme='outline'
              onClick={() => handleSelfServeApprove(record)}
            >
              {t('通过')}
            </Button>,
            <Button
              key='self-serve-edit'
              size='small'
              theme='outline'
              icon={<IconEdit />}
              onClick={() => openSelfServeEditModal(record)}
            >
              {t('编辑')}
            </Button>,
            <Button
              key='self-serve-reject'
              size='small'
              type='danger'
              theme='outline'
              onClick={() => handleSelfServeReject(record)}
            >
              {t('拒绝')}
            </Button>,
          );
        }
        if (userIsAdmin && record.refund_request_status === 'pending') {
          actions.push(
            <Button
              key='refund-approve'
              size='small'
              type='warning'
              theme='outline'
              onClick={() => openRefundModal(record)}
            >
              {t('审批退款')}
            </Button>,
            <Button
              key='refund-reject'
              size='small'
              type='danger'
              theme='outline'
              onClick={() => handleRejectRefundRequest(record)}
            >
              {t('拒绝')}
            </Button>,
          );
        }
        return actions.length > 0 ? (
          <Space spacing={6} wrap>
            {actions}
          </Space>
        ) : null;
      },
    });

    baseColumns.push({
      title: t('创建时间'),
      dataIndex: 'create_time',
      key: 'create_time',
      render: (time) => timestamp2string(time),
    });

    return baseColumns;
  }, [
    t,
    userIsAdmin,
    confirmAdminComplete,
    handleQueryAlipayOfficial,
    handleQueryWechatPayOfficial,
    confirmCloseAlipayOfficial,
    confirmCloseWechatPayOfficial,
    openEditModal,
    openRefundModal,
    handleRejectRefundRequest,
    handleSelfServeApprove,
    handleSelfServeReject,
    openSelfServeEditModal,
  ]);

  const tableProps = {
    columns: compactMode ? columns.map(({ fixed, ...rest }) => rest) : columns,
    dataSource: topups,
    loading: loading,
    rowKey: 'id',
    size: 'small',
    empty: (
      <Empty
        image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
        darkModeImage={
          <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
        }
        description={t('暂无充值记录')}
        style={{ padding: 30 }}
      />
    ),
    scroll: compactMode ? undefined : { x: 'max-content' },
  };

  const pagination = {
    currentPage: page,
    pageSize: pageSize,
    total: total,
    showSizeChanger: true,
    pageSizeOpts: [10, 20, 50, 100],
    onPageChange: handlePageChange,
    onPageSizeChange: handlePageSizeChange,
  };

  const searchArea = !hideFilters ? (
    <div className={isPageVariant ? '' : 'mb-3'}>
      <Input
        prefix={<IconSearch />}
        placeholder={userIsAdmin ? t('用户ID/用户名/订单号') : t('订单号')}
        value={keyword}
        onChange={handleKeywordChange}
        showClear
        pure={isPageVariant}
        size={isPageVariant ? 'small' : 'default'}
      />
    </div>
  ) : null;

  return (
    <>
      {searchArea}
      {isPageVariant ? (
        <CardTable
          {...tableProps}
          className='rounded-xl overflow-hidden'
          pagination={pagination}
          hidePagination={hidePagination}
        />
      ) : (
        <Table {...tableProps} pagination={pagination} />
      )}
      <Modal
        title={t('编辑管理员充值订单')}
        visible={editVisible}
        onCancel={() => setEditVisible(false)}
        onOk={handleEditAdminTopup}
        confirmLoading={editLoading}
        okText={t('保存')}
        cancelText={t('取消')}
      >
        <div className='space-y-4'>
          <div>
            <Text type='tertiary'>{t('订单号')}</Text>
            <div>
              <Text copyable>{editRecord?.trade_no || '-'}</Text>
            </div>
          </div>
          <div>
            <Text type='tertiary'>{t('当前充值额度')}</Text>
            <div>
              <Text>{renderQuota(editRecord?.amount || 0)}</Text>
            </div>
          </div>
          <div>
            <Text type='tertiary'>{t('操作类型')}</Text>
            <RadioGroup
              type='button'
              value={editForm.operationType}
              onChange={(event) =>
                setEditField('operationType', event.target.value)
              }
              style={{ width: '100%', marginTop: 6 }}
            >
              <Radio value='recharge'>{t('管理员充值')}</Radio>
              <Radio value='gift'>{t('管理员赠送')}</Radio>
            </RadioGroup>
          </div>
          <div>
            <Text type='tertiary'>{t('充值额度')}</Text>
            <InputNumber
              min={1}
              precision={0}
              step={500000}
              value={editForm.amount}
              onChange={(value) => setEditField('amount', value)}
              placeholder={t('请输入充值额度')}
              style={{ width: '100%', marginTop: 6 }}
            />
            <Text type='secondary' size='small'>
              {t('显示额度')}：{renderQuota(editForm.amount || 0)}
            </Text>
          </div>
          {editForm.operationType === 'recharge' ? (
            <>
              <Checkbox
                checked={editForm.useDefaultMoney}
                onChange={(event) =>
                  setEditField('useDefaultMoney', event.target.checked)
                }
              >
                {t('按支付宝官方配置自动计算')}
              </Checkbox>
              <div className='rounded-md border border-dashed border-[var(--semi-color-border)] p-3'>
                <Text type='secondary' size='small'>
                  {editForm.useDefaultMoney
                    ? t(
                        '保存时按支付宝官方充值价格和手续费比例重新计算支付金额；手续费不退。',
                      )
                    : t(
                        '手动金额会直接写入账单；手续费不退，退款只按支付金额计算。',
                      )}
                </Text>
              </div>
              <div>
                <Text type='tertiary'>{t('手续费比例（%）')}</Text>
                <InputNumber
                  min={0}
                  precision={3}
                  step={0.1}
                  value={editForm.serviceFeePercent}
                  onChange={(value) => setEditField('serviceFeePercent', value)}
                  placeholder={t('例如：0.6')}
                  style={{ width: '100%', marginTop: 6 }}
                />
                <Text type='secondary' size='small'>
                  {t('当前手续费比例：{{percent}}%，手续费不退', {
                    percent: formatPercent(editForm.serviceFeePercent),
                  })}
                </Text>
              </div>
              <div>
                <Text type='tertiary'>{t('支付金额')}</Text>
                <InputNumber
                  prefix='¥'
                  min={0}
                  precision={2}
                  step={0.01}
                  value={editForm.money}
                  disabled={editForm.useDefaultMoney}
                  onChange={(value) => setEditField('money', value)}
                  placeholder={t('支付金额')}
                  style={{ width: '100%', marginTop: 6 }}
                />
              </div>
              <div>
                <Text type='tertiary'>{t('手续费')}</Text>
                <InputNumber
                  prefix='¥'
                  min={0}
                  precision={2}
                  step={0.01}
                  value={editForm.fee}
                  disabled={editForm.useDefaultMoney}
                  onChange={(value) => setEditField('fee', value)}
                  placeholder={t('手续费')}
                  style={{ width: '100%', marginTop: 6 }}
                />
              </div>
            </>
          ) : (
            <div className='rounded-md border border-dashed border-[var(--semi-color-border)] p-3'>
              <Text type='warning' size='small'>
                {t(
                  '转为管理员赠送后会同步修改日志、移出账单管理和充值排行，支付金额与手续费会归零。',
                )}
              </Text>
            </div>
          )}
        </div>
      </Modal>
      <Modal
        title={t('编辑自助充值订单')}
        visible={selfServeEditVisible}
        onCancel={() => setSelfServeEditVisible(false)}
        onOk={handleSelfServeUpdate}
        confirmLoading={selfServeEditLoading}
        okText={t('保存')}
        cancelText={t('取消')}
      >
        <div className='space-y-4'>
          <div>
            <Text type='tertiary'>{t('订单号')}</Text>
            <div>
              <Text copyable>{selfServeEditRecord?.trade_no || '-'}</Text>
            </div>
          </div>
          <div>
            <Text type='tertiary'>{t('用户ID')}</Text>
            <div>
              <Text>{selfServeEditRecord?.user_id || '-'}</Text>
            </div>
          </div>
          <div>
            <Text type='tertiary'>{t('申报金额')}</Text>
            <InputNumber
              prefix='¥'
              min={0.01}
              max={selfServeSingleMax || undefined}
              precision={2}
              step={0.01}
              value={selfServeEditForm.declaredMoney}
              onChange={(value) =>
                setSelfServeEditField('declaredMoney', value)
              }
              placeholder={t('请输入充值金额')}
              style={{ width: '100%', marginTop: 6 }}
            />
            <Text type='secondary' size='small'>
              {selfServeSingleMax
                ? t(
                    '单笔自助充值金额上限为 {{amount}} 元。保存后会按新金额调整用户余额。',
                    { amount: selfServeSingleMax.toFixed(2) },
                  )
                : t('请先配置自助充值限额')}
            </Text>
          </div>
          <div>
            <Text type='tertiary'>{t('交易订单号')}</Text>
            <Input
              value={selfServeEditForm.transactionNo}
              onChange={(value) =>
                setSelfServeEditField('transactionNo', value)
              }
              placeholder={t('请输入交易订单号')}
              showClear
              style={{ marginTop: 6 }}
            />
          </div>
          <div>
            <Text type='tertiary'>{t('审核备注')}</Text>
            <Input
              value={selfServeEditForm.reason}
              onChange={(value) => setSelfServeEditField('reason', value)}
              placeholder={t('审核备注，可留空')}
              showClear
              style={{ marginTop: 6 }}
            />
          </div>
        </div>
      </Modal>
      <Modal
        title={
          refundMode === 'admin_quota'
            ? t('管理员充值退款')
            : refundMode === 'approve'
              ? t('审批退款')
              : userIsAdmin
                ? isBalanceTopup(refundRecord)
                  ? t('余额支付退款')
                  : isSelfServeTopup(refundRecord)
                    ? t('自助支付退款')
                    : t('官方支付退款')
                : t('申请退款')
        }
        visible={refundVisible}
        onCancel={() => setRefundVisible(false)}
        onOk={handleRefund}
        confirmLoading={refundLoading}
        okText={refundMode === 'approve' ? t('审批通过并退款') : t('确认退款')}
        cancelText={t('取消')}
      >
        <div className='space-y-3'>
          <div>
            <Text type='tertiary'>{t('订单号')}</Text>
            <div>
              <Text copyable>{refundRecord?.trade_no || '-'}</Text>
            </div>
          </div>
          <div>
            <Text type='tertiary'>
              {refundMode === 'admin_quota'
                ? t('剩余可退额度')
                : t('剩余可退金额')}
            </Text>
            <div>
              {refundMode === 'admin_quota' ? (
                <Text type='danger'>
                  {renderQuota(getRemainingAdminRefundQuota(refundRecord))}
                </Text>
              ) : (
                <Text type='danger'>
                  ¥
                  {formatCurrency(
                    refundPreview?.max_refund_amount ??
                      getRemainingRefundMoney(refundRecord),
                  )}
                </Text>
              )}
            </div>
          </div>
          {refundMode !== 'admin_quota' && refundPreview?.is_subscription ? (
            <div>
              <Text type='tertiary'>{t('订阅未使用部分')}</Text>
              <div>
                <Text>
                  {Math.round(
                    Math.max(
                      0,
                      1 - Number(refundPreview.subscription_used_ratio || 0),
                    ) * 100,
                  )}
                  %
                </Text>
              </div>
            </div>
          ) : null}
          {refundRecord?.refund_request_id ? (
            <>
              <div>
                <Text type='tertiary'>{t('申请退款金额')}</Text>
                <div>
                  <Text type='warning'>
                    ¥{formatCurrency(refundRecord.refund_request_amount)}
                  </Text>
                </div>
              </div>
              <div>
                <Text type='tertiary'>{t('申请原因')}</Text>
                <div>
                  <Text>{refundRecord.refund_request_reason || '-'}</Text>
                </div>
              </div>
              {adminSelfServeRefundApprove ? (
                <div>
                  <Text type='tertiary'>{t('退款收款码')}</Text>
                  <div className='mt-2'>
                    {refundQRCode ? (
                      <div className='space-y-2'>
                        {renderQRCodePreview(refundQRCode, t('退款收款码'))}
                        <div>
                          <Text copyable type='secondary'>
                            {refundQRCode}
                          </Text>
                        </div>
                      </div>
                    ) : (
                      <Text type='warning'>{t('用户未提交退款收款码')}</Text>
                    )}
                  </div>
                </div>
              ) : null}
            </>
          ) : null}
          {refundMode === 'admin_quota' ? (
            <InputNumber
              min={1}
              max={getRemainingAdminRefundQuota(refundRecord)}
              step={500000}
              precision={0}
              value={refundQuota}
              onChange={setRefundQuota}
              placeholder={t('退款额度')}
              style={{ width: '100%' }}
            />
          ) : (
            <InputNumber
              prefix='¥'
              min={0.01}
              max={
                refundFull
                  ? getRemainingRefundMoney(refundRecord)
                  : (refundPreview?.max_refund_amount ??
                    getRemainingRefundMoney(refundRecord))
              }
              step={0.01}
              precision={2}
              value={refundAmount}
              onChange={setRefundAmount}
              placeholder={t('退款金额')}
              style={{ width: '100%' }}
            />
          )}
          {userIsAdmin ? (
            <label className='flex items-center gap-2 text-sm'>
              <input
                type='checkbox'
                checked={refundFull}
                onChange={(event) => {
                  const checked = event.target.checked;
                  setRefundFull(checked);
                  if (refundMode === 'admin_quota') {
                    if (checked) {
                      setRefundQuota(
                        getRemainingAdminRefundQuota(refundRecord),
                      );
                    }
                  } else if (checked) {
                    setRefundAmount(getRemainingRefundMoney(refundRecord));
                  } else if (refundPreview?.max_refund_amount) {
                    setRefundAmount(refundPreview.max_refund_amount);
                  }
                }}
              />
              <span>
                {refundMode === 'admin_quota'
                  ? t('全额退回剩余额度')
                  : t('全额退款')}
              </span>
            </label>
          ) : null}
          <Input
            value={refundReason}
            onChange={setRefundReason}
            placeholder={userIsAdmin ? t('退款原因，可留空') : t('退款原因')}
            showClear
          />
          {userSelfServeRefundApply ? (
            <div className='space-y-2'>
              <Text type='tertiary'>{t('退款收款码')}</Text>
              <Input
                value={refundQRCode}
                onChange={setRefundQRCode}
                placeholder={t(
                  '可粘贴二维码内容或支付链接，上传图片会自动解码',
                )}
                showClear
              />
              <input
                ref={refundQRCodeFileRef}
                type='file'
                accept='image/png,image/jpeg,image/webp'
                className='hidden'
                onChange={handleRefundQRCodeFile}
              />
              <Button
                icon={<ImageUp size={16} />}
                theme='outline'
                onClick={() => refundQRCodeFileRef.current?.click()}
              >
                {t('上传退款收款码')}
              </Button>
              <div>
                <Text type='secondary' size='small'>
                  {t('系统只保存二维码内容；管理员手动退款后会审批录入系统。')}
                </Text>
              </div>
              {refundQRCode
                ? renderQRCodePreview(refundQRCode, t('退款收款码'))
                : null}
            </div>
          ) : null}
        </div>
      </Modal>
    </>
  );
};

export default TopupBillingTable;
