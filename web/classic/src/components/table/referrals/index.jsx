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
import CardPro from '../../common/ui/CardPro';
import ReferralsDescription from './ReferralsDescription';
import ReferralsFilters from './ReferralsFilters';
import ReferralsTable from './ReferralsTable';
import { useReferralsData } from '../../../hooks/referrals/useReferralsData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

const ReferralsPage = () => {
  const referralsData = useReferralsData();
  const isMobile = useIsMobile();

  return (
    <CardPro
      type='type1'
      descriptionArea={
        <ReferralsDescription
          compactMode={referralsData.compactMode}
          setCompactMode={referralsData.setCompactMode}
          t={referralsData.t}
        />
      }
      actionsArea={
        <div className='flex flex-col md:flex-row justify-end items-center gap-2 w-full'>
          <ReferralsFilters
            formInitValues={referralsData.formInitValues}
            setFormApi={referralsData.setFormApi}
            searchReferrals={referralsData.searchReferrals}
            resetFilters={referralsData.resetFilters}
            pageSize={referralsData.pageSize}
            loading={referralsData.loading}
            searching={referralsData.searching}
            t={referralsData.t}
          />
        </div>
      }
      paginationArea={createCardProPagination({
        currentPage: referralsData.activePage,
        pageSize: referralsData.pageSize,
        total: referralsData.recordCount,
        onPageChange: referralsData.handlePageChange,
        onPageSizeChange: referralsData.handlePageSizeChange,
        isMobile,
        t: referralsData.t,
      })}
      t={referralsData.t}
    >
      <ReferralsTable {...referralsData} />
    </CardPro>
  );
};

export default ReferralsPage;
