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
import PropTypes from 'prop-types';
import { Card, Tag, Typography } from '@douyinfe/semi-ui';
import { timestamp2string } from '../../../helpers';

const { Text } = Typography;

const labelClass =
  'mb-1 text-xs font-medium text-semi-color-text-2 select-none';

const getColumn = (columns, key) =>
  columns.find((column) => column.key === key);

const renderColumnCell = ({ columns, key, record, index }) => {
  const column = getColumn(columns, key);
  if (!column) {
    return null;
  }

  const value = column.dataIndex ? record[column.dataIndex] : undefined;
  const content = column.render
    ? column.render(value, record, index)
    : value === undefined || value === null
      ? '-'
      : value;

  return content === undefined || content === null ? '-' : content;
};

const renderTestTime = (testTime, t) => {
  const timestamp = Number(testTime);
  if (!Number.isFinite(timestamp) || timestamp <= 0) {
    return (
      <Tag color='grey' shape='circle'>
        {t('未测试')}
      </Tag>
    );
  }

  const value = timestamp2string(timestamp);
  return (
    <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 116 }}>
      {value}
    </Text>
  );
};

const ChannelMobileCard = ({ record, index, columns, columnKeys, t }) => {
  const isTagRow = record.children !== undefined;
  const idCell = renderColumnCell({
    columns,
    key: columnKeys.ID,
    record,
    index,
  });
  const nameCell = renderColumnCell({
    columns,
    key: columnKeys.NAME,
    record,
    index,
  });
  const groupCell = renderColumnCell({
    columns,
    key: columnKeys.GROUP,
    record,
    index,
  });
  const typeCell = renderColumnCell({
    columns,
    key: columnKeys.TYPE,
    record,
    index,
  });
  const statusCell = renderColumnCell({
    columns,
    key: columnKeys.STATUS,
    record,
    index,
  });
  const balanceCell = renderColumnCell({
    columns,
    key: columnKeys.BALANCE,
    record,
    index,
  });
  const responseTimeCell = renderColumnCell({
    columns,
    key: columnKeys.RESPONSE_TIME,
    record,
    index,
  });
  const priorityCell = renderColumnCell({
    columns,
    key: columnKeys.PRIORITY,
    record,
    index,
  });
  const weightCell = renderColumnCell({
    columns,
    key: columnKeys.WEIGHT,
    record,
    index,
  });
  const operateCell = renderColumnCell({
    columns,
    key: columnKeys.OPERATE,
    record,
    index,
  });
  const showRightColumn =
    priorityCell !== null || weightCell !== null || responseTimeCell !== null;

  return (
    <Card className='!rounded-2xl shadow-sm'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='flex min-w-0 flex-1 flex-wrap items-center gap-2'>
          {typeCell}
          {statusCell}
        </div>
      </div>

      <div className='mt-3 flex items-start justify-between gap-3'>
        <div className='flex min-w-0 flex-1 flex-col gap-3 overflow-hidden'>
          <div className='min-w-0'>
            {!isTagRow && idCell !== null ? (
              <div className={labelClass}>#{idCell}</div>
            ) : null}
            <div className='min-w-0 text-sm font-medium'>{nameCell ?? '-'}</div>
          </div>

          {balanceCell !== null ? (
            <div className='min-w-0'>
              <div className={labelClass}>{t('已用/剩余')}</div>
              <div className='min-w-0 overflow-hidden text-sm'>
                {balanceCell}
              </div>
            </div>
          ) : null}
        </div>

        {showRightColumn ? (
          <div className='grid shrink-0 grid-cols-[max-content_minmax(0,auto)] items-center gap-x-2 gap-y-2 text-left'>
            {priorityCell !== null ? (
              <>
                <span className={labelClass}>{t('优先级')}</span>
                <div className='max-w-[74px] overflow-hidden'>
                  {priorityCell}
                </div>
              </>
            ) : null}

            {weightCell !== null ? (
              <>
                <span className={labelClass}>{t('权重')}</span>
                <div className='max-w-[74px] overflow-hidden'>{weightCell}</div>
              </>
            ) : null}

            {responseTimeCell !== null ? (
              <>
                <span className={labelClass}>{t('响应')}</span>
                <div className='max-w-[116px] overflow-hidden text-sm'>
                  {responseTimeCell}
                </div>
                <span className={labelClass}>{t('上次测试')}</span>
                <div className='max-w-[116px] overflow-hidden text-sm'>
                  {renderTestTime(record.test_time, t)}
                </div>
              </>
            ) : null}
          </div>
        ) : null}
      </div>

      {groupCell !== null ? (
        <div className='mt-3 min-w-0'>
          <div className={labelClass}>{t('分组')}</div>
          <div className='min-w-0'>{groupCell}</div>
        </div>
      ) : null}

      {operateCell !== null ? (
        <div className='mt-3 flex justify-end overflow-hidden'>
          <div className='max-w-full overflow-hidden'>{operateCell}</div>
        </div>
      ) : null}
    </Card>
  );
};

ChannelMobileCard.propTypes = {
  record: PropTypes.object.isRequired,
  index: PropTypes.number.isRequired,
  columns: PropTypes.array.isRequired,
  columnKeys: PropTypes.object.isRequired,
  t: PropTypes.func.isRequired,
};

export default ChannelMobileCard;
