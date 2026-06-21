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
import { Button, ButtonGroup, Tooltip } from '@douyinfe/semi-ui';
import { Grid2X2, Table2 } from 'lucide-react';
import { TABLE_VIEW_MODES } from '../../../hooks/common/useTableViewMode';

const TableViewModeToggle = ({ viewMode, setViewMode, t }) => {
  return (
    <ButtonGroup size='small' className='w-full md:w-auto'>
      <Tooltip content={t('表格视图')}>
        <Button
          type={viewMode === TABLE_VIEW_MODES.TABLE ? 'primary' : 'tertiary'}
          theme={viewMode === TABLE_VIEW_MODES.TABLE ? 'solid' : 'borderless'}
          icon={<Table2 size={14} />}
          aria-label={t('表格视图')}
          aria-pressed={viewMode === TABLE_VIEW_MODES.TABLE}
          onClick={() => setViewMode(TABLE_VIEW_MODES.TABLE)}
        />
      </Tooltip>
      <Tooltip content={t('卡片视图')}>
        <Button
          type={viewMode === TABLE_VIEW_MODES.CARD ? 'primary' : 'tertiary'}
          theme={viewMode === TABLE_VIEW_MODES.CARD ? 'solid' : 'borderless'}
          icon={<Grid2X2 size={14} />}
          aria-label={t('卡片视图')}
          aria-pressed={viewMode === TABLE_VIEW_MODES.CARD}
          onClick={() => setViewMode(TABLE_VIEW_MODES.CARD)}
        />
      </Tooltip>
    </ButtonGroup>
  );
};

TableViewModeToggle.propTypes = {
  viewMode: PropTypes.oneOf(Object.values(TABLE_VIEW_MODES)).isRequired,
  setViewMode: PropTypes.func.isRequired,
  t: PropTypes.func.isRequired,
};

export default TableViewModeToggle;
