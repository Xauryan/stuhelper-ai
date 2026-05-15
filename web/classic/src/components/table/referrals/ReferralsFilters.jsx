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

import React, { useRef } from 'react';
import { Button, Form } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import { REFERRAL_REWARD_STATUS } from '../../../hooks/referrals/useReferralsData';

const ReferralsFilters = ({
  formInitValues,
  setFormApi,
  searchReferrals,
  resetFilters,
  pageSize,
  loading,
  searching,
  t,
}) => {
  const formApiRef = useRef(null);

  const rewardStatusOptions = [
    { label: t('全部奖励状态'), value: REFERRAL_REWARD_STATUS.ALL },
    { label: t('已解锁'), value: REFERRAL_REWARD_STATUS.UNLOCKED },
    { label: t('待首充解锁'), value: REFERRAL_REWARD_STATUS.PENDING },
  ];

  return (
    <Form
      initValues={formInitValues}
      getFormApi={(api) => {
        formApiRef.current = api;
        setFormApi(api);
      }}
      onSubmit={() => searchReferrals(1, pageSize)}
      allowEmpty
      autoComplete='off'
      layout='horizontal'
      trigger='change'
      stopValidateWithError={false}
      className='w-full md:w-auto order-1 md:order-2'
    >
      <div className='flex flex-col md:flex-row items-center gap-2 w-full md:w-auto'>
        <div className='relative w-full md:w-64'>
          <Form.Input
            field='searchKeyword'
            prefix={<IconSearch />}
            placeholder={t('搜索邀请人或被邀请用户')}
            showClear
            pure
            size='small'
          />
        </div>
        <div className='w-full md:w-48'>
          <Form.Select
            field='rewardStatus'
            optionList={rewardStatusOptions}
            placeholder={t('全部奖励状态')}
            className='w-full'
            pure
            size='small'
            onChange={() => {
              setTimeout(() => searchReferrals(1, pageSize), 100);
            }}
          />
        </div>
        <div className='flex gap-2 w-full md:w-auto'>
          <Button
            type='tertiary'
            htmlType='submit'
            loading={loading || searching}
            className='flex-1 md:flex-initial md:w-auto'
            size='small'
          >
            {t('查询')}
          </Button>
          <Button
            type='tertiary'
            onClick={() => {
              formApiRef.current?.reset();
              resetFilters();
            }}
            className='flex-1 md:flex-initial md:w-auto'
            size='small'
          >
            {t('重置')}
          </Button>
        </div>
      </div>
    </Form>
  );
};

export default ReferralsFilters;
