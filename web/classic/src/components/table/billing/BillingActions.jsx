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
import { Space, Tag } from '@douyinfe/semi-ui';
import CompactModeToggle from '../../common/ui/CompactModeToggle';
import { formatCurrency } from '../../topup/modals/topupHistoryUtils.mjs';

const tagBaseStyle = {
  fontWeight: 500,
  boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
  padding: 13,
};

const BillingActions = ({
  activeTab,
  compactMode,
  setCompactMode,
  total,
  totalMoney,
  t,
}) => {
  return (
    <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full'>
      <Space>
        <Tag color='blue' style={tagBaseStyle} className='!rounded-lg'>
          {t('支付成功金额')}: {formatCurrency(totalMoney)}
          {t('元')}
        </Tag>
        <Tag color='pink' style={tagBaseStyle} className='!rounded-lg'>
          {activeTab === 'pending_refund'
            ? t('待处理退款')
            : activeTab === 'pending_self_serve'
              ? t('待审核自助充值')
              : `${t('账单')}: ${total}`}
        </Tag>
      </Space>

      <CompactModeToggle
        compactMode={compactMode}
        setCompactMode={setCompactMode}
        t={t}
      />
    </div>
  );
};

export default BillingActions;
