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

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '@douyinfe/semi-ui';
import {
  API,
  getTodayStartTimestamp,
  isAdmin,
  isAuditAdmin,
  showError,
  showSuccess,
  timestamp2string,
  renderQuota,
  renderNumber,
  getLogOther,
  copy,
  renderClaudeLogContent,
  renderLogContent,
  renderAudioModelPrice,
  renderClaudeModelPrice,
  renderModelPrice,
  renderTieredModelPrice,
  renderTaskBillingProcess,
} from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';
import { useTablePageSize } from '../common/useTablePageSize';
import ParamOverrideEntry from '../../components/table/usage-logs/components/ParamOverrideEntry';
import { getBusinessLogExpandedDetailText } from '../../components/table/usage-logs/usageLogDisplayRules.mjs';

export const useLogsData = () => {
  const { t } = useTranslation();

  // Define column keys for selection
  const COLUMN_KEYS = {
    TIME: 'time',
    CHANNEL: 'channel',
    USERNAME: 'username',
    TOKEN: 'token',
    GROUP: 'group',
    TYPE: 'type',
    MODEL: 'model',
    USE_TIME: 'use_time',
    PROMPT: 'prompt',
    COMPLETION: 'completion',
    COST: 'cost',
    RETRY: 'retry',
    IP: 'ip',
    DETAILS: 'details',
  };

  // Basic state
  const [logs, setLogs] = useState([]);
  const [expandData, setExpandData] = useState({});
  const [showStat, setShowStat] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadingStat, setLoadingStat] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [logCount, setLogCount] = useState(0);
  const [pageSize, setPageSize] = useTablePageSize(ITEMS_PER_PAGE);
  const [logType, setLogType] = useState(0);

  // User and admin
  const isAdminUser = isAdmin();
  const canReadAllLogs = isAuditAdmin();
  const canViewChannelDetail = isAdminUser;
  // Role-specific storage key to prevent different roles from overwriting each other
  const STORAGE_KEY = canReadAllLogs
    ? 'logs-table-columns-admin'
    : 'logs-table-columns-user';
  const BILLING_DISPLAY_MODE_STORAGE_KEY = canReadAllLogs
    ? 'logs-billing-display-mode-admin'
    : 'logs-billing-display-mode-user';

  // Statistics state
  const [stat, setStat] = useState({
    quota: 0,
    token: 0,
  });

  // Form state
  const [formApi, setFormApi] = useState(null);
  let now = new Date();
  const formInitValues = {
    username: '',
    token_name: '',
    model_name: '',
    channel: '',
    group: '',
    request_id: '',
    dateRange: [
      timestamp2string(getTodayStartTimestamp()),
      timestamp2string(now.getTime() / 1000 + 3600),
    ],
    logType: '0',
  };

  // Get default column visibility based on user role
  const getDefaultColumnVisibility = () => {
    return {
      [COLUMN_KEYS.TIME]: true,
      [COLUMN_KEYS.CHANNEL]: canReadAllLogs,
      [COLUMN_KEYS.USERNAME]: canReadAllLogs,
      [COLUMN_KEYS.TOKEN]: true,
      [COLUMN_KEYS.GROUP]: true,
      [COLUMN_KEYS.TYPE]: true,
      [COLUMN_KEYS.MODEL]: true,
      [COLUMN_KEYS.USE_TIME]: true,
      [COLUMN_KEYS.PROMPT]: true,
      [COLUMN_KEYS.COMPLETION]: true,
      [COLUMN_KEYS.COST]: true,
      [COLUMN_KEYS.RETRY]: canReadAllLogs,
      [COLUMN_KEYS.IP]: true,
      [COLUMN_KEYS.DETAILS]: true,
    };
  };

  const getInitialVisibleColumns = () => {
    const defaults = getDefaultColumnVisibility();
    const savedColumns = localStorage.getItem(STORAGE_KEY);

    if (!savedColumns) {
      return defaults;
    }

    try {
      const parsed = JSON.parse(savedColumns);
      const merged = { ...defaults, ...parsed };

      if (!canReadAllLogs) {
        merged[COLUMN_KEYS.CHANNEL] = false;
        merged[COLUMN_KEYS.USERNAME] = false;
        merged[COLUMN_KEYS.RETRY] = false;
      }

      return merged;
    } catch (e) {
      console.error('Failed to parse saved column preferences', e);
      return defaults;
    }
  };

  const getInitialBillingDisplayMode = () => {
    const savedMode = localStorage.getItem(BILLING_DISPLAY_MODE_STORAGE_KEY);
    if (savedMode === 'price' || savedMode === 'ratio') {
      return savedMode;
    }
    return localStorage.getItem('quota_display_type') === 'TOKENS'
      ? 'ratio'
      : 'price';
  };

  // Column visibility state
  const [visibleColumns, setVisibleColumns] = useState(
    getInitialVisibleColumns,
  );
  const [showColumnSelector, setShowColumnSelector] = useState(false);
  const [billingDisplayMode, setBillingDisplayMode] = useState(
    getInitialBillingDisplayMode,
  );
  const [sensitiveVisible, setSensitiveVisible] = useState(isAdminUser);

  // Compact mode
  const [compactMode, setCompactMode] = useTableCompactMode('logs');

  // User info modal state
  const [showUserInfo, setShowUserInfoModal] = useState(false);
  const [userInfoData, setUserInfoData] = useState(null);

  // Channel affinity usage cache stats modal state (admin only)
  const [
    showChannelAffinityUsageCacheModal,
    setShowChannelAffinityUsageCacheModal,
  ] = useState(false);
  const [channelAffinityUsageCacheTarget, setChannelAffinityUsageCacheTarget] =
    useState(null);
  const [showParamOverrideModal, setShowParamOverrideModal] = useState(false);
  const [paramOverrideTarget, setParamOverrideTarget] = useState(null);

  // Initialize default column visibility
  const initDefaultColumns = () => {
    const defaults = getDefaultColumnVisibility();
    setVisibleColumns(defaults);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(defaults));
  };

  // Handle column visibility change
  const handleColumnVisibilityChange = (columnKey, checked) => {
    const updatedColumns = { ...visibleColumns, [columnKey]: checked };
    setVisibleColumns(updatedColumns);
  };

  // Handle "Select All" checkbox
  const handleSelectAll = (checked) => {
    const allKeys = Object.keys(COLUMN_KEYS).map((key) => COLUMN_KEYS[key]);
    const updatedColumns = {};

    allKeys.forEach((key) => {
      if (
        (key === COLUMN_KEYS.CHANNEL ||
          key === COLUMN_KEYS.USERNAME ||
          key === COLUMN_KEYS.RETRY) &&
        !canReadAllLogs
      ) {
        updatedColumns[key] = false;
      } else {
        updatedColumns[key] = checked;
      }
    });

    setVisibleColumns(updatedColumns);
  };

  // Persist column settings to the role-specific STORAGE_KEY
  useEffect(() => {
    if (Object.keys(visibleColumns).length > 0) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(visibleColumns));
    }
  }, [visibleColumns]);

  useEffect(() => {
    localStorage.setItem(BILLING_DISPLAY_MODE_STORAGE_KEY, billingDisplayMode);
  }, [BILLING_DISPLAY_MODE_STORAGE_KEY, billingDisplayMode]);

  // 获取表单值的辅助函数，确保所有值都是字符串
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};

    let start_timestamp = timestamp2string(getTodayStartTimestamp());
    let end_timestamp = timestamp2string(now.getTime() / 1000 + 3600);

    if (
      formValues.dateRange &&
      Array.isArray(formValues.dateRange) &&
      formValues.dateRange.length === 2
    ) {
      start_timestamp = formValues.dateRange[0];
      end_timestamp = formValues.dateRange[1];
    }

    return {
      username: formValues.username || '',
      token_name: formValues.token_name || '',
      model_name: formValues.model_name || '',
      start_timestamp,
      end_timestamp,
      channel: formValues.channel || '',
      group: formValues.group || '',
      request_id: formValues.request_id || '',
      logType: formValues.logType ? parseInt(formValues.logType) : 0,
    };
  };

  // Statistics functions
  const getLogSelfStat = async () => {
    const {
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      group,
      logType: formLogType,
    } = getFormValues();
    const currentLogType = formLogType !== undefined ? formLogType : logType;
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    let url = `/api/log/self/stat?type=${currentLogType}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${group}`;
    url = encodeURI(url);
    let res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const getLogStat = async () => {
    const {
      username,
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      channel,
      group,
      logType: formLogType,
    } = getFormValues();
    const currentLogType = formLogType !== undefined ? formLogType : logType;
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    let url = `/api/log/stat?type=${currentLogType}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${group}`;
    url = encodeURI(url);
    let res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const handleEyeClick = async () => {
    if (loadingStat) {
      return;
    }
    setLoadingStat(true);
    if (canReadAllLogs) {
      await getLogStat();
    } else {
      await getLogSelfStat();
    }
    setShowStat(true);
    setLoadingStat(false);
  };

  // User info function
  const showUserInfoFunc = async (userId) => {
    if (!isAdminUser) {
      return;
    }
    const res = await API.get(`/api/user/${userId}`);
    const { success, message, data } = res.data;
    if (success) {
      setUserInfoData(data);
      setShowUserInfoModal(true);
    } else {
      showError(message);
    }
  };

  const openChannelAffinityUsageCacheModal = (affinity) => {
    const a = affinity || {};
    setChannelAffinityUsageCacheTarget({
      rule_name: a.rule_name || a.reason || '',
      using_group: a.using_group || '',
      key_hint: a.key_hint || '',
      key_fp: a.key_fp || '',
    });
    setShowChannelAffinityUsageCacheModal(true);
  };

  const openParamOverrideModal = (log, other) => {
    const lines = Array.isArray(other?.po) ? other.po.filter(Boolean) : [];
    if (lines.length === 0) {
      return;
    }
    setParamOverrideTarget({
      lines,
      modelName: log?.model_name || '',
      requestId: log?.request_id || '',
      requestPath: other?.request_path || '',
    });
    setShowParamOverrideModal(true);
  };

  // Format logs data
  const setLogsFormat = (logs) => {
    const requestConversionDisplayValue = (conversionChain) => {
      const chain = Array.isArray(conversionChain)
        ? conversionChain.filter(Boolean)
        : [];
      if (chain.length <= 1) {
        return t('原生格式');
      }
      return `${chain.join(' -> ')}`;
    };
    const adminQuotaOperationType = (log, other) => {
      const explicitType = other?.admin_info?.operation_type;
      if (explicitType === 'recharge' || explicitType === 'gift') {
        return explicitType;
      }
      const content = String(log?.content || '');
      if (
        content.startsWith('管理员充值用户额度') ||
        content.startsWith('管理员增加用户额度')
      ) {
        return 'recharge';
      }
      if (content.startsWith('管理员赠送用户额度')) {
        return 'gift';
      }
      return '';
    };
    const adminQuotaOperationLabel = (operationType) => {
      if (operationType === 'recharge') {
        return t('充值');
      }
      if (operationType === 'gift') {
        return t('赠送');
      }
      return operationType;
    };
    const auditActionLabel = (action) => {
      switch (action) {
        case 'login':
          return t('登录成功');
        case 'user.create':
          return t('创建用户');
        case 'user.update':
          return t('更新用户');
        case 'user.delete':
          return t('删除用户');
        case 'user.manage':
          return t('管理用户');
        case 'user.quota_add':
          return t('增加用户额度');
        case 'user.quota_subtract':
          return t('减少用户额度');
        case 'user.quota_override':
          return t('覆盖用户额度');
        case 'user.binding_clear':
          return t('清除用户绑定');
        case 'user.2fa_disable':
          return t('强制禁用两步验证');
        case 'user.passkey_register':
          return t('注册 Passkey');
        case 'user.passkey_delete':
          return t('删除 Passkey');
        case 'user.reset_passkey':
          return t('重置用户 Passkey');
        case 'user.topup_complete':
          return t('完成用户充值');
        case 'user.oauth_unbind':
          return t('解绑用户 OAuth');
        case 'option.update':
          return t('更新系统设置');
        case 'option.payment_compliance':
          return t('更新支付合规确认');
        case 'option.reset_ratio':
          return t('重置模型倍率');
        case 'option.clear_affinity_cache':
          return t('清理渠道亲和缓存');
        case 'custom_oauth.create':
          return t('创建自定义 OAuth');
        case 'custom_oauth.update':
          return t('更新自定义 OAuth');
        case 'custom_oauth.delete':
          return t('删除自定义 OAuth');
        case 'performance.clear_disk_cache':
          return t('清理磁盘缓存');
        case 'performance.gc':
          return t('触发垃圾回收');
        case 'performance.clear_logs':
          return t('清理性能日志');
        case 'channel.create':
          return t('创建渠道');
        case 'channel.update':
          return t('更新渠道');
        case 'channel.delete':
          return t('删除渠道');
        case 'channel.delete_batch':
          return t('批量删除渠道');
        case 'channel.delete_disabled':
          return t('删除禁用渠道');
        case 'channel.key_view':
          return t('查看渠道密钥');
        case 'channel.tag_disable':
          return t('禁用标签渠道');
        case 'channel.tag_enable':
          return t('启用标签渠道');
        case 'channel.tag_edit':
          return t('编辑标签渠道');
        case 'channel.tag_batch_set':
          return t('批量设置渠道标签');
        case 'channel.copy':
          return t('复制渠道');
        case 'channel.multi_key_manage':
          return t('管理多密钥渠道');
        case 'channel.upstream_apply':
          return t('应用渠道上游模型变更');
        case 'channel.upstream_apply_all':
          return t('批量应用渠道上游模型变更');
        case 'redemption.create':
          return t('创建兑换码');
        case 'redemption.update':
          return t('更新兑换码');
        case 'redemption.delete':
          return t('删除兑换码');
        case 'redemption.delete_invalid':
          return t('删除无效兑换码');
        case 'prefill_group.create':
          return t('创建预填分组');
        case 'prefill_group.update':
          return t('更新预填分组');
        case 'prefill_group.delete':
          return t('删除预填分组');
        case 'vendor.create':
          return t('创建供应商');
        case 'vendor.update':
          return t('更新供应商');
        case 'vendor.delete':
          return t('删除供应商');
        case 'model.create':
          return t('创建模型');
        case 'model.update':
          return t('更新模型');
        case 'model.delete':
          return t('删除模型');
        case 'model.sync_upstream':
          return t('同步上游模型');
        case 'deployment.create':
          return t('创建部署');
        case 'deployment.update':
          return t('更新部署');
        case 'deployment.delete':
          return t('删除部署');
        case 'subscription.plan_create':
          return t('创建订阅套餐');
        case 'subscription.plan_update':
          return t('更新订阅套餐');
        case 'subscription.plan_status_update':
          return t('更新订阅套餐状态');
        case 'subscription.bind':
          return t('绑定用户订阅');
        case 'log.clear':
          return t('清理历史日志');
        case 'generic':
          return t('管理操作');
        default:
          return action || '';
      }
    };
    const auditParamKeyLabel = (key) => {
      switch (key) {
        case 'id':
          return 'ID';
        case 'name':
          return t('名称');
        case 'username':
          return t('用户名称');
        case 'role':
          return t('角色');
        case 'type':
          return t('类型');
        case 'count':
          return t('数量');
        case 'quota':
          return t('额度');
        case 'method':
          return t('方式');
        case 'tag':
          return t('标签');
        case 'bindingType':
          return t('绑定类型');
        case 'sourceId':
          return t('来源 ID');
        case 'action':
          return t('动作');
        case 'changed_fields':
          return t('变更字段');
        case 'route':
          return t('路由');
        default:
          return key;
      }
    };
    const formatAuditValue = (value) => {
      if (Array.isArray(value)) {
        return value.join(', ');
      }
      if (value && typeof value === 'object') {
        return JSON.stringify(value);
      }
      if (value === true) {
        return t('是');
      }
      if (value === false) {
        return t('否');
      }
      if (value === undefined || value === null || value === '') {
        return '-';
      }
      return String(value);
    };
    const auditParamVisibleForCurrentRole = (action, key) => {
      if (!action?.startsWith?.('channel.') || canViewChannelDetail) {
        return true;
      }
      return [
        'id',
        'sourceId',
        'count',
        'changed_fields',
        'action',
        'status',
        'success',
      ].includes(key);
    };
    const renderAuditParams = (params, action = '') => {
      if (!params || typeof params !== 'object') {
        return null;
      }
      const entries = Object.entries(params).filter(([key]) =>
        auditParamVisibleForCurrentRole(action, key),
      );
      if (entries.length === 0) {
        return null;
      }
      return (
        <div style={{ whiteSpace: 'pre-line', lineHeight: 1.6 }}>
          {entries
            .map(
              ([key, value]) =>
                `${auditParamKeyLabel(key)}：${formatAuditValue(value)}`,
            )
            .join('\n')}
        </div>
      );
    };
    const loginMethodLabel = (method) => {
      if (method === 'password') {
        return t('密码');
      }
      if (method === '2fa') {
        return t('两步验证');
      }
      if (method === 'passkey') {
        return 'Passkey';
      }
      if (method === 'wechat') {
        return t('微信登录');
      }
      if (method === 'telegram') {
        return t('Telegram 登录');
      }
      if (typeof method === 'string' && method.startsWith('oauth:')) {
        return `OAuth: ${method.slice('oauth:'.length)}`;
      }
      if (method === 'oauth') {
        return 'OAuth';
      }
      return method || '-';
    };
    const authMethodLabel = (method) => {
      if (method === 'access_token') {
        return t('访问令牌');
      }
      if (method === 'session') {
        return t('会话');
      }
      return method || '-';
    };

    let expandDatesLocal = {};
    for (let i = 0; i < logs.length; i++) {
      logs[i].timestamp2string = timestamp2string(logs[i].created_at);
      logs[i].key = logs[i].id;
      let other = getLogOther(logs[i].other);
      const quotaOperationType = adminQuotaOperationType(logs[i], other);
      let expandDataLocal = [];

      if (canReadAllLogs && (logs[i].type === 0 || logs[i].type === 2)) {
        expandDataLocal.push({
          key: t('渠道信息'),
          value: `#${logs[i].channel}`,
        });
      }
      if (logs[i].request_id) {
        expandDataLocal.push({
          key: t('Request ID'),
          value: logs[i].request_id,
        });
      }
      if (other?.op?.action) {
        expandDataLocal.push({
          key: t('审计操作'),
          value: auditActionLabel(other.op.action),
        });
        const auditParams = renderAuditParams(other.op.params, other.op.action);
        if (auditParams) {
          expandDataLocal.push({
            key: t('审计参数'),
            value: auditParams,
          });
        }
      }
      if (logs[i].type === 7) {
        if (other?.login_method) {
          expandDataLocal.push({
            key: t('登录方式'),
            value: loginMethodLabel(other.login_method),
          });
        }
        if (other?.user_agent) {
          expandDataLocal.push({
            key: t('User-Agent'),
            value: (
              <div
                style={{
                  maxWidth: 600,
                  whiteSpace: 'normal',
                  wordBreak: 'break-word',
                  lineHeight: 1.6,
                }}
              >
                {other.user_agent}
              </div>
            ),
          });
        }
      }
      if (other?.ws || other?.audio) {
        expandDataLocal.push({
          key: t('语音输入'),
          value: other.audio_input,
        });
        expandDataLocal.push({
          key: t('语音输出'),
          value: other.audio_output,
        });
        expandDataLocal.push({
          key: t('文字输入'),
          value: other.text_input,
        });
        expandDataLocal.push({
          key: t('文字输出'),
          value: other.text_output,
        });
      }
      if (other?.cache_tokens > 0) {
        expandDataLocal.push({
          key: t('缓存 Tokens'),
          value: other.cache_tokens,
        });
      }
      if (other?.cache_creation_tokens > 0) {
        expandDataLocal.push({
          key: t('缓存创建 Tokens'),
          value: other.cache_creation_tokens,
        });
      }
      if (logs[i].type === 2) {
        if (other?.billing_mode !== 'tiered_expr') {
          expandDataLocal.push({
            key: t('日志详情'),
            value: other?.claude
              ? renderClaudeLogContent({
                  ...other,
                  displayMode: billingDisplayMode,
                })
              : renderLogContent({ ...other, displayMode: billingDisplayMode }),
          });
        }
        if (logs[i]?.content) {
          expandDataLocal.push({
            key: t('其他详情'),
            value: logs[i].content,
          });
        }
        if (canReadAllLogs && other?.reject_reason) {
          expandDataLocal.push({
            key: t('拦截原因'),
            value: other.reject_reason,
          });
        }
      }
      if (logs[i].type === 2) {
        let modelMapped =
          other?.is_model_mapped &&
          other?.upstream_model_name &&
          other?.upstream_model_name !== '';
        if (modelMapped) {
          expandDataLocal.push({
            key: t('请求并计费模型'),
            value: logs[i].model_name,
          });
          expandDataLocal.push({
            key: t('实际模型'),
            value: other.upstream_model_name,
          });
        }

        const isViolationFeeLog =
          other?.violation_fee === true ||
          Boolean(other?.violation_fee_code) ||
          Boolean(other?.violation_fee_marker);

        let content = '';
        if (!isViolationFeeLog && other?.billing_mode !== 'tiered_expr') {
          const logOpts = {
            ...other,
            prompt_tokens: logs[i].prompt_tokens,
            completion_tokens: logs[i].completion_tokens,
            displayMode: billingDisplayMode,
          };
          const isTaskLog = other?.is_task === true || other?.task_id != null;
          if (isTaskLog && other?.model_price === -1) {
            content = renderTaskBillingProcess(other, logs[i].content);
          } else if (other?.ws || other?.audio) {
            content = renderAudioModelPrice(logOpts);
          } else if (other?.claude) {
            content = renderClaudeModelPrice(logOpts);
          } else {
            content = renderModelPrice(logOpts);
          }
          expandDataLocal.push({
            key: t('计费过程'),
            value: content,
          });
        }
        if (other?.reasoning_effort) {
          expandDataLocal.push({
            key: t('Reasoning Effort'),
            value: other.reasoning_effort,
          });
        }
        if (other?.billing_mode === 'tiered_expr' && other?.expr_b64) {
          expandDataLocal.push({
            key: t('计费过程'),
            value: renderTieredModelPrice({
              ...other,
              prompt_tokens: logs[i].prompt_tokens,
              completion_tokens: logs[i].completion_tokens,
              displayMode: billingDisplayMode,
            }),
          });
        }
      }
      if (logs[i].type === 6) {
        const businessDetailText = getBusinessLogExpandedDetailText(logs[i]);
        if (businessDetailText) {
          expandDataLocal.push({
            key: t('日志详情'),
            value: businessDetailText,
          });
        }
        if (other?.task_id) {
          expandDataLocal.push({
            key: t('任务ID'),
            value: other.task_id,
          });
        }
        if (other?.reason) {
          expandDataLocal.push({
            key: t('失败原因'),
            value: (
              <div
                style={{
                  maxWidth: 600,
                  whiteSpace: 'normal',
                  wordBreak: 'break-word',
                  lineHeight: 1.6,
                }}
              >
                {other.reason}
              </div>
            ),
          });
        }
      }
      if (canReadAllLogs && logs[i].type === 6) {
        const adminInfo = other?.admin_info;
        if (adminInfo) {
          if (adminInfo.payment_method) {
            expandDataLocal.push({
              key: t('订单支付方式'),
              value: adminInfo.payment_method,
            });
          }
          if (adminInfo.callback_payment_method) {
            expandDataLocal.push({
              key: t('回调支付方式'),
              value: adminInfo.callback_payment_method,
            });
          }
          if (adminInfo.caller_ip) {
            expandDataLocal.push({
              key: t('回调调用者IP'),
              value: adminInfo.caller_ip,
            });
          }
          if (adminInfo.server_ip) {
            expandDataLocal.push({
              key: t('服务器IP'),
              value: adminInfo.server_ip,
            });
          }
          if (adminInfo.node_name) {
            expandDataLocal.push({
              key: t('节点名称'),
              value: adminInfo.node_name,
            });
          }
          if (adminInfo.version) {
            expandDataLocal.push({
              key: t('系统版本'),
              value: adminInfo.version,
            });
          }
        }
      }
      if (other?.request_path) {
        expandDataLocal.push({
          key: t('请求路径'),
          value: other.request_path,
        });
      }
      if (canReadAllLogs && other?.stream_status) {
        const ss = other.stream_status;
        const isOk = ss.status === 'ok';
        const statusLabel = isOk ? '✓ ' + t('正常') : '✗ ' + t('异常');
        let streamValue =
          statusLabel + ' (' + (ss.end_reason || 'unknown') + ')';
        if (ss.error_count > 0) {
          streamValue += ` [${t('软错误')}: ${ss.error_count}]`;
        }
        if (ss.end_error) {
          streamValue += ` - ${ss.end_error}`;
        }
        expandDataLocal.push({
          key: t('流状态'),
          value: streamValue,
        });
        if (Array.isArray(ss.errors) && ss.errors.length > 0) {
          expandDataLocal.push({
            key: t('流错误详情'),
            value: (
              <div
                style={{
                  maxWidth: 600,
                  whiteSpace: 'pre-line',
                  wordBreak: 'break-word',
                  lineHeight: 1.6,
                }}
              >
                {ss.errors.join('\n')}
              </div>
            ),
          });
        }
      }
      if (Array.isArray(other?.po) && other.po.length > 0) {
        expandDataLocal.push({
          key: t('参数覆盖'),
          value: (
            <ParamOverrideEntry
              count={other.po.length}
              t={t}
              onOpen={(event) => {
                event.stopPropagation();
                openParamOverrideModal(logs[i], other);
              }}
            />
          ),
        });
      }
      if (other?.billing_source === 'subscription') {
        const planId = other?.subscription_plan_id;
        const planTitle = other?.subscription_plan_title || '';
        const subscriptionId = other?.subscription_id;
        const unit = t('额度');
        const pre = other?.subscription_pre_consumed ?? 0;
        const postDelta = other?.subscription_post_delta ?? 0;
        const finalConsumed = other?.subscription_consumed ?? pre + postDelta;
        const remain = other?.subscription_remain;
        const total = other?.subscription_total;
        // Use multiple Description items to avoid an overlong single line.
        if (planId) {
          expandDataLocal.push({
            key: t('订阅套餐'),
            value: `#${planId} ${planTitle}`.trim(),
          });
        }
        if (subscriptionId) {
          expandDataLocal.push({
            key: t('订阅实例'),
            value: `#${subscriptionId}`,
          });
        }
        const settlementLines = [
          `${t('预扣')}：${pre} ${unit}`,
          `${t('结算差额')}：${postDelta > 0 ? '+' : ''}${postDelta} ${unit}`,
          `${t('最终抵扣')}：${finalConsumed} ${unit}`,
        ]
          .filter(Boolean)
          .join('\n');
        expandDataLocal.push({
          key: t('订阅结算'),
          value: (
            <div style={{ whiteSpace: 'pre-line' }}>{settlementLines}</div>
          ),
        });
        if (remain !== undefined && total !== undefined) {
          expandDataLocal.push({
            key: t('订阅剩余'),
            value: `${remain}/${total} ${unit}`,
          });
        }
        expandDataLocal.push({
          key: t('订阅说明'),
          value: t(
            'token 会按倍率换算成“额度/次数”，请求结束后再做差额结算（补扣/返还）。',
          ),
        });
      }
      const shouldShowGenericAuditDetails =
        canReadAllLogs &&
        logs[i].type !== 6 &&
        (logs[i].type !== 1 || quotaOperationType !== '');
      if (shouldShowGenericAuditDetails) {
        expandDataLocal.push({
          key: t('请求转换'),
          value: requestConversionDisplayValue(other?.request_conversion),
        });
      }
      if (shouldShowGenericAuditDetails) {
        let localCountMode = '';
        if (other?.admin_info?.local_count_tokens) {
          localCountMode = t('本地计费');
        } else {
          localCountMode = t('上游返回');
        }
        expandDataLocal.push({
          key: t('计费模式'),
          value: localCountMode,
        });
      }
      if (canReadAllLogs && (logs[i].type === 3 || quotaOperationType !== '')) {
        const adminInfo = other?.admin_info;
        const hasUsername =
          adminInfo?.admin_username !== undefined &&
          adminInfo?.admin_username !== null &&
          adminInfo?.admin_username !== '';
        const hasId =
          adminInfo?.admin_id !== undefined &&
          adminInfo?.admin_id !== null &&
          adminInfo?.admin_id !== '';
        if (hasUsername || hasId) {
          let operatorValue = '';
          if (hasUsername && hasId) {
            operatorValue = `${adminInfo.admin_username} (ID: ${adminInfo.admin_id})`;
          } else if (hasUsername) {
            operatorValue = String(adminInfo.admin_username);
          } else {
            operatorValue = `ID: ${adminInfo.admin_id}`;
          }
          expandDataLocal.push({
            key: t('操作管理员'),
            value: operatorValue,
          });
        }
        if (adminInfo?.auth_method) {
          expandDataLocal.push({
            key: t('认证方式'),
            value: authMethodLabel(adminInfo.auth_method),
          });
        }
        if (quotaOperationType !== '') {
          expandDataLocal.push({
            key: t('操作类型'),
            value: adminQuotaOperationLabel(quotaOperationType),
          });
        }
      }
      if (canReadAllLogs && logs[i].type === 3 && other?.audit_info) {
        const auditInfo = other.audit_info;
        const routeLine = [auditInfo.method, auditInfo.route || auditInfo.path]
          .filter(Boolean)
          .join(' ');
        if (routeLine) {
          expandDataLocal.push({
            key: t('审计路由'),
            value: routeLine,
          });
        }
        if (auditInfo.path && auditInfo.path !== auditInfo.route) {
          expandDataLocal.push({
            key: t('请求路径'),
            value: auditInfo.path,
          });
        }
        if (auditInfo.status !== undefined) {
          expandDataLocal.push({
            key: t('HTTP 状态'),
            value: auditInfo.status,
          });
        }
        if (auditInfo.success !== undefined) {
          expandDataLocal.push({
            key: t('操作结果'),
            value: auditInfo.success ? t('成功') : t('失败'),
          });
        }
        const routeParams = renderAuditParams(
          auditInfo.params,
          other?.op?.action,
        );
        if (routeParams) {
          expandDataLocal.push({
            key: t('路由参数'),
            value: routeParams,
          });
        }
      }
      if (canReadAllLogs && logs[i].type === 1) {
        const adminInfo = other?.admin_info;
        if (adminInfo) {
          if (adminInfo.payment_method) {
            expandDataLocal.push({
              key: t('订单支付方式'),
              value: adminInfo.payment_method,
            });
          }
          if (adminInfo.callback_payment_method) {
            expandDataLocal.push({
              key: t('回调支付方式'),
              value: adminInfo.callback_payment_method,
            });
          }
          if (adminInfo.caller_ip) {
            expandDataLocal.push({
              key: t('回调调用者IP'),
              value: adminInfo.caller_ip,
            });
          }
          if (adminInfo.server_ip) {
            expandDataLocal.push({
              key: t('服务器IP'),
              value: adminInfo.server_ip,
            });
          }
          if (adminInfo.node_name) {
            expandDataLocal.push({
              key: t('节点名称'),
              value: adminInfo.node_name,
            });
          }
          if (adminInfo.version) {
            expandDataLocal.push({
              key: t('系统版本'),
              value: adminInfo.version,
            });
          }
        } else {
          expandDataLocal.push({
            key: t('审计信息'),
            value: (
              <span style={{ color: 'var(--semi-color-warning)' }}>
                {t(
                  '该记录由旧版本实例写入，缺少审计信息，建议将实例升级至最新版本以便记录服务器IP、回调IP、支付方式与系统版本等审计字段。',
                )}
              </span>
            ),
          });
        }
      }
      expandDatesLocal[logs[i].key] = expandDataLocal;
    }

    setExpandData(expandDatesLocal);
    setLogs(logs);
  };

  // Load logs function
  const loadLogs = async (startIdx, pageSize, customLogType = null) => {
    setLoading(true);

    let url = '';
    const {
      username,
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      channel,
      group,
      request_id,
      logType: formLogType,
    } = getFormValues();

    const currentLogType =
      customLogType !== null
        ? customLogType
        : formLogType !== undefined
          ? formLogType
          : logType;

    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    if (canReadAllLogs) {
      url = `/api/log/?p=${startIdx}&page_size=${pageSize}&type=${currentLogType}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${group}&request_id=${request_id}`;
    } else {
      url = `/api/log/self/?p=${startIdx}&page_size=${pageSize}&type=${currentLogType}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${group}&request_id=${request_id}`;
    }
    url = encodeURI(url);
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      const newPageData = data.items;
      setActivePage(data.page);
      setPageSize(data.page_size);
      setLogCount(data.total);

      setLogsFormat(newPageData);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  // Page handlers
  const handlePageChange = (page) => {
    setActivePage(page);
    loadLogs(page, pageSize).then((r) => {});
  };

  const handlePageSizeChange = async (size) => {
    setPageSize(size);
    setActivePage(1);
    loadLogs(1, size)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  };

  // Refresh function
  const refresh = async () => {
    setActivePage(1);
    handleEyeClick();
    await loadLogs(1, pageSize);
  };

  // Copy text function
  const copyText = async (e, text) => {
    e.stopPropagation();
    if (await copy(text)) {
      showSuccess('已复制：' + text);
    } else {
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  // Initialize data
  useEffect(() => {
    loadLogs(activePage, pageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  }, []);

  // Initialize statistics when formApi is available
  useEffect(() => {
    if (formApi) {
      handleEyeClick();
    }
  }, [formApi]);

  // Check if any record has expandable content
  const hasExpandableRows = () => {
    return logs.some(
      (log) => expandData[log.key] && expandData[log.key].length > 0,
    );
  };

  return {
    // Basic state
    logs,
    expandData,
    showStat,
    loading,
    loadingStat,
    activePage,
    logCount,
    pageSize,
    logType,
    stat,
    isAdminUser: canReadAllLogs,
    canViewUserDetail: isAdminUser,
    canViewChannelDetail,

    // Form state
    formApi,
    setFormApi,
    formInitValues,
    getFormValues,

    // Column visibility
    visibleColumns,
    showColumnSelector,
    setShowColumnSelector,
    billingDisplayMode,
    setBillingDisplayMode,
    sensitiveVisible,
    setSensitiveVisible,
    handleColumnVisibilityChange,
    handleSelectAll,
    initDefaultColumns,
    COLUMN_KEYS,

    // Compact mode
    compactMode,
    setCompactMode,

    // User info modal
    showUserInfo,
    setShowUserInfoModal,
    userInfoData,
    showUserInfoFunc,

    // Channel affinity usage cache stats modal
    showChannelAffinityUsageCacheModal,
    setShowChannelAffinityUsageCacheModal,
    channelAffinityUsageCacheTarget,
    openChannelAffinityUsageCacheModal,
    showParamOverrideModal,
    setShowParamOverrideModal,
    paramOverrideTarget,

    // Functions
    loadLogs,
    handlePageChange,
    handlePageSizeChange,
    refresh,
    copyText,
    handleEyeClick,
    setLogsFormat,
    hasExpandableRows,
    setLogType,
    openParamOverrideModal,

    // Translation
    t,
  };
};
