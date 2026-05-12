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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Card,
  Empty,
  Select,
  Spin,
  TabPane,
  Tabs,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconCreditCard,
  IconHistogram,
  IconRefresh,
  IconUser,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';
import CardTable from '../../components/common/ui/CardTable';
import { renderQuota } from '../../helpers/render';
import './index.css';

const { Text, Title } = Typography;

const PERIODS = [
  { value: 'all', label: '总榜' },
  { value: 'month', label: '月榜' },
  { value: 'week', label: '周榜' },
  { value: 'day', label: '日榜' },
];

const LIST_TYPES = [
  {
    key: 'consumption',
    title: '用户消耗排行',
    totalKey: 'consumption_total',
    icon: <IconHistogram />,
    tone: 'blue',
  },
  {
    key: 'recharge',
    title: '充值排行',
    totalKey: 'recharge_total',
    icon: <IconCreditCard />,
    tone: 'green',
  },
];

function formatShare(value) {
  const numeric = Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) return '0%';
  if (numeric < 0.001) return '<0.1%';
  return `${(numeric * 100).toFixed(numeric < 0.01 ? 2 : 1)}%`;
}

function RankingBadge({ rank }) {
  const rankColor =
    rank === 1
      ? 'amber'
      : rank === 2
        ? 'grey'
        : rank === 3
          ? 'orange'
          : 'white';
  return (
    <Tag
      color={rankColor}
      shape='circle'
      size='large'
      style={{ minWidth: 36, justifyContent: 'center' }}
    >
      #{rank}
    </Tag>
  );
}

function StatCard({ item, total }) {
  const { t } = useTranslation();
  return (
    <Card className='rankings-stat-card' bodyStyle={{ padding: 18 }}>
      <div className='rankings-stat-header'>
        <span className={`rankings-stat-icon rankings-stat-icon-${item.tone}`}>
          {item.icon}
        </span>
        <Text strong>{t(item.title)}</Text>
      </div>
      <div className='rankings-stat-value'>{renderQuota(total || 0, 2)}</div>
      <Text type='secondary' size='small'>
        {t('当前周期累计额度')}
      </Text>
    </Card>
  );
}

function RankingTable({ rows, loading }) {
  const { t } = useTranslation();
  const columns = useMemo(
    () => [
      {
        title: t('排名'),
        dataIndex: 'rank',
        key: 'rank',
        width: 110,
        render: (rank) => <RankingBadge rank={rank} />,
      },
      {
        title: t('用户'),
        dataIndex: 'display',
        key: 'display',
        render: (display) => (
          <div className='rankings-user-cell'>
            <span className='rankings-user-avatar' aria-hidden='true'>
              <IconUser />
            </span>
            <span className='rankings-user-name'>{display}</span>
          </div>
        ),
      },
      {
        title: t('额度'),
        dataIndex: 'total_quota',
        key: 'total_quota',
        align: 'right',
        render: (quota) => (
          <span className='rankings-quota'>{renderQuota(quota || 0, 2)}</span>
        ),
      },
      {
        title: t('占比'),
        dataIndex: 'share',
        key: 'share',
        align: 'right',
        width: 120,
        render: (share) => <Text type='secondary'>{formatShare(share)}</Text>,
      },
    ],
    [t],
  );

  return (
    <CardTable
      columns={columns}
      dataSource={rows}
      loading={loading}
      rowKey={(row) => `${row.user_id}-${row.rank}`}
      hidePagination
      pagination={false}
      empty={
        <Empty description={t('暂无排行榜数据')} style={{ padding: 32 }} />
      }
      size='middle'
      className='rankings-table'
    />
  );
}

const Rankings = () => {
  const { t } = useTranslation();
  const [period, setPeriod] = useState('week');
  const [activeType, setActiveType] = useState('consumption');
  const [loading, setLoading] = useState(false);
  const [snapshot, setSnapshot] = useState(null);

  const fetchRankings = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/rankings/users', { params: { period } });
      if (res.data?.success) {
        setSnapshot(res.data.data);
      } else {
        showError(res.data?.message || t('加载排行榜失败'));
      }
    } catch (error) {
      showError(t('加载排行榜失败'));
    } finally {
      setLoading(false);
    }
  }, [period, t]);

  useEffect(() => {
    fetchRankings();
  }, [fetchRankings]);

  const activeConfig =
    LIST_TYPES.find((item) => item.key === activeType) || LIST_TYPES[0];
  const activeRows = snapshot?.[activeConfig.key] || [];

  return (
    <main className='rankings-page' aria-labelledby='rankings-title'>
      <section className='rankings-shell'>
        <Card className='rankings-hero' bodyStyle={{ padding: 24 }}>
          <div className='rankings-hero-main'>
            <div>
              <Text type='secondary' className='rankings-eyebrow'>
                {t('Leaderboards')}
              </Text>
              <Title heading={2} id='rankings-title' className='rankings-title'>
                {t('排行榜')}
              </Title>
              <Text type='secondary'>
                {t('查看用户消耗排行和充值排行，数据按当前周期实时汇总。')}
              </Text>
            </div>
            <div className='rankings-controls'>
              <Select
                value={period}
                onChange={setPeriod}
                aria-label={t('排行榜周期')}
                style={{ width: 140 }}
              >
                {PERIODS.map((item) => (
                  <Select.Option key={item.value} value={item.value}>
                    {t(item.label)}
                  </Select.Option>
                ))}
              </Select>
              <button
                type='button'
                className='rankings-refresh'
                onClick={fetchRankings}
                aria-label={t('刷新排行榜')}
                disabled={loading}
              >
                <IconRefresh />
              </button>
            </div>
          </div>
        </Card>

        <div className='rankings-stats-grid'>
          {LIST_TYPES.map((item) => (
            <StatCard
              key={item.key}
              item={item}
              total={snapshot?.[item.totalKey] || 0}
            />
          ))}
        </div>

        <Card className='rankings-content' bodyStyle={{ padding: 0 }}>
          <Tabs
            type='button'
            activeKey={activeType}
            onChange={setActiveType}
            className='rankings-tabs'
          >
            {LIST_TYPES.map((item) => (
              <TabPane
                key={item.key}
                itemKey={item.key}
                tab={
                  <span className='rankings-tab-label'>
                    {item.icon}
                    {t(item.title)}
                  </span>
                }
              />
            ))}
          </Tabs>
          <div className='rankings-table-wrap' aria-busy={loading}>
            <Spin spinning={loading}>
              <RankingTable rows={activeRows} loading={loading} />
            </Spin>
          </div>
        </Card>
      </section>
    </main>
  );
};

export default Rankings;
