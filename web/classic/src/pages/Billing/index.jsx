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

import React, { useCallback, useState } from 'react';
import {
  Button,
  Form,
  TabPane,
  Tabs,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import CardPro from '../../components/common/ui/CardPro';
import CompactModeToggle from '../../components/common/ui/CompactModeToggle';
import TopupBillingTable from '../../components/topup/modals/TopupBillingTable';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useTableCompactMode } from '../../hooks/common/useTableCompactMode';
import { createCardProPagination } from '../../helpers/utils';

const { Text } = Typography;

const Billing = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [activeTab, setActiveTab] = useState('all');
  const [compactMode, setCompactMode] = useTableCompactMode('billing');
  const [submittedKeyword, setSubmittedKeyword] = useState('');
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0,
  });
  const [tableApi, setTableApi] = useState(null);
  const [formApi, setFormApi] = useState(null);

  const handlePaginationChange = useCallback((nextPagination) => {
    setPagination((previousPagination) => {
      if (
        previousPagination.page === nextPagination.page &&
        previousPagination.pageSize === nextPagination.pageSize &&
        previousPagination.total === nextPagination.total
      ) {
        return previousPagination;
      }
      return nextPagination;
    });
  }, []);

  const handleSearch = (values = {}) => {
    setSubmittedKeyword((values.keyword || '').trim());
    tableApi?.setPage(1);
  };

  const handleReset = () => {
    formApi?.reset();
    setSubmittedKeyword('');
    tableApi?.setPage(1);
  };

  const handleTabChange = (key) => {
    setActiveTab(key);
    tableApi?.setPage(1);
  };

  const statsArea = (
    <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full'>
      <div className='flex flex-wrap gap-2'>
        <Tag
          color='blue'
          style={{
            fontWeight: 500,
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
            padding: 13,
          }}
          className='!rounded-lg'
        >
          {t('账单管理')}
        </Tag>
        <Tag
          color={activeTab === 'pending_refund' ? 'orange' : 'white'}
          style={{
            border: 'none',
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
            fontWeight: 500,
            padding: 13,
          }}
          className='!rounded-lg'
        >
          {activeTab === 'pending_refund' ? t('待处理退款') : t('全部账单')}
        </Tag>
        <Tag
          color='white'
          style={{
            border: 'none',
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
            fontWeight: 500,
            padding: 13,
          }}
          className='!rounded-lg'
        >
          {t('账单')}: {pagination.total}
        </Tag>
      </div>
      <CompactModeToggle
        compactMode={compactMode}
        setCompactMode={setCompactMode}
        t={t}
      />
    </div>
  );

  const searchArea = (
    <Form
      getFormApi={setFormApi}
      initValues={{ keyword: '' }}
      onSubmit={handleSearch}
      allowEmpty
      autoComplete='off'
      layout='vertical'
      trigger='change'
      stopValidateWithError={false}
    >
      <div className='flex flex-col gap-2'>
        <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-2'>
          <Form.Input
            field='keyword'
            prefix={<IconSearch />}
            placeholder={t('用户ID/用户名/订单号')}
            showClear
            pure
            size='small'
          />
        </div>
        <div className='flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3'>
          <Tabs type='button' activeKey={activeTab} onChange={handleTabChange}>
            <TabPane tab={t('全部账单')} itemKey='all' />
            <TabPane tab={t('待处理退款')} itemKey='pending_refund' />
          </Tabs>
          <div className='flex gap-2 w-full sm:w-auto justify-end'>
            <Button type='tertiary' htmlType='submit' size='small'>
              {t('查询')}
            </Button>
            <Button type='tertiary' onClick={handleReset} size='small'>
              {t('重置')}
            </Button>
          </div>
        </div>
      </div>
    </Form>
  );

  return (
    <CardPro
      type='type2'
      statsArea={statsArea}
      searchArea={searchArea}
      paginationArea={createCardProPagination({
        currentPage: pagination.page,
        pageSize: pagination.pageSize,
        total: pagination.total,
        onPageChange: tableApi?.setPage,
        onPageSizeChange: tableApi?.setPageSize,
        isMobile,
        t,
      })}
      t={t}
    >
      <div className='mb-3 md:hidden'>
        <Text type='secondary'>{t('查看充值、订阅和退款记录')}</Text>
      </div>
      <TopupBillingTable
        key={activeTab}
        active
        compactMode={compactMode}
        externalKeyword={submittedKeyword}
        hideFilters
        hidePagination
        onPaginationChange={handlePaginationChange}
        onReady={setTableApi}
        pendingRefundOnly={activeTab === 'pending_refund'}
        t={t}
        variant='page'
      />
    </CardPro>
  );
};

export default Billing;
