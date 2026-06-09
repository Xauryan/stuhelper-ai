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
import { useTranslation } from 'react-i18next';
import CardPro from '../../common/ui/CardPro';
import TopupBillingTable from '../../topup/modals/TopupBillingTable';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { useTableCompactMode } from '../../../hooks/common/useTableCompactMode';
import {
  isAdmin,
  getTodayStartTimestamp,
  timestamp2string,
} from '../../../helpers';
import { createCardProPagination } from '../../../helpers/utils';
import BillingActions from './BillingActions';
import BillingFilters from './BillingFilters';

const BillingTable = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [activeTab, setActiveTab] = useState('all');
  const [compactMode, setCompactMode] = useTableCompactMode('billing');
  const [submittedFilters, setSubmittedFilters] = useState(() =>
    getDefaultBillingFilters(),
  );
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0,
    totalMoney: 0,
  });
  const [tableApi, setTableApi] = useState(null);
  const [formApi, setFormApi] = useState(null);
  const userIsAdmin = isAdmin();
  const [formInitValues] = useState(() => getDefaultBillingFilters());

  const handlePaginationChange = useCallback((nextPagination) => {
    setPagination((previousPagination) => {
      if (
        previousPagination.page === nextPagination.page &&
        previousPagination.pageSize === nextPagination.pageSize &&
        previousPagination.total === nextPagination.total &&
        previousPagination.totalMoney === nextPagination.totalMoney
      ) {
        return previousPagination;
      }
      return nextPagination;
    });
  }, []);

  const handleSearch = (values = {}) => {
    setSubmittedFilters(normalizeBillingFilters(values));
    tableApi?.setPage(1);
  };

  const handleReset = () => {
    const defaultFilters = getDefaultBillingFilters();
    formApi?.reset();
    formApi?.setValues?.(defaultFilters);
    setSubmittedFilters(defaultFilters);
    setActiveTab('all');
    tableApi?.setPage(1);
  };

  const handleViewChange = (key) => {
    setActiveTab(key);
    tableApi?.setPage(1);
  };

  return (
    <CardPro
      type='type2'
      statsArea={
        <BillingActions
          activeTab={activeTab}
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          total={pagination.total}
          totalMoney={pagination.totalMoney}
          t={t}
        />
      }
      searchArea={
        <BillingFilters
          formApi={formApi}
          formInitValues={formInitValues}
          handleReset={handleReset}
          handleSearch={handleSearch}
          handleViewChange={handleViewChange}
          isAdminUser={userIsAdmin}
          setFormApi={setFormApi}
          t={t}
        />
      }
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
      <TopupBillingTable
        active
        compactMode={compactMode}
        externalFilters={submittedFilters}
        hideFilters
        hidePagination
        onPaginationChange={handlePaginationChange}
        onReady={setTableApi}
        pendingRefundOnly={activeTab === 'pending_refund'}
        pendingSelfServeAuditOnly={activeTab === 'pending_self_serve'}
        t={t}
        variant='page'
      />
    </CardPro>
  );
};

const getDefaultBillingFilters = () => {
  const now = new Date();
  return {
    user_id: '',
    username: '',
    trade_no: '',
    payment_method: '',
    billingView: 'all',
    dateRange: [
      timestamp2string(getTodayStartTimestamp()),
      timestamp2string(now.getTime() / 1000),
    ],
  };
};

const normalizeBillingFilters = (values = {}) => ({
  user_id: (values.user_id || '').trim(),
  username: (values.username || '').trim(),
  trade_no: (values.trade_no || '').trim(),
  payment_method: values.payment_method || '',
  billingView: values.billingView || 'all',
  dateRange: values.dateRange,
});

export default BillingTable;
