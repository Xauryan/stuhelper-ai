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
import { Button, Form, Radio } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';

import { DATE_RANGE_PRESETS } from '../../../constants/console.constants';
import { TOPUP_PAYMENT_METHODS } from '../../topup/modals/topupHistoryUtils.mjs';

const BillingFilters = ({
  formApi,
  formInitValues,
  handleReset,
  handleSearch,
  handleViewChange,
  isAdminUser,
  setFormApi,
  t,
}) => {
  return (
    <Form
      initValues={formInitValues}
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
            <Form.DatePicker
              field='dateRange'
              className='w-full'
              type='dateTimeRange'
              placeholder={[t('开始时间'), t('结束时间')]}
              showClear
              pure
              size='small'
              presets={DATE_RANGE_PRESETS.map((preset) => ({
                text: t(preset.text),
                start: preset.start(),
                end: preset.end(),
              }))}
            />
          </div>

          {isAdminUser && (
            <>
              <Form.Input
                field='user_id'
                prefix={<IconSearch />}
                placeholder={t('用户ID')}
                showClear
                pure
                size='small'
              />
              <Form.Input
                field='username'
                prefix={<IconSearch />}
                placeholder={t('用户名称')}
                showClear
                pure
                size='small'
              />
            </>
          )}

          <Form.Input
            field='trade_no'
            prefix={<IconSearch />}
            placeholder={t('订单号')}
            showClear
            pure
            size='small'
          />
        </div>

        <div>
          <Form.RadioGroup
            field='payment_method'
            label={t('支付方式')}
            type='button'
            buttonSize='small'
            direction='horizontal'
            aria-label={t('支付方式')}
            className='w-full flex flex-wrap gap-2'
            onChange={() => {
              setTimeout(() => {
                handleSearch(formApi?.getValues?.() || {});
              }, 0);
            }}
          >
            {TOPUP_PAYMENT_METHODS.map((method) => (
              <Radio value={method.value} key={method.value || 'all'}>
                {t(method.key)}
              </Radio>
            ))}
          </Form.RadioGroup>
        </div>

        <div className='flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3'>
          <div className='w-full sm:w-auto'>
            <Form.RadioGroup
              field='billingView'
              label={t('账单视图')}
              type='button'
              buttonSize='small'
              direction='horizontal'
              aria-label={t('账单视图')}
              className='flex flex-wrap gap-2'
              onChange={() => {
                setTimeout(() => {
                  const values = formApi?.getValues?.() || {};
                  handleViewChange(values.billingView || 'all');
                }, 0);
              }}
            >
              <Radio value='all'>{t('全部账单')}</Radio>
              <Radio value='pending_refund'>{t('待处理退款')}</Radio>
            </Form.RadioGroup>
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
