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

import { useCallback, useEffect, useState } from 'react';
import { getTableViewMode, setTableViewMode } from '../../helpers';
import { TABLE_VIEW_MODES_KEY } from '../../constants';

export const TABLE_VIEW_MODES = {
  TABLE: 'table',
  CARD: 'card',
};

export function useTableViewMode(tableKey = 'global', fallback = 'table') {
  const [viewMode, setViewModeState] = useState(() =>
    getTableViewMode(tableKey, fallback),
  );

  const setViewMode = useCallback(
    (mode) => {
      setViewModeState(mode);
      setTableViewMode(mode, tableKey);
    },
    [tableKey],
  );

  useEffect(() => {
    const handleStorage = (event) => {
      if (event.key !== TABLE_VIEW_MODES_KEY) {
        return;
      }

      setViewModeState(getTableViewMode(tableKey, fallback));
    };

    window.addEventListener('storage', handleStorage);
    return () => window.removeEventListener('storage', handleStorage);
  }, [fallback, tableKey]);

  return [viewMode, setViewMode];
}
