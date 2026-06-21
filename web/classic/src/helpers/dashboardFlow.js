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

import { modelToColor } from './render';

const DEFAULT_COLOR = '#1664ff';
const MASK_TEXT = '••••';
const MIN_STAGES = 2;
const DEFAULT_OVERFLOW_MODE = 'aggregate';
const FLOW_NODE_KINDS = ['user', 'node', 'token', 'group', 'model', 'channel'];
const FLOW_NODE_KIND_SET = new Set(FLOW_NODE_KINDS);

const ROLE_STAGES = {
  root: ['user', 'node', 'token', 'group', 'model', 'channel'],
  admin: ['user', 'group', 'model', 'channel'],
  user: ['token', 'group', 'model'],
};

const SENSITIVE_KINDS = new Set(['user', 'node', 'token', 'group', 'channel']);

const OTHER_LABEL_KEYS = {
  user: '其他用户',
  node: '其他节点',
  token: '其他令牌',
  group: '其他分组',
  model: '其他模型',
  channel: '其他渠道',
};

const numberValue = (value) => {
  const num = Number(value);
  return Number.isFinite(num) ? num : 0;
};

const formatNumber = (value) =>
  Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(
    numberValue(value),
  );

export const getFlowStages = (role) => ROLE_STAGES[role] || ROLE_STAGES.user;

export const getFlowMetricValue = (row, metric) => {
  if (metric === 'tokens') return numberValue(row.token_used);
  if (metric === 'requests') return numberValue(row.count);
  return numberValue(row.quota);
};

const rowMetrics = (row) => ({
  quota: numberValue(row.quota),
  tokens: numberValue(row.token_used),
  requests: numberValue(row.count),
});

const nodeId = (kind, value, fallback = 'unknown') =>
  `${kind}:${value || fallback}`;

const tokenLabel = (row, t) => {
  if (row.token_name) return row.token_name;
  const tokenId = numberValue(row.token_id);
  return tokenId > 0 ? t('已删除令牌 {{id}}', { id: tokenId }) : t('未知令牌');
};

const buildNode = (row, kind, t) => {
  switch (kind) {
    case 'user': {
      const userId = numberValue(row.user_id);
      const label =
        row.username || (userId > 0 ? `user-${userId}` : t('未知用户'));
      return {
        id: userId > 0 ? nodeId('user', userId) : nodeId('user', row.username),
        label,
        kind,
      };
    }
    case 'node': {
      const label = row.node_name || t('默认节点');
      return { id: nodeId('node', row.node_name || 'default'), label, kind };
    }
    case 'token': {
      const tokenId = numberValue(row.token_id);
      return {
        id:
          tokenId > 0
            ? nodeId('token', tokenId)
            : nodeId('token', row.token_name),
        label: tokenLabel(row, t),
        kind,
      };
    }
    case 'group': {
      const label = row.use_group || t('未知分组');
      return { id: nodeId('group', row.use_group), label, kind };
    }
    case 'channel': {
      const channelId = numberValue(row.channel_id);
      const label =
        row.channel_name ||
        (channelId > 0 ? `channel-${channelId}` : t('未知渠道'));
      return {
        id:
          channelId > 0
            ? nodeId('channel', channelId)
            : nodeId('channel', row.channel_name),
        label,
        kind,
      };
    }
    case 'model':
    default: {
      const label = row.model_name || t('未知模型');
      return { id: nodeId('model', row.model_name), label, kind: 'model' };
    }
  }
};

const buildPath = (row, stages, t) =>
  stages.map((stage) => buildNode(row, stage, t));

const normalizeStages = (role, visibleStages) => {
  const stages = getFlowStages(role);
  if (!visibleStages || visibleStages.length < MIN_STAGES) return stages;
  const visibleSet = new Set(visibleStages);
  const normalized = stages.filter((stage) => visibleSet.has(stage));
  return normalized.length >= MIN_STAGES ? normalized : stages;
};

