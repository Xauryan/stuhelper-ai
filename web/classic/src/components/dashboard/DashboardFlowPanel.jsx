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
  Button,
  Empty,
  Select,
  Skeleton,
  Spin,
  Tag,
  Tooltip,
} from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import {
  Activity,
  Filter,
  Eye,
  EyeOff,
  GitBranch,
  Hash,
  Route,
  X,
  WalletCards,
} from 'lucide-react';
import {
  buildDashboardFlowData,
  buildFlowSankeySpec,
  flowNodeFilterFromSankeyDatum,
  flowNodeFilterKey,
  flowSankeyDatumValue,
  getFlowStages,
} from '../../helpers/dashboardFlow';
import { renderNumber, renderQuota, selectFilter } from '../../helpers';

const METRIC_OPTIONS = [
  { value: 'quota', label: '按额度', icon: WalletCards },
  { value: 'tokens', label: '按 Token', icon: Hash },
  { value: 'requests', label: '按次数', icon: Activity },
];

const TOP_LIMIT_OPTIONS = [10, 20, 50, 100];

const OVERFLOW_MODE_OPTIONS = [
  { value: 'aggregate', label: '合并到其他' },
  { value: 'hide', label: '隐藏超出项' },
];

const DEFAULT_FLOW_PREFERENCES = {
  metric: 'quota',
  topNodeLimit: 50,
  overflowMode: 'aggregate',
  sensitiveVisible: true,
  hiddenStages: [],
  selectedNodes: [],
};

const FLOW_PREFERENCES_KEY = 'dashboard-flow-preferences';

const STAGE_LABELS = {
  user: '用户',
  node: '部署节点',
  token: '令牌',
  group: '分组',
  model: '模型',
  channel: '渠道',
};

const formatCompactMetric = (value, metric) => {
  if (metric === 'quota') return renderQuota(value, 4);
  return renderNumber(value);
};

const chartRecordValue = (value) =>
  value && typeof value === 'object' ? value : null;

const chartGraphicDatum = (value) => {
  const record = chartRecordValue(value);
  const context = chartRecordValue(record?.context);
  const data = context?.data;
  if (Array.isArray(data)) return data[0];
  return data;
};

const looksLikeFlowDatum = (value) => {
  const record = chartRecordValue(value);
  if (!record) return false;
  return (
    (record.key !== undefined && record.kind !== undefined) ||
    (record.source !== undefined && record.target !== undefined)
  );
};

const flowChartEventDatum = (event) => {
  const record = chartRecordValue(event);
  if (!record) return undefined;
  if (record.datum !== undefined && record.datum !== null) return record.datum;

  const itemRecord = chartRecordValue(record.item);
  if (itemRecord?.datum !== undefined && itemRecord.datum !== null) {
    return itemRecord.datum;
  }

  const graphicDatum = chartGraphicDatum(record.item);
  if (graphicDatum !== undefined && graphicDatum !== null) return graphicDatum;

  const itemData = itemRecord?.data;
  if (Array.isArray(itemData)) return itemData[0];
  if (itemData !== undefined && itemData !== null) return itemData;

  return looksLikeFlowDatum(record) ? record : undefined;
};

const isSameFlowNodeFilter = (a, b) =>
  Boolean(a && b && a.kind === b.kind && a.id === b.id);

const readFlowPreferences = () => {
  try {
    const raw = localStorage.getItem(FLOW_PREFERENCES_KEY);
    if (!raw) return DEFAULT_FLOW_PREFERENCES;
    const parsed = JSON.parse(raw);
    const metric = METRIC_OPTIONS.some(
      (option) => option.value === parsed.metric,
    )
      ? parsed.metric
      : DEFAULT_FLOW_PREFERENCES.metric;
    const topNodeLimit = TOP_LIMIT_OPTIONS.includes(parsed.topNodeLimit)
      ? parsed.topNodeLimit
      : DEFAULT_FLOW_PREFERENCES.topNodeLimit;
    const overflowMode = OVERFLOW_MODE_OPTIONS.some(
      (option) => option.value === parsed.overflowMode,
    )
      ? parsed.overflowMode
      : DEFAULT_FLOW_PREFERENCES.overflowMode;
    return {
      ...DEFAULT_FLOW_PREFERENCES,
      ...parsed,
      metric,
      topNodeLimit,
      overflowMode,
      sensitiveVisible:
        typeof parsed.sensitiveVisible === 'boolean'
          ? parsed.sensitiveVisible
          : DEFAULT_FLOW_PREFERENCES.sensitiveVisible,
      hiddenStages: Array.isArray(parsed.hiddenStages)
        ? parsed.hiddenStages
        : [],
      selectedNodes: Array.isArray(parsed.selectedNodes)
        ? parsed.selectedNodes
        : [],
    };
  } catch {
    return DEFAULT_FLOW_PREFERENCES;
  }
};

