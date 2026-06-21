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

export const SUCCESS_RATE_LEVEL = {
  EXCELLENT: 'excellent',
  GOOD: 'good',
  WARNING: 'warning',
  CRITICAL: 'critical',
  UNKNOWN: 'unknown',
};

const SUCCESS_RATE_EXCELLENT_MIN = 100;
const SUCCESS_RATE_GOOD_MIN = 90;
const SUCCESS_RATE_WARNING_MIN = 70;

const SUCCESS_RATE_TAG_COLOR = {
  [SUCCESS_RATE_LEVEL.EXCELLENT]: 'green',
  [SUCCESS_RATE_LEVEL.GOOD]: 'teal',
  [SUCCESS_RATE_LEVEL.WARNING]: 'orange',
  [SUCCESS_RATE_LEVEL.CRITICAL]: 'red',
  [SUCCESS_RATE_LEVEL.UNKNOWN]: 'grey',
};

const SUCCESS_RATE_TEXT_CLASS = {
  [SUCCESS_RATE_LEVEL.EXCELLENT]: 'text-green-600',
  [SUCCESS_RATE_LEVEL.GOOD]: 'text-teal-600',
  [SUCCESS_RATE_LEVEL.WARNING]: 'text-orange-500',
  [SUCCESS_RATE_LEVEL.CRITICAL]: 'text-red-600',
  [SUCCESS_RATE_LEVEL.UNKNOWN]: 'text-gray-500',
};

const SUCCESS_RATE_DOT_CLASS = {
  [SUCCESS_RATE_LEVEL.EXCELLENT]: 'bg-green-600',
  [SUCCESS_RATE_LEVEL.GOOD]: 'bg-teal-600',
  [SUCCESS_RATE_LEVEL.WARNING]: 'bg-orange-500',
  [SUCCESS_RATE_LEVEL.CRITICAL]: 'bg-red-600',
  [SUCCESS_RATE_LEVEL.UNKNOWN]: 'bg-gray-500',
};

export function normalizePercent(value) {
  const num = Number(value);
  if (!Number.isFinite(num)) {
    return NaN;
  }
  const clamped = Math.min(100, Math.max(0, num));
  return Math.round(clamped * 100) / 100;
}

export function getSuccessRateLevel(rate) {
  const value = normalizePercent(rate);
  if (!Number.isFinite(value)) {
    return SUCCESS_RATE_LEVEL.UNKNOWN;
  }
  if (value >= SUCCESS_RATE_EXCELLENT_MIN) {
    return SUCCESS_RATE_LEVEL.EXCELLENT;
  }
  if (value >= SUCCESS_RATE_GOOD_MIN) {
    return SUCCESS_RATE_LEVEL.GOOD;
  }
  if (value >= SUCCESS_RATE_WARNING_MIN) {
    return SUCCESS_RATE_LEVEL.WARNING;
  }
  return SUCCESS_RATE_LEVEL.CRITICAL;
}

export function getSuccessRateTagColor(rate) {
  return SUCCESS_RATE_TAG_COLOR[getSuccessRateLevel(rate)];
}

export function getSuccessRateTextClass(rate) {
  return SUCCESS_RATE_TEXT_CLASS[getSuccessRateLevel(rate)];
}

export function getSuccessRateDotClass(rate) {
  return SUCCESS_RATE_DOT_CLASS[getSuccessRateLevel(rate)];
}

export function formatSuccessRate(rate) {
  const value = normalizePercent(rate);
  if (!Number.isFinite(value)) {
    return '-';
  }
  return `${value.toFixed(2)}%`;
}

function formatCompactNumber(value) {
  const num = Number(value);
  if (!Number.isFinite(num) || num <= 0) {
    return '-';
  }
  if (num > 1) {
    return String(Math.round(num));
  }
  return num.toFixed(1);
}

export function formatCompactLatency(ms) {
  const value = Number(ms);
  if (!Number.isFinite(value) || value <= 0) {
    return '-';
  }
  if (value >= 1000) {
    return `${formatCompactNumber(value / 1000)}s`;
  }
  return `${formatCompactNumber(value)}ms`;
}

export function formatCompactThroughput(tps) {
  const value = Number(tps);
  if (!Number.isFinite(value) || value <= 0) {
    return '-';
  }
  if (value >= 1000) {
    return `${formatCompactNumber(value / 1000)}Kt`;
  }
  return `${formatCompactNumber(value)}t`;
}

export function hasPerformanceSummary(perf) {
  if (!perf) {
    return false;
  }
  return (
    Number.isFinite(Number(perf.success_rate)) ||
    Number(perf.avg_latency_ms) > 0 ||
    Number(perf.avg_tps) > 0
  );
}