const topNodeSets = (rows, stages, metric, topNodeLimit, t) => {
  const limit = Number(topNodeLimit);
  if (!Number.isFinite(limit) || limit <= 0) return null;

  const totalsByStage = new Map(stages.map((stage) => [stage, new Map()]));
  rows.forEach((row) => {
    const value = getFlowMetricValue(row, metric);
    buildPath(row, stages, t).forEach((node) => {
      const totals = totalsByStage.get(node.kind);
      if (!totals) return;
      const current = totals.get(node.id) || { node, value: 0 };
      current.value += value;
      totals.set(node.id, current);
    });
  });

  const sets = new Map();
  totalsByStage.forEach((totals, stage) => {
    const ids = Array.from(totals.values())
      .sort(
        (a, b) =>
          b.value - a.value ||
          a.node.label.localeCompare(b.node.label) ||
          a.node.id.localeCompare(b.node.id),
      )
      .slice(0, limit)
      .map((item) => item.node.id);
    sets.set(stage, new Set(ids));
  });
  return sets;
};

const applyTopNodeLimit = (
  path,
  sets,
  t,
  overflowMode = DEFAULT_OVERFLOW_MODE,
) => {
  if (!sets) return path;
  const hasOverflow = path.some((node) => {
    const stageSet = sets.get(node.kind);
    return stageSet && !stageSet.has(node.id);
  });
  if (!hasOverflow) return path;
  if (overflowMode === 'hide') return null;
  return path.map((node) => {
    const stageSet = sets.get(node.kind);
    if (!stageSet || stageSet.has(node.id)) return node;
    return {
      id: `${node.kind}:__other__`,
      label: t(OTHER_LABEL_KEYS[node.kind] || '其他'),
      kind: node.kind,
    };
  });
};

const selectedNodeKey = (filter) => `${filter.kind}\x00${filter.id}`;

export const flowNodeFilterKey = selectedNodeKey;

const normalizeSelectedNodeFilters = (selectedNodes, stages) => {
  const visibleKinds = new Set(stages);
  const filters = new Map();
  (selectedNodes || []).forEach((filter) => {
    if (!filter || !visibleKinds.has(filter.kind)) return;
    const selected = filters.get(filter.kind) || new Set();
    selected.add(String(filter.id));
    filters.set(filter.kind, selected);
  });
  return filters;
};

const pathMatchesNodeFilters = (path, filters) => {
  if (filters.size === 0) return true;
  const pathByKind = new Map();
  path.forEach((node) => {
    const values = pathByKind.get(node.kind) || new Set();
    values.add(node.id);
    pathByKind.set(node.kind, values);
  });

  for (const [kind, selectedIds] of filters) {
    const pathIds = pathByKind.get(kind);
    if (!pathIds) return false;
    let hasSelectedNode = false;
    for (const id of selectedIds) {
      if (pathIds.has(id)) {
        hasSelectedNode = true;
        break;
      }
    }
    if (!hasSelectedNode) return false;
  }
  return true;
};

const filterRowsByNodes = (rows, selectedNodes, stages, t) => {
  const filters = normalizeSelectedNodeFilters(selectedNodes, stages);
  if (filters.size === 0) return rows;
  return rows.filter((row) =>
    pathMatchesNodeFilters(buildPath(row, stages, t), filters),
  );
};

const selectedNodeFiltersExceptKind = (selectedNodes, kind) =>
  (selectedNodes || []).filter((filter) => filter.kind !== kind);

const addNode = (map, node, metrics, metric, color, colorKey) => {
  const current = map.get(node.id) || {
    id: node.id,
    label: node.label,
    kind: node.kind,
    value: 0,
    quota: 0,
    tokens: 0,
    requests: 0,
    color,
    colorKey,
  };
  current.value += metrics[metric] || 0;
  current.quota += metrics.quota;
  current.tokens += metrics.tokens;
  current.requests += metrics.requests;
  map.set(node.id, current);
};