const writeFlowPreferences = (preferences) => {
  try {
    localStorage.setItem(FLOW_PREFERENCES_KEY, JSON.stringify(preferences));
  } catch {
    // ignore localStorage failures
  }
};

const SummaryItem = ({ icon, title, value, loading }) => (
  <div className='min-w-0 rounded-lg border border-gray-100 bg-white/70 px-3 py-2'>
    <div className='flex items-center gap-2 text-xs text-gray-500'>
      {icon}
      <span className='truncate'>{title}</span>
    </div>
    <Skeleton
      loading={loading}
      active
      placeholder={
        <Skeleton.Title style={{ width: 72, height: 24, marginTop: 8 }} />
      }
    >
      <div
        className='mt-1 truncate text-lg font-semibold tabular-nums'
        title={value}
      >
        {value}
      </div>
    </Skeleton>
  </div>
);

const DashboardFlowPanel = ({ data, loading, flowRole, CHART_CONFIG, t }) => {
  const chartRef = useRef(null);
  const initialPreferences = useMemo(() => readFlowPreferences(), []);
  const [metric, setMetric] = useState(initialPreferences.metric);
  const [topNodeLimit, setTopNodeLimit] = useState(
    initialPreferences.topNodeLimit,
  );
  const [overflowMode, setOverflowMode] = useState(
    initialPreferences.overflowMode,
  );
  const [sensitiveVisible, setSensitiveVisible] = useState(
    initialPreferences.sensitiveVisible,
  );
  const [hiddenStages, setHiddenStages] = useState(
    initialPreferences.hiddenStages,
  );
  const [selectedNodes, setSelectedNodes] = useState(
    initialPreferences.selectedNodes,
  );
  const [activeFlowNode, setActiveFlowNode] = useState(null);
  const [activeFlowLink, setActiveFlowLink] = useState(null);
  const stages = useMemo(() => getFlowStages(flowRole), [flowRole]);
  const visibleStages = useMemo(
    () => stages.filter((stage) => !hiddenStages.includes(stage)),
    [stages, hiddenStages],
  );

  useEffect(() => {
    const visible = new Set(visibleStages);
    setSelectedNodes((prev) =>
      prev.filter((filter) => visible.has(filter.kind)),
    );
    setActiveFlowNode((prev) => (prev && visible.has(prev.kind) ? prev : null));
    setActiveFlowLink(null);
  }, [visibleStages]);

  useEffect(() => {
    const validStageSet = new Set(stages);
    setHiddenStages((prev) => prev.filter((stage) => validStageSet.has(stage)));
  }, [stages]);

  useEffect(() => {
    writeFlowPreferences({
      metric,
      topNodeLimit,
      overflowMode,
      sensitiveVisible,
      hiddenStages,
      selectedNodes,
    });
  }, [
    metric,
    topNodeLimit,
    overflowMode,
    sensitiveVisible,
    hiddenStages,
    selectedNodes,
  ]);

  const toggleStage = (stage) => {
    setHiddenStages((prev) => {
      const hidden = new Set(prev);
      if (hidden.has(stage)) {
        hidden.delete(stage);
      } else {
        const remaining = stages.filter((item) => !hidden.has(item)).length;
        if (remaining <= 2) return prev;
        hidden.add(stage);
      }
      return stages.filter((item) => hidden.has(item));
    });
  };

  const handleNodeFilterChange = (values) => {
    const selected = new Set(values || []);
    const optionByValue = new Map(
      flowData.filterOptions.map((option) => [option.value, option]),
    );
    setSelectedNodes(
      Array.from(selected)
        .map((value) => optionByValue.get(value))
        .filter(Boolean)
        .map((option) => ({ kind: option.kind, id: option.value })),
    );
  };

  const removeNodeFilter = (filter) => {
    const key = flowNodeFilterKey(filter);
    setSelectedNodes((prev) =>
      prev.filter((item) => flowNodeFilterKey(item) !== key),
    );
  };

  const clearNodeFilters = () => {
    setSelectedNodes([]);
  };

  const handleChartPointerDown = useCallback((event) => {
    const datum = flowChartEventDatum(event);
    const filter = flowNodeFilterFromSankeyDatum(datum);
    if (filter) {
      setActiveFlowLink(null);
      setActiveFlowNode((prev) =>
        isSameFlowNodeFilter(prev, filter) ? null : filter,
      );
      return;
    }

    const source = flowSankeyDatumValue(datum, 'source');
    const target = flowSankeyDatumValue(datum, 'target');
    if (typeof source === 'string' && typeof target === 'string') {
      setActiveFlowNode(null);
      setActiveFlowLink((prev) =>
        prev && prev.source === source && prev.target === target
          ? null
          : { source, target },
      );
      return;
    }

    setActiveFlowNode(null);
    setActiveFlowLink(null);
    chartRef.current?.clearState?.('selected');
    chartRef.current?.clearState?.('blur');
  }, []);

  const flowData = useMemo(
    () =>
      buildDashboardFlowData(data, {
        role: flowRole,
        metric,
        visibleStages,
        topNodeLimit,
        overflowMode,
        selectedNodes,
        activeNode: activeFlowNode,
        activeLink: activeFlowLink,
        sensitiveVisible,
        t,
      }),
    [
      data,
      flowRole,
      metric,
      visibleStages,
      topNodeLimit,
      overflowMode,
      selectedNodes,
      activeFlowNode,
      activeFlowLink,
      sensitiveVisible,
      t,
    ],
  );

  const selectedNodeValues = useMemo(
    () => selectedNodes.map((filter) => filter.id),
    [selectedNodes],
  );
  const selectedOptionByKey = useMemo(() => {
    const options = new Map();
    flowData.filterOptions.forEach((option) => {
      options.set(
        flowNodeFilterKey({ kind: option.kind, id: option.value }),
        option,
      );
    });
    return options;
  }, [flowData.filterOptions]);
  const selectedNodeOptions = useMemo(
    () =>
      selectedNodes.map((filter) => {
        const option = selectedOptionByKey.get(flowNodeFilterKey(filter));
        return {
          ...filter,
          label: option?.label || filter.id,
        };
      }),
    [selectedNodes, selectedOptionByKey],
  );
  const nodeFilterOptionList = useMemo(
    () =>
      flowData.filterOptions.map((option) => ({
        value: option.value,
        label: `${t(STAGE_LABELS[option.kind])}: ${option.label} · ${option.valueLabel}`,
      })),
    [flowData.filterOptions, t],
  );

  const spec = useMemo(
    () =>
      buildFlowSankeySpec(flowData.flow, t('流量流向'), formatCompactMetric, {
        quota: t('额度'),
        tokens: t('Token 数'),
        requests: t('请求次数'),
        share: t('占比'),
      }),
    [flowData.flow, t],
  );

  const hasFlow = flowData.flow.links.length > 0;
  const hasActiveHighlight = Boolean(activeFlowNode || activeFlowLink);

  return (
    <div className='flex h-full min-h-0 flex-col gap-3'>
      <div className='grid grid-cols-1 gap-2 sm:grid-cols-3'>
        <SummaryItem
          icon={<WalletCards size={14} />}
          title={t('额度')}
          value={renderQuota(flowData.summary.quota, 2)}
          loading={loading}
        />
        <SummaryItem
          icon={<Hash size={14} />}
          title={t('Token 数')}
          value={renderNumber(flowData.summary.tokens)}
          loading={loading}
        />
        <SummaryItem
          icon={<Activity size={14} />}
          title={t('请求次数')}
          value={renderNumber(flowData.summary.requests)}
          loading={loading}
        />
      </div>

      <div className='flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between'>
        <div className='flex flex-wrap items-center gap-2'>
          <Tag
            color='white'
            shape='circle'
            prefixIcon={<GitBranch size={13} />}
          >
            {t('流向宽度')}
          </Tag>
          {METRIC_OPTIONS.map((option) => {
            const Icon = option.icon;
            return (
              <Button
                key={option.value}
                size='small'
                type={metric === option.value ? 'primary' : 'tertiary'}
                icon={<Icon size={14} />}
                onClick={() => setMetric(option.value)}
              >
                {t(option.label)}
              </Button>
            );
          })}
          <Select
            size='small'
            value={topNodeLimit}
            onChange={setTopNodeLimit}
            className='w-28'
            optionList={TOP_LIMIT_OPTIONS.map((count) => ({
              value: count,
              label: t('Top {{count}}', { count }),
            }))}
          />
          <Select
            size='small'
            value={overflowMode}
            onChange={setOverflowMode}
            className='w-32'
            optionList={OVERFLOW_MODE_OPTIONS.map((option) => ({
              value: option.value,
              label: t(option.label),
            }))}
          />
        </div>

        <div className='flex flex-wrap items-center gap-2'>
          {hasActiveHighlight && (
            <Button
              size='small'
              type='tertiary'
              icon={<Route size={14} />}
              onClick={() => {
                setActiveFlowNode(null);
                setActiveFlowLink(null);
              }}
            >
              {t('清除高亮')}
            </Button>
          )}
          <Tooltip
            content={t('隐藏或显示可能暴露用户、令牌、分组、渠道的信息')}
          >
            <Button
              size='small'
              type={sensitiveVisible ? 'tertiary' : 'warning'}
              icon={sensitiveVisible ? <Eye size={14} /> : <EyeOff size={14} />}
              onClick={() => setSensitiveVisible((value) => !value)}
            >
              {sensitiveVisible ? t('隐藏敏感信息') : t('显示敏感信息')}
            </Button>
          </Tooltip>
        </div>
      </div>

      <div className='flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between'>
        <div className='flex min-w-0 flex-1 flex-col gap-1'>
          <div className='flex items-center gap-2 text-xs text-gray-500'>
            <Filter size={13} />
            <span>{t('节点筛选')}</span>
          </div>
          <Select
            multiple
            filter={selectFilter}
            size='small'
            value={selectedNodeValues}
            onChange={handleNodeFilterChange}
            className='w-full max-w-2xl'
            placeholder={t('全部节点')}
            emptyContent={t('暂无可筛选节点')}
            optionList={nodeFilterOptionList}
            maxTagCount={2}
          />
        </div>
        {selectedNodeOptions.length > 0 && (
          <div className='flex max-w-full flex-wrap items-center gap-1 pt-5 lg:max-w-[45%] lg:justify-end'>
            {selectedNodeOptions.map((option) => (
              <Tag
                key={flowNodeFilterKey(option)}
                color='light-blue'
                closable
                onClose={() => removeNodeFilter(option)}
                closeIcon={<X size={12} />}
              >
                {t(STAGE_LABELS[option.kind])}: {option.label}
              </Tag>
            ))}
            {selectedNodeOptions.length > 1 && (
              <Button size='small' type='tertiary' onClick={clearNodeFilters}>
                {t('清空筛选')}
              </Button>
            )}
          </div>
        )}
      </div>

      <div className='flex flex-wrap items-center gap-1'>
        <span className='mr-1 text-xs text-gray-500'>{t('流向阶段')}</span>
        {stages.map((stage) => {
          const visible = !hiddenStages.includes(stage);
          return (
            <Button
              key={stage}
              size='small'
              type={visible ? 'secondary' : 'tertiary'}
              icon={!visible ? <EyeOff size={13} /> : null}
              onClick={() => toggleStage(stage)}
            >
              {t(STAGE_LABELS[stage])}
            </Button>
          );
        })}
      </div>

      <div className='min-h-0 flex-1 rounded-lg border border-gray-100 bg-white/60 p-2'>
        {loading ? (
          <div className='flex h-full min-h-[320px] items-center justify-center'>
            <Spin size='large' />
          </div>
        ) : hasFlow ? (
          <VChart
            spec={spec}
            option={CHART_CONFIG}
            onReady={(instance) => {
              chartRef.current = instance;
            }}
            onPointerDown={handleChartPointerDown}
          />
        ) : (
          <div className='flex h-full min-h-[320px] items-center justify-center'>
            <Empty
              title={t('暂无流向数据')}
              description={t('当前时间范围内没有可用于流向分析的数据')}
            />
          </div>
        )}
      </div>
    </div>
  );
};

export default DashboardFlowPanel;
