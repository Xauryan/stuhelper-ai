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

import React from 'react';
import { Avatar, Space, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import { Activity, Clock3, Gauge } from 'lucide-react';
import {
  formatCompactLatency,
  formatCompactThroughput,
  formatSuccessRate,
  getSuccessRateDotClass,
  getSuccessRateTagColor,
  getSuccessRateTextClass,
  hasPerformanceSummary,
} from '../../../../helpers';

const { Text } = Typography;

const METRIC_ICON_SIZE = 12;

function MetricTag({ color = 'white', icon, label, value }) {
  return (
    <Tooltip content={`${label}: ${value}`}>
      <Tag color={color} shape='circle' size='small' prefixIcon={icon}>
        {value}
      </Tag>
    </Tooltip>
  );
}

function PerformanceTrendDots({ rates, t }) {
  if (!Array.isArray(rates) || rates.length === 0) {
    return null;
  }
  return (
    <div className='flex items-center gap-1'>
      {rates.map((rate, index) => (
        <Tooltip
          key={`${rate}-${index}`}
          content={`${t('最近成功率')} ${index + 1}: ${formatSuccessRate(rate)}`}
        >
          <span
            className={`inline-block h-1.5 w-4 rounded-full ${getSuccessRateDotClass(rate)}`}
          />
        </Tooltip>
      ))}
    </div>
  );
}

export function ModelPerformanceSummary({ perf, t, variant = 'table' }) {
  if (!hasPerformanceSummary(perf)) {
    if (variant === 'panel') {
      return (
        <Text type='tertiary' size='small'>
          {t('暂无性能数据')}
        </Text>
      );
    }
    return <Text type='tertiary'>-</Text>;
  }

  const successRate = formatSuccessRate(perf.success_rate);
  const latency = formatCompactLatency(perf.avg_latency_ms);
  const throughput = formatCompactThroughput(perf.avg_tps);
  const successRateClassName = getSuccessRateTextClass(perf.success_rate);

  if (variant === 'card') {
    return (
      <div className='mt-2 flex flex-wrap items-center gap-1.5 text-xs'>
        <MetricTag
          color={getSuccessRateTagColor(perf.success_rate)}
          icon={<Activity size={METRIC_ICON_SIZE} />}
          label={t('成功率')}
          value={successRate}
        />
        <MetricTag
          icon={<Clock3 size={METRIC_ICON_SIZE} />}
          label={t('平均延迟')}
          value={latency}
        />
        <MetricTag
          icon={<Gauge size={METRIC_ICON_SIZE} />}
          label={t('平均吞吐')}
          value={throughput}
        />
      </div>
    );
  }

  if (variant === 'panel') {
    return (
      <div className='grid grid-cols-1 sm:grid-cols-3 gap-3'>
        <div className='rounded-lg border border-semi-color-border p-3'>
          <div className='flex items-center gap-2 text-xs text-semi-color-text-2'>
            <Activity size={14} />
            {t('成功率')}
          </div>
          <div className={`mt-2 font-mono text-lg ${successRateClassName}`}>
            {successRate}
          </div>
          <PerformanceTrendDots rates={perf.recent_success_rates} t={t} />
        </div>
        <div className='rounded-lg border border-semi-color-border p-3'>
          <div className='flex items-center gap-2 text-xs text-semi-color-text-2'>
            <Clock3 size={14} />
            {t('平均延迟')}
          </div>
          <div className='mt-2 font-mono text-lg text-semi-color-text-0'>
            {latency}
          </div>
        </div>
        <div className='rounded-lg border border-semi-color-border p-3'>
          <div className='flex items-center gap-2 text-xs text-semi-color-text-2'>
            <Gauge size={14} />
            {t('平均吞吐')}
          </div>
          <div className='mt-2 font-mono text-lg text-semi-color-text-0'>
            {throughput}
          </div>
        </div>
      </div>
    );
  }

  return (
    <Space wrap spacing={4}>
      <MetricTag
        color={getSuccessRateTagColor(perf.success_rate)}
        icon={<Activity size={METRIC_ICON_SIZE} />}
        label={t('成功率')}
        value={successRate}
      />
      <MetricTag
        icon={<Clock3 size={METRIC_ICON_SIZE} />}
        label={t('平均延迟')}
        value={latency}
      />
      <MetricTag
        icon={<Gauge size={METRIC_ICON_SIZE} />}
        label={t('平均吞吐')}
        value={throughput}
      />
    </Space>
  );
}

export function ModelPerformanceSection({ perf, t }) {
  return (
    <div>
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='green' className='mr-2 shadow-md'>
          <Activity size={16} />
        </Avatar>
        <div>
          <Text className='text-lg font-medium'>{t('性能摘要')}</Text>
          <div className='text-xs text-gray-600'>
            {t('近 24 小时模型调用表现')}
          </div>
        </div>
      </div>
      <ModelPerformanceSummary perf={perf} t={t} variant='panel' />
    </div>
  );
}
