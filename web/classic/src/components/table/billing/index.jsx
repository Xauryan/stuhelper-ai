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
import { createCardProPagination } from '../../../helpers/utils';
import BillingActions from './BillingActions';
import BillingFilters from './BillingFilters';

const BillingTable = () => {
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
          t={t}
        />
      }
      searchArea={
        <BillingFilters
          formApi={formApi}
          handleReset={handleReset}
          handleSearch={handleSearch}
          handleViewChange={handleViewChange}
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

export default BillingTable;
