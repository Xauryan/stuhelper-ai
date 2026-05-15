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

import React, { useEffect, useMemo, useState } from 'react';
import { Descriptions, Modal, Table, Tag } from '@douyinfe/semi-ui';
import {
  API,
  renderQuota,
  showError,
  timestamp2string,
} from '../../../helpers';

const sourceLabel = (sourceType, t) => {
  if (sourceType === 'subscription') return t('订阅支付');
  if (sourceType === 'topup') return t('充值');
  return sourceType || '-';
};

const ReferralCommissionsModal = ({ visible, onCancel, record, t }) => {
  const [commissions, setCommissions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  const loadCommissions = async (nextPage = page, nextPageSize = pageSize) => {
    if (!record?.invitee_id) {
      setCommissions([]);
      setTotal(0);
      return;
    }
    setLoading(true);
    try {
      const params = new URLSearchParams({
        p: String(nextPage),
        page_size: String(nextPageSize),
      });
      const res = await API.get(
        `/api/user/referrals/${record.invitee_id}/commissions?${params.toString()}`,
      );
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setCommissions(data.items || []);
      setTotal(data.total || 0);
      setPage(data.page <= 0 ? nextPage : data.page);
      setPageSize(data.page_size || nextPageSize);
    } catch (error) {
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  const columns = useMemo(
    () => [
      {
        title: t('来源'),
        dataIndex: 'source_type',
        render: (text) => <Tag size='small'>{sourceLabel(text, t)}</Tag>,
      },
      {
        title: t('支付方式'),
        dataIndex: 'payment_method',
        render: (text) => text || '-',
      },
      {
        title: t('净支付金额'),
        dataIndex: 'net_recharge_amount',
        render: (text) => `$${Number(text || 0).toFixed(2)}`,
      },
      {
        title: t('已退款金额'),
        dataIndex: 'refunded_recharge_amount',
        render: (text) => `$${Number(text || 0).toFixed(2)}`,
      },
      {
        title: t('返佣比例'),
        dataIndex: 'commission_rate',
        render: (text) => `${Number(text || 0).toFixed(2)}%`,
      },
      {
        title: t('返佣额度'),
        dataIndex: 'net_commission_quota',
        render: (text) => renderQuota(text || 0),
      },
      {
        title: t('已冲销返佣'),
        dataIndex: 'refunded_commission_quota',
        render: (text) => renderQuota(text || 0),
      },
      {
        title: t('时间'),
        dataIndex: 'created_at',
        render: (text) => (text ? timestamp2string(text) : '-'),
      },
    ],
    [t],
  );

  useEffect(() => {
    if (!visible || !record?.invitee_id) {
      return;
    }
    setPage(1);
    setCommissions([]);
    setTotal(0);
    loadCommissions(1, pageSize);
  }, [visible, record?.invitee_id]);

  return (
    <Modal
      title={t('邀请返佣明细')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size='large'
      width='min(860px, calc(100vw - 48px))'
    >
      {record && (
        <div className='flex flex-col gap-4'>
          <Descriptions
            size='small'
            data={[
              {
                key: t('邀请人'),
                value: `${record.inviter_username || '-'} (ID: ${record.inviter_id})`,
              },
              {
                key: t('被邀请用户'),
                value: `${record.invitee_username || '-'} (ID: ${record.invitee_id})`,
              },
              {
                key: t('返佣次数'),
                value: record.commission_count || 0,
              },
              {
                key: t('累计返佣'),
                value: renderQuota(record.total_commission_quota || 0),
              },
            ]}
          />
          <Table
            columns={columns}
            dataSource={commissions}
            rowKey='id'
            loading={loading}
            pagination={{
              currentPage: page,
              pageSize,
              total,
              showSizeChanger: true,
              pageSizeOpts: [10, 20, 50, 100],
              onPageChange: (nextPage) => {
                setPage(nextPage);
                loadCommissions(nextPage, pageSize);
              },
              onPageSizeChange: (nextPageSize) => {
                setPage(1);
                setPageSize(nextPageSize);
                loadCommissions(1, nextPageSize);
              },
            }}
            size='small'
          />
        </div>
      )}
    </Modal>
  );
};

export default ReferralCommissionsModal;