const addLink = (map, source, target, metrics, metric, color, colorKey) => {
  const key = `${source.id}\x00${target.id}`;
  const current = map.get(key) || {
    source: source.id,
    target: target.id,
    sourceLabel: source.label,
    targetLabel: target.label,
    value: 0,
    quota: 0,
    tokens: 0,
    requests: 0,
    share: 0,
    color,
    linkColor: color,
    linkAlpha: 0.32,
    hoverColor: color,
    colorKey,
  };
  current.value += metrics[metric] || 0;
  current.quota += metrics.quota;
  current.tokens += metrics.tokens;
  current.requests += metrics.requests;
  map.set(key, current);
};

const alphaColor = (color, alpha) => {
  const normalized = String(color || DEFAULT_COLOR).trim();
  const hex = normalized.startsWith('#') ? normalized.slice(1) : normalized;
  if (!/^[0-9a-f]{6}$/i.test(hex)) {
    return { color: normalized, alpha };
  }
  const value = Number.parseInt(hex, 16);
  const r = (value >> 16) & 255;
  const g = (value >> 8) & 255;
  const b = value & 255;
  return { color: `rgba(${r}, ${g}, ${b}, ${alpha.toFixed(2)})`, alpha: 1 };
};

const applyLinkColors = (links) => {
  const bySource = new Map();
  links.forEach((link) => {
    const list = bySource.get(link.source) || [];
    list.push(link);
    bySource.set(link.source, list);
  });
  bySource.forEach((list) => {
    const sorted = [...list].sort(
      (a, b) =>
        b.value - a.value ||
        `${a.source}\x00${a.target}`.localeCompare(
          `${b.source}\x00${b.target}`,
        ),
    );
    const denominator = Math.max(sorted.length - 1, 1);
    sorted.forEach((link, index) => {
      const alpha =
        sorted.length === 1 ? 0.34 : 0.24 + (index / denominator) * 0.2;
      const display = alphaColor(link.color, alpha);
      link.linkColor = display.color;
      link.linkAlpha = display.alpha;
      link.hoverColor = link.color;
    });
  });
};

const linkStableKey = (link) => `${link.source}\x00${link.target}`;

const pathLinkKey = (source, target) => `${source.id}\x00${target.id}`;

const byLinkDrawPriority = (a, b) =>
  Number(Boolean(a.dimmed)) - Number(Boolean(b.dimmed)) ||
  Number(Boolean(b.highlighted)) - Number(Boolean(a.highlighted)) ||
  b.value - a.value ||
  linkStableKey(a).localeCompare(linkStableKey(b));

const pathContainsNode = (path, filter) =>
  path.some((node) => node.kind === filter.kind && node.id === filter.id);

const pathContainsLink = (path, link) => {
  for (let index = 0; index < path.length - 1; index += 1) {
    if (path[index].id === link.source && path[index + 1].id === link.target) {
      return true;
    }
  }
  return false;
};

const buildHighlightSets = (preparedPaths, activeNode, activeLink, stages) => {
  const nodeActive = Boolean(activeNode && stages.includes(activeNode.kind));
  if (!nodeActive && !activeLink) return null;

  const matchesPath = (path) => {
    if (activeLink) return pathContainsLink(path, activeLink);
    return activeNode ? pathContainsNode(path, activeNode) : false;
  };

  const nodes = new Set();
  const links = new Set();
  preparedPaths.forEach(({ path }) => {
    if (!matchesPath(path)) return;
    path.forEach((node) => nodes.add(node.id));
    for (let index = 0; index < path.length - 1; index += 1) {
      links.add(pathLinkKey(path[index], path[index + 1]));
    }
  });

  if (nodes.size === 0) return null;
  return { nodes, links };
};

const applyHighlights = (nodes, links, highlightSets) => {
  if (!highlightSets) return;
  nodes.forEach((node) => {
    node.highlighted = highlightSets.nodes.has(node.id);
    node.dimmed = !node.highlighted;
  });
  links.forEach((link) => {
    link.highlighted = highlightSets.links.has(linkStableKey(link));
    link.dimmed = !link.highlighted;
  });
};

