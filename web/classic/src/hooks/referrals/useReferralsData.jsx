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

import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';

export const REFERRAL_REWARD_STATUS = {
  ALL: '',
  UNLOCKED: 'unlocked',
  PENDING: 'pending',
};

export const useReferralsData = () => {
  const { t } = useTranslation();
  const [records, setRecords] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [recordCount, setRecordCount] = useState(0);
  const [formApi, setFormApi] = useState(null);
  const [compactMode, setCompactMode] = useTableCompactMode('referrals');

  const formInitValues = {
    searchKeyword: '',
    rewardStatus: REFERRAL_REWARD_STATUS.ALL,
  };

  const getFormValues = () => {
    const values = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: values.searchKeyword || '',
      rewardStatus: values.rewardStatus || REFERRAL_REWARD_STATUS.ALL,
    };
  };

  const buildQuery = (page, size) => {
    const { searchKeyword, rewardStatus } = getFormValues();
    const params = new URLSearchParams({
      p: String(page),
      page_size: String(size),
    });
    if (searchKeyword.trim() !== '') {
      params.set('keyword', searchKeyword.trim());
    }
    if (rewardStatus) {
      params.set('reward_status', rewardStatus);
    }
    return params.toString();
  };

  const loadReferrals = async (page = 1, size = pageSize) => {
    setLoading(true);
    try {
      const res = await API.get(
        `/api/user/referrals?${buildQuery(page, size)}`,
      );
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setRecords(data.items || []);
      setActivePage(data.page <= 0 ? 1 : data.page);
      setRecordCount(data.total || 0);
    } catch (error) {
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  const searchReferrals = async (page = 1, size = pageSize) => {
    setSearching(true);
    try {
      await loadReferrals(page, size);
    } finally {
      setSearching(false);
    }
  };

  const handlePageChange = (page) => {
    setActivePage(page);
    loadReferrals(page, pageSize);
  };

  const handlePageSizeChange = (size) => {
    setPageSize(size);
    setActivePage(1);
    loadReferrals(1, size);
  };

  const resetFilters = () => {
    formApi?.reset();
    setTimeout(() => loadReferrals(1, pageSize), 100);
  };

  useEffect(() => {
    loadReferrals(1, pageSize);
  }, []);

  return {
    records,
    loading,
    searching,
    activePage,
    pageSize,
    recordCount,
    formInitValues,
    setFormApi,
    searchReferrals,
    loadReferrals,
    resetFilters,
    handlePageChange,
    handlePageSizeChange,
    compactMode,
    setCompactMode,
    t,
  };
};
