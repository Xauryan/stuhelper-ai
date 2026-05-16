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
import { Button, Form } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';

const BillingFilters = ({
  formApi,
  handleReset,
  handleSearch,
  handleViewChange,
  setFormApi,
  t,
}) => {
  return (
    <Form
      initValues={{ keyword: '', billingView: 'all' }}
      getFormApi={(api) => setFormApi(api)}
      onSubmit={handleSearch}
      allowEmpty={true}
      autoComplete='off'
      layout='vertical'
      trigger='change'
      stopValidateWithError={false}
    >
      <div className='flex flex-col gap-2'>
        <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-2'>
          <div className='col-span-1 lg:col-span-2'>
            <Form.Input
              field='keyword'
              className='w-full'
              prefix={<IconSearch />}
              placeholder={t('用户ID/用户名/订单号')}
              showClear
              pure
              size='small'
            />
          </div>
        </div>

        <div className='flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3'>
          <div className='w-full sm:w-auto'>
            <Form.Select
              field='billingView'
              placeholder={t('账单视图')}
              className='w-full sm:w-auto min-w-[120px]'
              pure
              onChange={(value) => {
                setTimeout(() => {
                  handleViewChange(value || 'all');
                }, 0);
              }}
              size='small'
            >
              <Form.Select.Option value='all'>
                {t('全部账单')}
              </Form.Select.Option>
              <Form.Select.Option value='pending_refund'>
                {t('待处理退款')}
              </Form.Select.Option>
            </Form.Select>
          </div>

          <div className='flex gap-2 w-full sm:w-auto justify-end'>
            <Button type='tertiary' htmlType='submit' size='small'>
              {t('查询')}
            </Button>
            <Button
              type='tertiary'
              onClick={() => {
                if (formApi) {
                  handleReset();
                }
              }}
              size='small'
            >
              {t('重置')}
            </Button>
          </div>
        </div>
      </div>
    </Form>
  );
};

export default BillingFilters;