const maskSensitiveLabels = (nodes, links) => {
  const masked = new Map();
  nodes.forEach((node) => {
    if (!SENSITIVE_KINDS.has(node.kind)) return;
    if (node.id.endsWith(':__other__')) return;
    node.label = MASK_TEXT;
    masked.set(node.id, MASK_TEXT);
  });
  links.forEach((link) => {
    if (masked.has(link.source)) link.sourceLabel = MASK_TEXT;
    if (masked.has(link.target)) link.targetLabel = MASK_TEXT;
  });
};

const buildSummary = (rows) =>
  rows.reduce(
    (summary, row) => {
      const metrics = rowMetrics(row);
      summary.quota += metrics.quota;
      summary.tokens += metrics.tokens;
      summary.requests += metrics.requests;
      return summary;
    },
    { quota: 0, tokens: 0, requests: 0 },
  );

const buildFilterOptions = (rows, stages, metric, t, selectedNodes) => {
  const stageOrder = new Map(stages.map((stage, index) => [stage, index]));
  const options = [];

  stages.forEach((stage) => {
    const totals = new Map();
    const candidateRows = filterRowsByNodes(
      rows,
      selectedNodeFiltersExceptKind(selectedNodes, stage),
      stages,
      t,
    );

    candidateRows.forEach((row) => {
      const metrics = rowMetrics(row);
      const node = buildNode(row, stage, t);
      const key = selectedNodeKey({ kind: node.kind, id: node.id });
      const current = totals.get(key) || { node, value: 0 };
      current.value += metrics[metric] || 0;
      totals.set(key, current);
    });

    totals.forEach(({ node, value }) => {
      options.push({
        kind: node.kind,
        value: node.id,
        label: node.label,
        valueLabel: formatNumber(value),
        valueRaw: value,
        color: modelToColor(node.id || node.label || 'flow') || DEFAULT_COLOR,
      });
    });
  });

  return options.sort(
    (a, b) =>
      (stageOrder.get(a.kind) || 0) - (stageOrder.get(b.kind) || 0) ||
      b.valueRaw - a.valueRaw ||
      a.label.localeCompare(b.label) ||
      a.value.localeCompare(b.value),
  );
};

export const buildDashboardFlowData = (
  rows,
  {
    role,
    metric,
    visibleStages,
    topNodeLimit,
    overflowMode = DEFAULT_OVERFLOW_MODE,
    selectedNodes,
    activeNode,
    activeLink,
    sensitiveVisible,
    t,
  },
) => {
  const safeRows = Array.isArray(rows) ? rows : [];
  const stages = normalizeStages(role, visibleStages);
  const filteredRows = filterRowsByNodes(safeRows, selectedNodes, stages, t);
  const topSets = topNodeSets(filteredRows, stages, metric, topNodeLimit, t);
  const nodes = new Map();
  const links = new Map();
  const summary = buildSummary(filteredRows);
  const preparedPaths = [];

  filteredRows.forEach((row) => {
    const path = applyTopNodeLimit(
      buildPath(row, stages, t),
      topSets,
      t,
      overflowMode,
    );
    if (!path || path.length < MIN_STAGES) return;
    const metrics = rowMetrics(row);
    preparedPaths.push({ path, metrics });
  });

  preparedPaths.forEach(({ path, metrics }) => {
    const root = path[0];
    const color =
      modelToColor(root?.id || root?.label || 'flow') || DEFAULT_COLOR;

    path.forEach((node) =>
      addNode(nodes, node, metrics, metric, color, root.id),
    );
    for (let index = 0; index < path.length - 1; index += 1) {
      addLink(
        links,
        path[index],
        path[index + 1],
        metrics,
        metric,
        color,
        root.id,
      );
    }
  });

  applyHighlights(
    nodes,
    links,
    buildHighlightSets(preparedPaths, activeNode, activeLink, stages),
  );

  const nodeList = Array.from(nodes.values()).sort(
    (a, b) => b.value - a.value || a.label.localeCompare(b.label),
  );
  const linkList = Array.from(links.values()).sort(
    (a, b) =>
      a.source.localeCompare(b.source) || a.target.localeCompare(b.target),
  );
  const firstSources = new Set(
    preparedPaths.map(({ path }) => path[0]?.id).filter(Boolean),
  );
  const total = linkList
    .filter((link) => firstSources.has(link.source))
    .reduce((sum, link) => sum + link.value, 0);
  linkList.forEach((link) => {
    link.share = total > 0 ? link.value / total : 0;
  });
  applyLinkColors(linkList);

  if (sensitiveVisible === false) {
    maskSensitiveLabels(nodeList, linkList);
  }

  return {
    summary,
    flow: {
      nodes: nodeList,
      links: linkList,
    },
    filterOptions: buildFilterOptions(
      safeRows,
      stages,
      metric,
      t,
      selectedNodes,
    ),
  };
};

