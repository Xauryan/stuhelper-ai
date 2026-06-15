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
import {
  Button,
  Empty,
  Progress,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { Activity, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showError, timestamp2string } from '../../helpers';

const { Text } = Typography;

const WINDOWS = [
  { label: '7d', seconds: 7 * 24 * 60 * 60 },
  { label: '15d', seconds: 15 * 24 * 60 * 60 },
  { label: '30d', seconds: 30 * 24 * 60 * 60 },
];

const pct = (value) => {
  const n = Number(value || 0);
  if (!Number.isFinite(n)) return '--';
  return `${(n * 100).toFixed(2)}%`;
};

const metricColor = (value, total) => {
  if (!total) return 'grey';
  if (value >= 0.99) return 'green';
  if (value >= 0.95) return 'lime';
  if (value >= 0.8) return 'yellow';
  return 'red';
};

const sourceLabel = (source, t) => {
  switch (source) {
    case 'probe':
      return t('主动探测');
    case 'log':
      return t('真实请求');
    default:
      return t('综合');
  }
};

const sourceTag = (source, t) => {
  const color =
    source === 'probe' ? 'blue' : source === 'log' ? 'green' : 'violet';
  return (
    <Tag color={color} size='small' shape='circle'>
      {sourceLabel(source, t)}
    </Tag>
  );
};

const MetricBlock = ({ title, bucket, t }) => {
  const total = Number(bucket?.total || 0);
  const sla = Number(bucket?.sla || 0);
  const color = metricColor(sla, total);
  const failureText = t(
    '失败 {{failures}} / 忽略 {{ignored}} / 平均 {{latency}} 秒',
    {
      failures: Number(bucket?.failures || 0),
      ignored: Number(bucket?.ignored || 0),
      latency: Number(bucket?.avg_use_time_seconds || 0).toFixed(2),
    },
  );

  return (
    <div
      className='min-w-[12rem] flex-1 rounded-lg border p-3'
      style={{ borderColor: 'var(--semi-color-border)' }}
    >
      <div className='flex items-center justify-between gap-2'>
        <Text strong>{title}</Text>
        <Tag color={color} size='small' shape='circle'>
          {total > 0 ? pct(sla) : '--'}
        </Tag>
      </div>
      <Progress
        percent={total > 0 ? sla * 100 : 0}
        showInfo={false}
        size='small'
        stroke={
          color === 'grey' ? 'var(--semi-color-disabled-text)' : undefined
        }
        className='mt-2'
      />
      <div className='mt-2 flex flex-wrap gap-2 text-xs text-gray-500'>
        <span>
          {t('样本')} {total}
        </span>
        <span>
          {t('成功')} {Number(bucket?.success || 0)}
        </span>
      </div>
      <div className='mt-1 text-xs text-gray-500'>{failureText}</div>
      {bucket?.last_error ? (
        <Tooltip content={bucket.last_error} position='topLeft'>
          <div className='mt-1 max-w-full truncate text-xs text-red-500'>
            {bucket.last_error}
          </div>
        </Tooltip>
      ) : null}
    </div>
  );
};

const ChannelMonitorPanel = ({ className = '', defaultWindowSeconds }) => {
  const { t } = useTranslation();
  const [windowSeconds, setWindowSeconds] = useState(
    defaultWindowSeconds || WINDOWS[0].seconds,
  );
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState(null);

  const load = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/channel/monitor/summary', {
        params: {
          window_seconds: windowSeconds,
          source: 'all',
          error_limit: 8,
        },
      });
      const { success, message, data: payload } = res.data || {};
      if (success) {
        setData(payload || null);
      } else {
        showError(message || t('获取渠道监控失败'));
      }
    } catch (error) {
      showError(t('获取渠道监控失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [windowSeconds]);

  const columns = useMemo(
    () => [
      {
        title: t('来源'),
        dataIndex: 'source',
        width: 100,
        render: (source) => sourceTag(source, t),
      },
      {
        title: t('时间'),
        dataIndex: 'created_at',
        width: 170,
        render: (value) => timestamp2string(value),
      },
      {
        title: t('渠道'),
        dataIndex: 'channel_name',
        width: 180,
        render: (text, record) => text || record.channel_id || '-',
      },
      {
        title: t('模型'),
        dataIndex: 'model_name',
        width: 180,
        render: (text) => text || '-',
      },
      {
        title: t('状态'),
        dataIndex: 'status_code',
        width: 90,
        render: (code, record) => (
          <Space spacing={4}>
            {code ? (
              <Tag
                size='small'
                shape='circle'
                color={code >= 500 ? 'red' : 'orange'}
              >
                {code}
              </Tag>
            ) : (
              <Tag size='small' shape='circle' color='grey'>
                --
              </Tag>
            )}
            {record.ignored ? (
              <Tooltip content={t('该错误未计入 SLA 分母')}>
                <Tag size='small' shape='circle' color='grey'>
                  {t('忽略')}
                </Tag>
              </Tooltip>
            ) : null}
          </Space>
        ),
      },
      {
        title: t('错误'),
        dataIndex: 'error_code',
        width: 180,
        render: (text, record) => text || record.error_type || '-',
      },
      {
        title: t('信息'),
        dataIndex: 'message',
        render: (text) => (
          <Tooltip content={text || '-'}>
            <span className='inline-block max-w-[32rem] truncate align-bottom'>
              {text || '-'}
            </span>
          </Tooltip>
        ),
      },
    ],
    [t],
  );

  return (
    <div
      className={`mb-3 rounded-lg border bg-white p-3 ${className}`}
      style={{ borderColor: 'var(--semi-color-border)' }}
    >
      <div className='mb-3 flex flex-col gap-2 md:flex-row md:items-center md:justify-between'>
        <div className='flex items-center gap-2'>
          <Activity size={16} />
          <Text strong>{t('渠道可用性监控')}</Text>
        </div>
        <Space spacing={6} wrap>
          {WINDOWS.map((item) => (
            <Button
              key={item.seconds}
              size='small'
              theme={windowSeconds === item.seconds ? 'solid' : 'outline'}
              type={windowSeconds === item.seconds ? 'primary' : 'tertiary'}
              onClick={() => setWindowSeconds(item.seconds)}
            >
              {item.label}
            </Button>
          ))}
          <Button
            icon={<RefreshCw size={14} />}
            size='small'
            theme='borderless'
            type='tertiary'
            loading={loading}
            onClick={load}
            aria-label={t('刷新')}
          />
        </Space>
      </div>
      <Spin spinning={loading}>
        <div className='flex flex-col gap-3'>
          <div className='flex flex-col gap-2 lg:flex-row'>
            <MetricBlock title={t('真实请求 SLA')} bucket={data?.log} t={t} />
            <MetricBlock title={t('主动探测 SLA')} bucket={data?.probe} t={t} />
            <MetricBlock title={t('综合 SLA')} bucket={data?.combined} t={t} />
          </div>
          <div>
            <div className='mb-2 flex items-center justify-between'>
              <Text strong>{t('最近错误')}</Text>
              <Text type='tertiary' size='small'>
                {data?.generated_at ? timestamp2string(data.generated_at) : ''}
              </Text>
            </div>
            {data?.errors?.length > 0 ? (
              <Table
                columns={columns}
                dataSource={data.errors}
                pagination={false}
                size='small'
                rowKey='id'
                scroll={{ x: 'max-content' }}
              />
            ) : (
              <Empty description={t('暂无错误记录')} />
            )}
          </div>
        </div>
      </Spin>
    </div>
  );
};

export default ChannelMonitorPanel;
