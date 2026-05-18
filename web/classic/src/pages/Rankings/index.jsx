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
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Empty, Spin, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { API, showError } from '../../helpers';
import { renderQuota } from '../../helpers/render';
import { UserContext } from '../../context/User';
import './index.css';

const { Title } = Typography;

const PERIODS = [
  { value: 'day', label: '24小时' },
  { value: 'week', label: '7天' },
  { value: 'month', label: '30天' },
  { value: 'all', label: '全部' },
];

const METRICS = [
  {
    key: 'tokens',
    label: 'Token 用量',
    accessor: (row) => Number(row?.total_tokens || 0),
    format: (value) => formatCompactNumber(value),
  },
  {
    key: 'quota',
    label: '消费额度',
    accessor: (row) => Number(row?.total_quota || 0),
    format: (value) => renderQuota(value || 0, 2),
  },
  {
    key: 'calls',
    label: '调用次数',
    accessor: (row) => Number(row?.request_count || 0),
    format: (value) => formatCompactNumber(value),
  },
];
const METRIC_BY_KEY = METRICS.reduce((acc, item) => {
  acc[item.key] = item;
  return acc;
}, {});

function formatCompactNumber(value) {
  const num = Number(value || 0);
  if (!Number.isFinite(num)) return '0';
  const abs = Math.abs(num);
  if (abs >= 1000000000) return `${(num / 1000000000).toFixed(1)}B`;
  if (abs >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
  if (abs >= 10000) return `${(num / 1000).toFixed(1)}K`;
  return num.toLocaleString();
}

function computeElapsedMinutes(updatedAtSeconds) {
  if (!updatedAtSeconds) return 0;
  const diffMs = Date.now() - Number(updatedAtSeconds) * 1000;
  if (diffMs <= 0) return 0;
  return Math.floor(diffMs / 60000);
}

function sortRowsByMetric(rows, metricKey) {
  const metric = METRIC_BY_KEY[metricKey] || METRIC_BY_KEY.tokens;
  const indexed = rows.map((row, originalIndex) => ({ row, originalIndex }));
  indexed.sort((a, b) => {
    const va = metric.accessor(a.row);
    const vb = metric.accessor(b.row);
    if (vb !== va) return vb - va;
    return a.originalIndex - b.originalIndex;
  });
  return indexed.map(({ row }) => row);
}

function PeriodPills({ value, onChange, disabled }) {
  const { t } = useTranslation();
  const refs = useRef([]);

  const handleKey = useCallback(
    (event, index) => {
      if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
      event.preventDefault();
      const delta = event.key === 'ArrowRight' ? 1 : -1;
      const next = (index + delta + PERIODS.length) % PERIODS.length;
      const target = refs.current[next];
      if (target) {
        target.focus();
        onChange(PERIODS[next].value);
      }
    },
    [onChange],
  );

  return (
    <div
      className='rk-period-pills'
      role='tablist'
      aria-label={t('排行榜周期')}
    >
      {PERIODS.map((item, index) => {
        const active = item.value === value;
        return (
          <button
            key={item.value}
            ref={(el) => (refs.current[index] = el)}
            type='button'
            role='tab'
            aria-selected={active}
            tabIndex={active ? 0 : -1}
            data-active={active}
            className='rk-pill'
            disabled={disabled}
            onClick={() => onChange(item.value)}
            onKeyDown={(event) => handleKey(event, index)}
          >
            {t(item.label)}
          </button>
        );
      })}
    </div>
  );
}

function MetricTabs({ value, onChange }) {
  const { t } = useTranslation();
  return (
    <div className='rk-metric-tabs' role='group' aria-label={t('排行指标')}>
      {METRICS.map((item) => {
        const active = item.key === value;
        return (
          <button
            key={item.key}
            type='button'
            className='rk-pill rk-pill--metric'
            data-active={active}
            aria-pressed={active}
            onClick={() => onChange(item.key)}
          >
            {t(item.label)}
          </button>
        );
      })}
    </div>
  );
}

function ListTypeTabs({ value, onChange }) {
  const { t } = useTranslation();
  const items = [
    { key: 'consumption', label: '消费榜' },
    { key: 'recharge', label: '充值榜' },
  ];
  return (
    <div className='rk-list-tabs' role='tablist' aria-label={t('排行榜类型')}>
      {items.map((item) => {
        const active = item.key === value;
        return (
          <button
            key={item.key}
            type='button'
            role='tab'
            aria-selected={active}
            data-active={active}
            className='rk-pill rk-pill--list'
            onClick={() => onChange(item.key)}
          >
            {t(item.label)}
          </button>
        );
      })}
    </div>
  );
}

function MedalChip({ rank }) {
  return (
    <span className={`rk-chip rk-chip--rank-${rank}`} aria-hidden='true'>
      {rank}
    </span>
  );
}

function PodiumCard({ row, place, metric, isSelf }) {
  const { t } = useTranslation();
  if (!row) {
    return <div className='rankings-podium__slot' aria-hidden='true' />;
  }
  const valueText = metric.format(metric.accessor(row));
  return (
    <article
      className={`rankings-podium__card rankings-podium__card--${place}`}
      data-self={isSelf || undefined}
      aria-label={t('第 {{rank}} 名 {{name}}', {
        rank: place,
        name: row.display,
      })}
    >
      <div
        className={`rankings-podium__medal rankings-podium__medal--${place}`}
      >
        <span>{place}</span>
      </div>
      <div className='rankings-podium__name' title={row.display}>
        {row.display}
      </div>
      <div className='rankings-podium__metric'>
        <span className='rankings-podium__metric-label'>{t(metric.label)}</span>
        <span className='rankings-podium__metric-value'>{valueText}</span>
      </div>
    </article>
  );
}

function Podium({ rows, metric, meDisplay }) {
  const visualOrder = useMemo(() => {
    const [first, second, third] = rows;
    return [
      { row: second, place: 2 },
      { row: first, place: 1 },
      { row: third, place: 3 },
    ];
  }, [rows]);

  return (
    <div className='rankings-podium' role='presentation'>
      <div className='rankings-podium__halo' aria-hidden='true' />
      {visualOrder.map(({ row, place }) => (
        <PodiumCard
          key={place}
          row={row}
          place={place}
          metric={metric}
          isSelf={row && meDisplay && row.display === meDisplay}
        />
      ))}
    </div>
  );
}

function RankRow({
  row,
  displayRank,
  metric,
  visibleMetrics,
  rank1Value,
  isMe,
  staggerIndex,
}) {
  const { t } = useTranslation();
  const value = metric.accessor(row);
  const percent =
    rank1Value > 0 ? Math.min(100, Math.round((value / rank1Value) * 100)) : 0;

  const style = { '--rk-stagger': `${staggerIndex * 45}ms` };

  return (
    <tr
      className='rankings-row'
      data-self={isMe || undefined}
      data-rank={displayRank <= 3 ? displayRank : undefined}
      style={style}
    >
      <td className='rk-col-rank'>
        {displayRank <= 3 ? (
          <MedalChip rank={displayRank} />
        ) : (
          <span className='rk-rank-text'>#{displayRank}</span>
        )}
      </td>
      <td className='rk-col-user'>
        <span className='rk-user-name' title={row.display}>
          {row.display}
        </span>
        {isMe && (
          <span className='rk-self-tag' aria-label={t('就是你')}>
            {t('你')}
          </span>
        )}
      </td>
      {visibleMetrics.map((m) => {
        const active = m.key === metric.key;
        const cellValue = m.format(m.accessor(row));
        return (
          <td
            key={m.key}
            className='rk-col-num'
            data-active={active || undefined}
          >
            <span className='rk-num'>{cellValue}</span>
            {active && (
              <span
                className='rk-bar'
                style={{ '--pct': `${percent}%` }}
                aria-hidden='true'
              />
            )}
          </td>
        );
      })}
    </tr>
  );
}

function MeRow({ row, metric, rank1Value }) {
  const { t } = useTranslation();
  if (!row) return null;
  const value = metric.accessor(row);
  const percent =
    rank1Value > 0 ? Math.min(100, Math.round((value / rank1Value) * 100)) : 0;

  return (
    <div
      className='rankings-me'
      role='complementary'
      aria-label={t('你的排名')}
    >
      <div className='rankings-me__lead'>
        <span className='rk-chip rk-chip--me' aria-hidden='true'>
          #{row.rank}
        </span>
        <div className='rankings-me__identity'>
          <span className='rankings-me__tag'>
            {t('你 #{{rank}}', { rank: row.rank })}
          </span>
          <span className='rankings-me__name' title={row.display}>
            {row.display}
          </span>
        </div>
      </div>
      <div className='rankings-me__metrics'>
        {METRICS.map((m) => {
          const active = m.key === metric.key;
          return (
            <div
              key={m.key}
              className='rankings-me__metric'
              data-active={active || undefined}
            >
              <span className='rankings-me__metric-label'>{t(m.label)}</span>
              <span className='rankings-me__metric-value'>
                {m.format(m.accessor(row))}
              </span>
            </div>
          );
        })}
      </div>
      <div
        className='rankings-me__bar'
        style={{ '--pct': `${percent}%` }}
        aria-hidden='true'
      />
    </div>
  );
}

function SignInPrompt() {
  const { t } = useTranslation();
  return (
    <div className='rankings-signin' role='note'>
      <span className='rankings-signin__text'>{t('登录后查看你的排名')}</span>
      <Link to='/login' className='rankings-signin__btn'>
        {t('立即登录')}
      </Link>
    </div>
  );
}

function ErrorState({ onRetry }) {
  const { t } = useTranslation();
  return (
    <div className='rankings-error' role='alert'>
      <Empty description={t('加载排行榜失败')} style={{ padding: 24 }} />
      <button type='button' className='rankings-error__btn' onClick={onRetry}>
        {t('重试')}
      </button>
    </div>
  );
}

const Rankings = () => {
  const { t } = useTranslation();
  const [userState] = useContext(UserContext);
  const [period, setPeriod] = useState('week');
  const [listType, setListType] = useState('consumption');
  const [metric, setMetric] = useState('tokens');
  const [snapshot, setSnapshot] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  const fetchRankings = useCallback(async () => {
    setLoading(true);
    setError(false);
    try {
      const res = await API.get('/api/rankings/users', { params: { period } });
      if (res.data?.success) {
        setSnapshot(res.data.data);
      } else {
        setError(true);
        showError(res.data?.message || t('加载排行榜失败'));
      }
    } catch (err) {
      setError(true);
      showError(t('加载排行榜失败'));
    } finally {
      setLoading(false);
    }
  }, [period, t]);

  useEffect(() => {
    fetchRankings();
  }, [fetchRankings]);

  const isRecharge = listType === 'recharge';
  const effectiveMetric = isRecharge ? 'quota' : metric;
  const activeMetric = METRIC_BY_KEY[effectiveMetric] || METRICS[0];
  const visibleMetrics = isRecharge ? [METRIC_BY_KEY.quota] : METRICS;
  const sourceRows = isRecharge
    ? snapshot?.recharge || []
    : snapshot?.consumption || [];
  const sortedRows = useMemo(
    () => sortRowsByMetric(sourceRows, effectiveMetric),
    [sourceRows, effectiveMetric],
  );

  const rank1Value = useMemo(() => {
    if (!sortedRows.length) return 0;
    return activeMetric.accessor(sortedRows[0]) || 0;
  }, [sortedRows, activeMetric]);

  const podiumRows = useMemo(() => sortedRows.slice(0, 3), [sortedRows]);

  const meMatch = useMemo(() => {
    const me = snapshot?.me;
    const isLoggedIn = !!userState?.user;
    if (!me) {
      return { me: null, mode: isLoggedIn ? 'none' : 'guest' };
    }
    if (!me.total_quota && !me.total_tokens && !me.request_count) {
      return { me: null, mode: 'none' };
    }
    const matchIndex = sortedRows.findIndex(
      (r) => r.is_me || r.display === me.display,
    );
    if (matchIndex >= 0) {
      return {
        me: { ...me, rank: matchIndex + 1 },
        mode: 'inline',
      };
    }
    return { me, mode: 'sticky' };
  }, [snapshot, sortedRows, userState]);

  const [minutesAgo, setMinutesAgo] = useState(() =>
    computeElapsedMinutes(snapshot?.updated_at),
  );
  useEffect(() => {
    setMinutesAgo(computeElapsedMinutes(snapshot?.updated_at));
    if (!snapshot?.updated_at) return undefined;
    const interval = setInterval(() => {
      setMinutesAgo(computeElapsedMinutes(snapshot.updated_at));
    }, 30000);
    return () => clearInterval(interval);
  }, [snapshot?.updated_at]);

  const subtitle =
    minutesAgo > 0
      ? t('最近更新 {{minutes}} 分钟前', { minutes: minutesAgo })
      : null;

  return (
    <main className='rankings-page' aria-labelledby='rankings-title'>
      <section className='rankings-shell'>
        <header className='rankings-hero'>
          <div className='rankings-hero__halo' aria-hidden='true' />
          <div className='rankings-hero__text'>
            <Title heading={2} id='rankings-title' className='rankings-title'>
              {t('排行榜')}
            </Title>
            <p className='rankings-subtitle'>
              <span>{t('数据每分钟更新')}</span>
              {subtitle && (
                <>
                  <span className='rankings-subtitle__dot' aria-hidden='true'>
                    ·
                  </span>
                  <span>{subtitle}</span>
                </>
              )}
            </p>
          </div>
          <PeriodPills value={period} onChange={setPeriod} disabled={loading} />
        </header>

        <article className='rankings-content'>
          <ListTypeTabs value={listType} onChange={setListType} />

          {!isRecharge && podiumRows.length >= 3 && (
            <Podium
              rows={podiumRows}
              metric={activeMetric}
              meDisplay={meMatch.me?.display}
            />
          )}

          {!isRecharge && <MetricTabs value={metric} onChange={setMetric} />}

          <div className='rankings-table-wrap' aria-busy={loading}>
            <Spin spinning={loading}>
              {error ? (
                <ErrorState onRetry={fetchRankings} />
              ) : sortedRows.length === 0 ? (
                <Empty
                  description={t('暂无排行榜数据')}
                  style={{ padding: 32 }}
                />
              ) : (
                <table className='rankings-table' role='table'>
                  <caption className='sr-only'>
                    {isRecharge
                      ? t('用户充值排行（按 {{metric}} 排序）', {
                          metric: t(activeMetric.label),
                        })
                      : t('用户消费排行（按 {{metric}} 排序）', {
                          metric: t(activeMetric.label),
                        })}
                  </caption>
                  <thead>
                    <tr>
                      <th scope='col' className='rk-col-rank'>
                        #
                      </th>
                      <th scope='col' className='rk-col-user'>
                        {t('用户')}
                      </th>
                      {visibleMetrics.map((m) => (
                        <th
                          key={m.key}
                          scope='col'
                          className='rk-col-num'
                          aria-sort={
                            m.key === effectiveMetric ? 'descending' : 'none'
                          }
                          data-active={m.key === effectiveMetric || undefined}
                        >
                          {isRecharge ? t('充值额度') : t(m.label)}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {sortedRows.map((row, i) => (
                      <RankRow
                        key={`${row.display}-${i}`}
                        row={row}
                        displayRank={i + 1}
                        metric={activeMetric}
                        visibleMetrics={visibleMetrics}
                        rank1Value={rank1Value}
                        isMe={
                          !isRecharge &&
                          meMatch.mode === 'inline' &&
                          (row.is_me || meMatch.me?.display === row.display)
                        }
                        staggerIndex={Math.min(i, 9)}
                      />
                    ))}
                  </tbody>
                </table>
              )}
            </Spin>
          </div>

          {!isRecharge && meMatch.mode === 'sticky' && (
            <MeRow
              row={meMatch.me}
              metric={activeMetric}
              rank1Value={rank1Value}
            />
          )}
          {!isRecharge && meMatch.mode === 'guest' && <SignInPrompt />}
        </article>
      </section>
    </main>
  );
};

export default Rankings;