const datumSource = (datum) => {
  if (!datum || typeof datum !== 'object') return {};
  if (Array.isArray(datum.datum)) {
    const depth = numberValue(datum.depth);
    return datum.datum[depth] || datum.datum[0] || datum;
  }
  return datum.datum && typeof datum.datum === 'object' ? datum.datum : datum;
};

const datumValue = (datum, key) => {
  if (datum && datum[key] !== undefined) return datum[key];
  return datumSource(datum)[key];
};

const datumFlag = (datum, key) => datumValue(datum, key) === true;

export const flowSankeyDatumValue = (datum, key) =>
  datum && typeof datum === 'object' ? datumValue(datum, key) : undefined;

const isSankeyLinkDatum = (datum) =>
  datumValue(datum, 'source') !== undefined &&
  datumValue(datum, 'target') !== undefined;

export const flowNodeFilterFromSankeyDatum = (datum) => {
  if (!datum || typeof datum !== 'object' || isSankeyLinkDatum(datum)) {
    return null;
  }
  const id = flowSankeyDatumValue(datum, 'key');
  const kind = flowSankeyDatumValue(datum, 'kind');
  if (
    (typeof id === 'string' || typeof id === 'number') &&
    FLOW_NODE_KIND_SET.has(kind)
  ) {
    return { kind, id: String(id) };
  }
  return null;
};

const tooltipMetricLines = (formatMetric, labels) => [
  {
    key: labels.quota,
    value: (datum) =>
      formatMetric(numberValue(datumValue(datum, 'quota')), 'quota'),
  },
  {
    key: labels.tokens,
    value: (datum) =>
      formatMetric(numberValue(datumValue(datum, 'tokens')), 'tokens'),
  },
  {
    key: labels.requests,
    value: (datum) =>
      formatMetric(numberValue(datumValue(datum, 'requests')), 'requests'),
  },
  {
    key: labels.share,
    value: (datum) =>
      `${(numberValue(datumValue(datum, 'share')) * 100).toFixed(1)}%`,
    visible: (datum) => numberValue(datumValue(datum, 'share')) > 0,
  },
];

