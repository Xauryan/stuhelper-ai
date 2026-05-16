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
import { Button, Empty, Popover, Space, Tag, Tooltip } from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import CardTable from '../../common/ui/CardTable';
import { renderNumber, renderQuota, timestamp2string } from '../../../helpers';
import ReferralCommissionsModal from './ReferralCommissionsModal';

const renderTimestamp = (text) => (text ? timestamp2string(text) : '-');

const renderUser = ({ username, displayName, email, id }) => (
  <Space spacing={2} wrap>
    <span>{username || displayName || email || '-'}</span>
    <Tag color='white' shape='circle' className='!text-xs'>
      ID {id || '-'}
    </Tag>
  </Space>
);

const rewardTag = (record, t) => {
  if (record.inviter_reward_quota <= 0) {
    return (
      <Tag color='grey' shape='circle' size='small'>
        {t('无一次性奖励')}
      </Tag>
    );
  }
  if (record.inviter_reward_unlocked) {
    return (
      <Tag size='small' color='green' shape='circle'>
        {t('已解锁')}
      </Tag>
    );
  }
  return (
    <Tag size='small' color='orange' shape='circle'>
      {t('待首充')}
    </Tag>
  );
};

const renderInviteRewards = (record, t) => {
  const popoverContent = (
    <div className='text-xs p-2'>
      <div>
        {t('被邀请人奖励')}: {renderQuota(record.invitee_reward_quota || 0)}
      </div>
      <div>
        {t('邀请人奖励')}: {renderQuota(record.inviter_reward_quota || 0)}
      </div>
    </div>
  );

  return (
    <Popover content={popoverContent} position='top'>
      <Space spacing={2} wrap>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('被邀请人奖励')}: {renderQuota(record.invitee_reward_quota || 0)}
        </Tag>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('邀请人奖励')}: {renderQuota(record.inviter_reward_quota || 0)}
        </Tag>
        {rewardTag(record, t)}
      </Space>
    </Popover>
  );
};

const renderPaymentState = (record, t) => {
  const paid = record.invitee_has_paid;
  const content = (
    <Tag color={paid ? 'green' : 'grey'} shape='circle' size='small'>
      {paid ? t('已首充') : t('未首充')}
    </Tag>
  );
  if (!paid) {
    return content;
  }
  return (
    <Tooltip
      content={`${t('首次支付时间')}: ${renderTimestamp(record.first_payment_time)}`}
      position='top'
    >
      {content}
    </Tooltip>
  );
};

const renderCommissionSummary = (record, t) => (
  <Space spacing={2} wrap>
    <Tag color='white' shape='circle' className='!text-xs'>
      {t('收益')}: {renderQuota(record.total_commission_quota || 0)}
    </Tag>
    <Tag color='white' shape='circle' className='!text-xs'>
      {t('次数')}: {renderNumber(record.commission_count || 0)}
    </Tag>
    <Tag color='white' shape='circle' className='!text-xs'>
      {t('支付')}: ${Number(record.total_recharge_amount || 0).toFixed(2)}
    </Tag>
  </Space>
);

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
          renderUser({
            username: record.inviter_username,
            displayName: record.inviter_display_name,
            id: record.inviter_id,
          }),
      },
      {
        title: t('被邀请用户'),
        dataIndex: 'invitee_username',
        key: 'invitee',
        render: (_, record) =>
          renderUser({
            username: record.invitee_username,
            displayName: record.invitee_display_name,
            email: record.invitee_email,
            id: record.invitee_id,
          }),
      },
      {
        title: t('注册时间'),
        dataIndex: 'invitee_created_at',
        key: 'invitee_created_at',
        render: renderTimestamp,
      },
      {
        title: t('邀请奖励'),
        dataIndex: 'invitee_reward_quota',
        key: 'invite_rewards',
        render: (_, record) => renderInviteRewards(record, t),
      },
      {
        title: t('首充状态'),
        dataIndex: 'invitee_has_paid',
        key: 'invitee_has_paid',
        render: (_, record) => renderPaymentState(record, t),
      },
      {
        title: t('返佣汇总'),
        dataIndex: 'total_commission_quota',
        key: 'commission_summary',
        render: (_, record) => renderCommissionSummary(record, t),
      },
      {
        title: '',
        dataIndex: 'operate',
        key: 'operate',
        fixed: 'right',
        render: (_, record) => (
          <Tooltip content={t('查看返佣记录')} position='top'>
            <Button
              type='tertiary'
              size='small'
              onClick={() => setSelectedRecord(record)}
            >
              {t('返佣记录')}
            </Button>
          </Tooltip>
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
