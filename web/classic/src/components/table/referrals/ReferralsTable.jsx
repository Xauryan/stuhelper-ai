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

import React, { useMemo, useState } from 'react';
import { Button, Empty, Space, Tag, Typography } from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import CardTable from '../../common/ui/CardTable';
import { renderQuota, timestamp2string } from '../../../helpers';
import ReferralCommissionsModal from './ReferralCommissionsModal';

const { Text } = Typography;

const renderUser = (username, displayName, id) => (
  <div className='flex flex-col'>
    <Text strong>{displayName || username || '-'}</Text>
    <Text type='tertiary' size='small'>
      {username || '-'} · ID {id}
    </Text>
  </div>
);

const rewardTag = (record, t) => {
  if (record.inviter_reward_quota <= 0) {
    return <Tag size='small'>{t('无一次性奖励')}</Tag>;
  }
  if (record.inviter_reward_unlocked) {
    return (
      <Tag size='small' color='green'>
        {t('已解锁')}
      </Tag>
    );
  }
  return (
    <Tag size='small' color='orange'>
      {t('待首充')}
    </Tag>
  );
};

const ReferralsTable = ({
  records,
  loading,
  activePage,
  pageSize,
  recordCount,
  compactMode,
  handlePageChange,
  handlePageSizeChange,
  t,
}) => {
  const [selectedRecord, setSelectedRecord] = useState(null);

  const columns = useMemo(
    () => [
      {
        title: t('邀请人'),
        dataIndex: 'inviter_username',
        key: 'inviter',
        render: (_, record) =>
          renderUser(
            record.inviter_username,
            record.inviter_display_name,
            record.inviter_id,
          ),
      },
      {
        title: t('被邀请用户'),
        dataIndex: 'invitee_username',
        key: 'invitee',
        render: (_, record) =>
          renderUser(
            record.invitee_username,
            record.invitee_display_name,
            record.invitee_id,
          ),
      },
      {
        title: t('注册时间'),
        dataIndex: 'invitee_created_at',
        key: 'invitee_created_at',
        render: (text) => (text ? timestamp2string(text) : '-'),
      },
      {
        title: t('被邀请人奖励'),
        dataIndex: 'invitee_reward_quota',
        key: 'invitee_reward_quota',
        render: (text) => renderQuota(text || 0),
      },
      {
        title: t('邀请人奖励'),
        dataIndex: 'inviter_reward_quota',
        key: 'inviter_reward_quota',
        render: (_, record) => (
          <Space vertical align='start' spacing={2}>
            <Text>{renderQuota(record.inviter_reward_quota || 0)}</Text>
            {rewardTag(record, t)}
          </Space>
        ),
      },
      {
        title: t('首充/订阅'),
        dataIndex: 'invitee_has_paid',
        key: 'invitee_has_paid',
        render: (_, record) => (
          <Space vertical align='start' spacing={2}>
            {record.invitee_has_paid ? (
              <Tag size='small' color='green'>
                {t('已支付')}
              </Tag>
            ) : (
              <Tag size='small'>{t('未支付')}</Tag>
            )}
            <Text type='tertiary' size='small'>
              {record.first_payment_time
                ? timestamp2string(record.first_payment_time)
                : '-'}
            </Text>
          </Space>
        ),
      },
      {
        title: t('返佣汇总'),
        dataIndex: 'total_commission_quota',
        key: 'commission_summary',
        render: (_, record) => (
          <Space vertical align='start' spacing={2}>
            <Text>{renderQuota(record.total_commission_quota || 0)}</Text>
            <Text type='tertiary' size='small'>
              {t('次数')}: {record.commission_count || 0} · {t('金额')}: $
              {Number(record.total_recharge_amount || 0).toFixed(2)}
            </Text>
          </Space>
        ),
      },
      {
        title: t('操作'),
        dataIndex: 'operate',
        key: 'operate',
        fixed: 'right',
        render: (_, record) => (
          <Button
            type='tertiary'
            size='small'
            onClick={() => setSelectedRecord(record)}
          >
            {t('返佣记录')}
          </Button>
        ),
      },
    ],
    [t],
  );

  const tableColumns = useMemo(() => {
    if (!compactMode) return columns;
    return columns.map((column) => {
      if (column.key === 'operate') {
        const { fixed, ...rest } = column;
        return rest;
      }
      return column;
    });
  }, [compactMode, columns]);

  return (
    <>
      <CardTable
        columns={tableColumns}
        dataSource={records}
        rowKey={(record) => `${record.inviter_id}-${record.invitee_id}`}
        scroll={compactMode ? undefined : { x: 'max-content' }}
        pagination={{
          currentPage: activePage,
          pageSize,
          total: recordCount,
          pageSizeOpts: [10, 20, 50, 100],
          showSizeChanger: true,
          onPageSizeChange: handlePageSizeChange,
          onPageChange: handlePageChange,
        }}
        hidePagination
        loading={loading}
        empty={
          <Empty
            image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
            darkModeImage={
              <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
            }
            description={t('暂无邀请关系')}
            style={{ padding: 30 }}
          />
        }
        className='overflow-hidden'
        size='middle'
      />
      <ReferralCommissionsModal
        visible={!!selectedRecord}
        onCancel={() => setSelectedRecord(null)}
        record={selectedRecord}
        t={t}
      />
    </>
  );
};

export default ReferralsTable;
