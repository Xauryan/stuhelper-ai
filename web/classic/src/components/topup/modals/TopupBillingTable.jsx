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
import React, { useState, useEffect, useMemo } from 'react';
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
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Coins } from 'lucide-react';
import { IconSearch } from '@douyinfe/semi-icons';
import { API, timestamp2string } from '../../../helpers';
import { isAdmin } from '../../../helpers/utils';
import {
  formatCurrency,
  getRemainingRefundMoney,
  isOfficialRefundable,
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

// 支付方式映射
const PAYMENT_METHOD_MAP = {
  stripe: 'Stripe',
  creem: 'Creem',
  waffo: 'Waffo',
  waffo_pancake: 'Waffo Pancake',
  alipay: '支付宝',
  wxpay: '微信',
  alipay_official: '支付宝',
  wxpay_official: '微信',
};

const isOfficialTopupRefundable = (record) =>
  isOfficialRefundable(record) && !record.refund_request_id;

const TopupBillingTable = ({ active = true, pendingRefundOnly = false, t }) => {
  const userIsAdmin = useMemo(() => isAdmin(), []);
  const [loading, setLoading] = useState(false);
  const [topups, setTopups] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [keyword, setKeyword] = useState('');
  const [refundVisible, setRefundVisible] = useState(false);
  const [refundRecord, setRefundRecord] = useState(null);
  const [refundAmount, setRefundAmount] = useState(0);
  const [refundReason, setRefundReason] = useState('');
  const [refundLoading, setRefundLoading] = useState(false);
  const [refundPreview, setRefundPreview] = useState(null);
  const [refundMode, setRefundMode] = useState('direct');
  const [refundFull, setRefundFull] = useState(false);

  const loadTopups = async (currentPage, currentPageSize) => {
    setLoading(true);
    try {
      const base = isAdmin() ? '/api/user/topup' : '/api/user/topup/self';
      const params = new URLSearchParams({
        p: String(currentPage),
        page_size: String(currentPageSize),
      });
      if (keyword) {
        params.set('keyword', keyword);
      }
      if (pendingRefundOnly) {
        params.set('pending_refund', 'true');
      }
      const endpoint = `${base}?${params.toString()}`;
      const res = await API.get(endpoint);
      const { success, message, data } = res.data;
      if (success) {
        setTopups(data.items || []);
        setTotal(data.total || 0);
      } else {
        Toast.error({ content: message || t('加载失败') });
      }
    } catch (error) {
      Toast.error({ content: t('加载账单失败') });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (active) {
      loadTopups(page, pageSize);
    }
  }, [active, page, pageSize, keyword, pendingRefundOnly]);

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
  const handleAdminComplete = async (tradeNo) => {
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
  };

  const confirmAdminComplete = (tradeNo) => {
    Modal.confirm({
      title: t('确认补单'),
      content: t('是否将该订单标记为成功并为用户入账？'),
      onOk: () => handleAdminComplete(tradeNo),
    });
  };

  const openRefundModal = async (record) => {
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
    setRefundReason('');
    setRefundMode(
      record.refund_request_id && userIsAdmin ? 'approve' : 'direct',
    );
    setRefundFull(false);
    setRefundVisible(true);
  };

  const handleRefund = async () => {
    if (!refundRecord) {
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
        const endpoint = userIsAdmin
          ? isWechatOfficial
            ? '/api/user/topup/wechat-pay-official/refund'
            : '/api/user/topup/alipay-official/refund'
          : '/api/user/topup/official/refund/apply';
        res = await API.post(endpoint, {
          trade_no: refundRecord.trade_no,
          refund_amount: normalizedAmount,
          reason: refundReason,
          full_refund: refundFull,
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

  const handleRejectRefundRequest = (record) => {
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
  };

  const handleQueryAlipayOfficial = async (tradeNo) => {
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
  };

  const handleQueryWechatPayOfficial = async (tradeNo) => {
    try {
      const res = await API.post('/api/user/topup/wechat-pay-official/query', {
        trade_no: tradeNo,
      });
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
  };

  const confirmCloseAlipayOfficial = (tradeNo) => {
    Modal.confirm({
      title: t('关闭订单'),
      content: t('确认关闭该支付宝官方待支付订单？'),
      onOk: async () => {
        try {
          const res = await API.post('/api/user/topup/alipay-official/close', {
            trade_no: tradeNo,
          });
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
  };

  const confirmCloseWechatPayOfficial = (tradeNo) => {
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
  };

  // 渲染状态徽章
  const renderStatusBadge = (status) => {
    const config = STATUS_CONFIG[status] || { type: 'primary', key: status };
    return (
      <span className='flex items-center gap-2'>
        <Badge dot type={config.type} />
        <span>{t(config.key)}</span>
      </span>
    );
  };

  // 渲染支付方式
  const renderPaymentMethod = (pm) => {
    const displayName = PAYMENT_METHOD_MAP[pm];
    return <Text>{displayName ? t(displayName) : pm || '-'}</Text>;
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
        title: t('已退款'),
        dataIndex: 'refunded_money',
        key: 'refunded_money',
        render: (money) =>
          Number(money || 0) > 0 ? (
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
      title: t('操作'),
      key: 'action',
      render: (_, record) => {
        const actions = [];
        const subscriptionTopup = isSubscriptionTopup(record);
        if (userIsAdmin && record.status === 'pending' && !subscriptionTopup) {
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
          record.status === 'pending' &&
          record.payment_provider === 'alipay_official' &&
          !subscriptionTopup
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
        if (
          userIsAdmin &&
          record.status === 'pending' &&
          record.payment_provider === 'wxpay_official' &&
          !subscriptionTopup
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
        if (
          isOfficialTopupRefundable(record) &&
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
              {userIsAdmin ? t('退款') : t('申请退款')}
            </Button>,
          );
        }
        if (record.refund_request_status === 'pending') {
          if (userIsAdmin) {
            actions.push(
              <Button
                key='refund-approve'
                size='small'
                type='warning'
                theme='solid'
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
          } else {
            actions.push(
              <Tag key='refund-pending' color='orange' shape='circle'>
                {t('退款审核中')}
              </Tag>,
            );
          }
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
    openRefundModal,
    handleRejectRefundRequest,
  ]);

  return (
    <>
      <div className='mb-3'>
        <Input
          prefix={<IconSearch />}
          placeholder={userIsAdmin ? t('用户ID/用户名/订单号') : t('订单号')}
          value={keyword}
          onChange={handleKeywordChange}
          showClear
        />
      </div>
      <Table
        columns={columns}
        dataSource={topups}
        loading={loading}
        rowKey='id'
        pagination={{
          currentPage: page,
          pageSize: pageSize,
          total: total,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
          onPageChange: handlePageChange,
          onPageSizeChange: handlePageSizeChange,
        }}
        size='small'
        empty={
          <Empty
            image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
            darkModeImage={
              <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
            }
            description={t('暂无充值记录')}
            style={{ padding: 30 }}
          />
        }
        scroll={{ x: 'max-content' }}
      />
      <Modal
        title={
          refundMode === 'approve'
            ? t('审批退款')
            : userIsAdmin
              ? t('官方支付退款')
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
            <Text type='tertiary'>{t('剩余可退金额')}</Text>
            <div>
              <Text type='danger'>
                ¥
                {formatCurrency(
                  refundPreview?.max_refund_amount ??
                    getRemainingRefundMoney(refundRecord),
                )}
              </Text>
            </div>
          </div>
          {refundPreview?.is_subscription ? (
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
            </>
          ) : null}
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
          {userIsAdmin ? (
            <label className='flex items-center gap-2 text-sm'>
              <input
                type='checkbox'
                checked={refundFull}
                onChange={(event) => {
                  const checked = event.target.checked;
                  setRefundFull(checked);
                  if (checked) {
                    setRefundAmount(getRemainingRefundMoney(refundRecord));
                  } else if (refundPreview?.max_refund_amount) {
                    setRefundAmount(refundPreview.max_refund_amount);
                  }
                }}
              />
              <span>{t('全额退款')}</span>
            </label>
          ) : null}
          <Input
            value={refundReason}
            onChange={setRefundReason}
            placeholder={userIsAdmin ? t('退款原因，可留空') : t('退款原因')}
            showClear
          />
        </div>
      </Modal>
    </>
  );
};

export default TopupBillingTable;
