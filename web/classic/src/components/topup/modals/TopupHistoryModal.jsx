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
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import {
  formatCurrency,
  getRemainingRefundMoney,
  isAlipayOfficialRefundable,
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
  wxpay_official: '微信支付官方',
};

const TopupHistoryModal = ({ visible, onCancel, t }) => {
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
  const isMobile = useIsMobile();

  const loadTopups = async (currentPage, currentPageSize) => {
    setLoading(true);
    try {
      const base = isAdmin() ? '/api/user/topup' : '/api/user/topup/self';
      const qs =
        `p=${currentPage}&page_size=${currentPageSize}` +
        (keyword ? `&keyword=${encodeURIComponent(keyword)}` : '');
      const endpoint = `${base}?${qs}`;
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
    if (visible) {
      loadTopups(page, pageSize);
    }
  }, [visible, page, pageSize, keyword]);

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

  const openRefundModal = (record) => {
    const remaining = getRemainingRefundMoney(record);
    setRefundRecord(record);
    setRefundAmount(remaining);
    setRefundReason('');
    setRefundVisible(true);
  };

  const handleRefund = async () => {
    if (!refundRecord) {
      return;
    }
    const remaining = getRemainingRefundMoney(refundRecord);
    const normalizedAmount = Math.round(Number(refundAmount || 0) * 100) / 100;
    if (!normalizedAmount || normalizedAmount <= 0) {
      Toast.error({ content: t('请输入退款金额') });
      return;
    }
    if (normalizedAmount > remaining) {
      Toast.error({ content: t('退款金额不能超过可退金额') });
      return;
    }
    setRefundLoading(true);
    try {
      const res = await API.post('/api/user/topup/alipay-official/refund', {
        trade_no: refundRecord.trade_no,
        refund_amount: normalizedAmount,
        reason: refundReason,
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

  const isSubscriptionTopup = (record) => {
    const tradeNo = (record?.trade_no || '').toLowerCase();
    return Number(record?.amount || 0) === 0 && tradeNo.startsWith('sub');
  };

  // 检查是否为管理员
  const userIsAdmin = useMemo(() => isAdmin(), []);

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

    // 管理员才显示操作列
    if (userIsAdmin) {
      baseColumns.push({
        title: t('操作'),
        key: 'action',
        render: (_, record) => {
          const actions = [];
          if (record.status === 'pending') {
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
            record.status === 'pending' &&
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
          if (isAlipayOfficialRefundable(record)) {
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
          return actions.length > 0 ? (
            <Space spacing={6} wrap>
              {actions}
            </Space>
          ) : null;
        },
      });
    }

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
    confirmCloseAlipayOfficial,
    openRefundModal,
  ]);

  return (
    <Modal
      title={t('充值账单')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size={isMobile ? 'full-width' : 'large'}
      width={isMobile ? undefined : 'min(1280px, calc(100vw - 48px))'}
    >
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
        title={t('支付宝退款')}
        visible={refundVisible}
        onCancel={() => setRefundVisible(false)}
        onOk={handleRefund}
        confirmLoading={refundLoading}
        okText={t('确认退款')}
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
                ¥{formatCurrency(getRemainingRefundMoney(refundRecord))}
              </Text>
            </div>
          </div>
          <InputNumber
            prefix='¥'
            min={0.01}
            max={getRemainingRefundMoney(refundRecord)}
            step={0.01}
            precision={2}
            value={refundAmount}
            onChange={setRefundAmount}
            placeholder={t('退款金额')}
            style={{ width: '100%' }}
          />
          <Input
            value={refundReason}
            onChange={setRefundReason}
            placeholder={t('退款原因，可留空')}
            showClear
          />
        </div>
      </Modal>
    </Modal>
  );
};

export default TopupHistoryModal;
