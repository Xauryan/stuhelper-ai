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

import React, { useContext } from 'react';
import { Empty, Typography } from '@douyinfe/semi-ui';
import {
  IllustrationNoAccess,
  IllustrationNoAccessDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';
import PropTypes from 'prop-types';
import { StatusContext } from '../../context/Status';

const { Text } = Typography;

const ACCESS_SOURCE_LABELS = {
  all: '全部来源',
  china_mainland: '中国大陆 IP',
  european_union: '欧盟 IP',
  unknown_country: '未知地区',
};

const ACCESS_RESOURCE_LABELS = {
  all: '全部资源',
  web: '官网页面',
  home: '官网首页',
  model_api: '模型 API',
  token: '令牌管理',
  wallet: '钱包充值',
  billing: '账单',
  usage_log: '使用日志',
  dashboard: '数据看板',
  playground: '操练场',
  chat: '聊天',
  personal: '个人设置',
  drawing_log: '绘图日志',
  task_log: '任务日志',
  admin_channel: '渠道管理',
  admin_subscription: '订阅管理',
  admin_model: '模型管理',
  admin_redemption: '兑换码管理',
  admin_user: '用户管理',
  admin_referral: '邀请管理',
  admin_setting: '系统设置',
};

const ACCESS_ROLE_LABELS = {
  guest: '游客',
  user: '普通用户',
  audit_admin: '审计管理员',
  admin: '管理员',
  root: '超级管理员',
  all: '全部角色',
};

const ACCESS_SCOPE_LABELS = {
  web: '官网 Web',
  api: 'API',
};

function accessSourceLabel(t, source) {
  if (!source) return t('未指定来源');
  return t(ACCESS_SOURCE_LABELS[source] || source);
}

function accessResourceLabel(t, resource) {
  if (!resource) return t('未指定资源');
  return t(ACCESS_RESOURCE_LABELS[resource] || resource);
}

function accessRoleLabel(t, role) {
  if (!role) return t('未识别角色');
  return t(ACCESS_ROLE_LABELS[role] || role);
}

function accessScopeLabel(t, scope) {
  if (!scope) return t('未指定范围');
  return t(ACCESS_SCOPE_LABELS[scope] || scope);
}

function accessDeniedReasonText(t, reason) {
  if (!reason) {
    return t('当前请求命中了访问限制策略。');
  }

  const source = accessSourceLabel(t, reason.source);
  const resource = accessResourceLabel(t, reason.resource);
  const role = accessRoleLabel(t, reason.role);
  switch (reason.kind) {
    case 'geo_china_mainland':
      return t('当前 IP 被识别为中国大陆，命中了中国大陆全局来源限制。');
    case 'geo_european_union':
      return t('当前 IP 被识别为欧盟地区，命中了欧盟全局来源限制。');
    case 'role_geo':
    case 'source_resource_all':
      return t('{{source}}的{{role}}被限制访问全部资源。', {
        source,
        role,
      });
    case 'source_resource':
      return t('{{source}}的{{role}}被限制访问{{resource}}。', {
        source,
        role,
        resource,
      });
    case 'china_mainland_home':
      return t('当前 IP 被识别为中国大陆，游客或普通用户被限制访问官网首页。');
    case 'china_mainland_sensitive':
      if (reason.resource) {
        return t('当前 IP 被识别为中国大陆，{{role}}被限制访问{{resource}}。', {
          role,
          resource,
        });
      }
      return t(
        '当前 IP 被识别为中国大陆，普通用户被限制访问令牌、钱包或账单资源。',
      );
    case 'resource':
      return t('不区分来源时，{{role}}被限制访问{{resource}}。', {
        role,
        resource,
      });
    case 'identity_guests':
      return t('游客访问已被全局身份策略限制。');
    case 'identity_users':
      return t('普通用户访问已被全局身份策略限制。');
    case 'identity_admins':
      return t('管理员访问已被全局身份策略限制。');
    default:
      return t('当前请求命中了访问限制策略。');
  }
}

const Forbidden = ({ accessLimited = false, accessReason = null }) => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const accessControl = statusState?.status?.access_control;
  const currentIp = accessControl?.request_ip || t('未知');
  const countryLabel =
    accessControl?.request_country_label ||
    accessControl?.request_country_code ||
    t('未知');
  const accessDetails = [
    [t('命中来源'), accessSourceLabel(t, accessReason?.source)],
    [t('命中资源'), accessResourceLabel(t, accessReason?.resource)],
    [t('命中角色'), accessRoleLabel(t, accessReason?.role)],
    [t('策略范围'), accessScopeLabel(t, accessReason?.scope)],
    [t('您当前 IP'), currentIp],
    [t('IP 归属地'), countryLabel],
  ];

  return (
    <div className='classic-page-fill flex justify-center items-center p-8'>
      <Empty
        image={<IllustrationNoAccess style={{ width: 250, height: 250 }} />}
        darkModeImage={
          <IllustrationNoAccessDark style={{ width: 250, height: 250 }} />
        }
        description={
          accessLimited ? (
            <div className='flex max-w-xl flex-col items-center gap-3 text-center'>
              <Text strong>{t('访问请求已被策略拦截。')}</Text>
              <Text type='secondary'>
                {accessDeniedReasonText(t, accessReason)}
              </Text>
              <div className='grid w-full max-w-md grid-cols-[max-content_minmax(0,1fr)] gap-x-3 gap-y-1 text-left'>
                {accessDetails.map(([label, value]) => (
                  <React.Fragment key={label}>
                    <Text type='tertiary'>{label}：</Text>
                    <Text strong className='min-w-0 break-words'>
                      {value}
                    </Text>
                  </React.Fragment>
                ))}
              </div>
            </div>
          ) : (
            t('您无权访问此页面，请联系管理员')
          )
        }
      />
    </div>
  );
};

Forbidden.propTypes = {
  accessLimited: PropTypes.bool,
  accessReason: PropTypes.shape({
    kind: PropTypes.string,
    source: PropTypes.string,
    resource: PropTypes.string,
    role: PropTypes.string,
    scope: PropTypes.string,
  }),
};

export default Forbidden;