export const buildFlowSankeySpec = (flow, title, formatMetric, labels) => ({
  type: 'sankey',
  data: [
    {
      id: 'flow',
      values: [
        {
          nodes: flow.nodes.map((node) => ({
            key: node.id,
            name: node.label,
            rawLabel: node.label,
            kind: node.kind,
            value: node.value,
            quota: node.quota,
            tokens: node.tokens,
            requests: node.requests,
            color: node.color,
            colorKey: node.colorKey,
            highlighted: node.highlighted,
            dimmed: node.dimmed,
          })),
          links: flow.links
            .filter((link) => link.value > 0)
            .sort(byLinkDrawPriority)
            .map((link, index) => {
              let zIndex = 100000 + index;
              if (link.highlighted) {
                zIndex = 1000000 + index;
              } else if (link.dimmed) {
                zIndex = index;
              }

              return {
                source: link.source,
                target: link.target,
                linkKey: linkStableKey(link),
                sourceLabel: link.sourceLabel,
                targetLabel: link.targetLabel,
                value: link.value,
                quota: link.quota,
                tokens: link.tokens,
                requests: link.requests,
                share: link.share,
                color: link.color,
                linkColor: link.linkColor,
                linkAlpha: link.linkAlpha,
                hoverColor: link.hoverColor,
                colorKey: link.colorKey,
                highlighted: link.highlighted,
                dimmed: link.dimmed,
                zIndex,
              };
            }),
        },
      ],
    },
  ],
  categoryField: 'name',
  sourceField: 'source',
  targetField: 'target',
  valueField: 'value',
  nodeKey: 'key',
  direction: 'horizontal',
  nodeAlign: 'justify',
  crossNodeAlign: 'middle',
  nodeGap: 14,
  nodeWidth: 16,
  minLinkHeight: 2,
  minNodeHeight: 8,
  title: {
    visible: false,
    text: title,
  },
  legends: { visible: false },
  label: {
    visible: true,
    position: 'outside',
    limit: 180,
    interactive: false,
    style: {
      fill: '#475569',
      fontSize: 11,
      fontWeight: 600,
    },
  },
  node: {
    interactive: true,
    style: {
      fill: (datum) => String(datumValue(datum, 'color') || DEFAULT_COLOR),
      fillOpacity: (datum) => {
        if (datumFlag(datum, 'dimmed')) return 0.18;
        if (datumFlag(datum, 'highlighted')) return 1;
        return 0.92;
      },
      stroke: (datum) =>
        datumFlag(datum, 'highlighted')
          ? 'rgba(15, 23, 42, 0.74)'
          : 'rgba(148, 163, 184, 0.45)',
      lineWidth: (datum) => (datumFlag(datum, 'highlighted') ? 1.5 : 1),
      cursor: 'pointer',
      pickMode: 'accurate',
    },
    state: {
      hover: {
        fillOpacity: 1,
        stroke: 'rgba(15, 23, 42, 0.68)',
        lineWidth: 1.5,
      },
      selected: {
        fillOpacity: 1,
        stroke: 'rgba(15, 23, 42, 0.68)',
        lineWidth: 1.5,
      },
      blur: {
        fillOpacity: 0.22,
      },
    },
  },
  link: {
    interactive: true,
    style: {
      fill: (datum) =>
        String(
          datumValue(datum, 'linkColor') ||
            datumValue(datum, 'color') ||
            DEFAULT_COLOR,
        ),
      fillOpacity: (datum) => {
        if (datumFlag(datum, 'dimmed')) return 0.08;
        if (datumFlag(datum, 'highlighted')) return 0.86;
        return numberValue(datumValue(datum, 'linkAlpha')) || 1;
      },
      cursor: 'pointer',
      pickMode: 'accurate',
      boundsMode: 'accurate',
      zIndex: (datum) => numberValue(datumValue(datum, 'zIndex')),
    },
    state: {
      hover: {
        fill: (datum) =>
          String(
            datumValue(datum, 'hoverColor') ||
              datumValue(datum, 'color') ||
              DEFAULT_COLOR,
          ),
        fillOpacity: 0.9,
      },
      selected: {
        fill: (datum) =>
          String(
            datumValue(datum, 'hoverColor') ||
              datumValue(datum, 'color') ||
              DEFAULT_COLOR,
          ),
        fillOpacity: 0.9,
      },
      blur: {
        fillOpacity: 0.22,
      },
    },
  },
  emphasis: { enable: false },
  tooltip: {
    trigger: 'hover',
    activeType: 'mark',
    dimension: { visible: false },
    group: { visible: false },
    mark: {
      checkOverlap: true,
      positionMode: 'pointer',
      title: {
        value: (datum) => {
          const source = datumValue(datum, 'source');
          const target = datumValue(datum, 'target');
          if (source && target) {
            return `${datumValue(datum, 'sourceLabel') || source} -> ${
              datumValue(datum, 'targetLabel') || target
            }`;
          }
          return `${datumValue(datum, 'name') || datumValue(datum, 'rawLabel') || ''}`;
        },
      },
      content: tooltipMetricLines(formatMetric, labels),
    },
  },
  background: { fill: 'transparent' },
  animation: false,
});
