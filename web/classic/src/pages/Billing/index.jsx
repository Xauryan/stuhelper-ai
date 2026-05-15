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

import React, { useState } from 'react';
import { Card, TabPane, Tabs, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import TopupBillingTable from '../../components/topup/modals/TopupBillingTable';

const { Text, Title } = Typography;

const Billing = () => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('all');

  return (
    <main className='p-2 md:p-4'>
      <Card bodyStyle={{ padding: 24 }}>
        <div className='mb-5'>
          <Title heading={3} className='!mb-2'>
            {t('账单管理')}
          </Title>
          <Text type='secondary'>{t('查看充值、订阅和退款记录')}</Text>
        </div>
        <Tabs type='button' activeKey={activeTab} onChange={setActiveTab}>
          <TabPane tab={t('全部账单')} itemKey='all'>
            <TopupBillingTable key='all' active={activeTab === 'all'} t={t} />
          </TabPane>
          <TabPane tab={t('待处理退款')} itemKey='pending_refund'>
            <TopupBillingTable
              key='pending_refund'
              active={activeTab === 'pending_refund'}
              pendingRefundOnly
              t={t}
            />
          </TabPane>
        </Tabs>
      </Card>
    </main>
  );
};

export default Billing;
