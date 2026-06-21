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

import { useCallback, useState } from 'react';

const DEFAULT_PAGE_SIZE_STORAGE_KEY = 'page-size';

const readPageSize = (storageKey, fallback) => {
  try {
    const value = Number.parseInt(localStorage.getItem(storageKey) || '', 10);
    return value > 0 ? value : fallback;
  } catch {
    return fallback;
  }
};

export function useTablePageSize(
  fallback,
  storageKey = DEFAULT_PAGE_SIZE_STORAGE_KEY,
) {
  const [pageSize, setPageSizeState] = useState(() =>
    readPageSize(storageKey, fallback),
  );

  const setPageSize = useCallback(
    (size) => {
      setPageSizeState(size);
      try {
        localStorage.setItem(storageKey, String(size));
      } catch {
        // ignore storage failures
      }
    },
    [storageKey],
  );

  return [pageSize, setPageSize];
}
