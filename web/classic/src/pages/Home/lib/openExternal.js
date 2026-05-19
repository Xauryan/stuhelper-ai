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

const SAFE_PROTOCOLS = new Set(['http:', 'https:']);

const openExternal = (url) => {
  if (!url || typeof window === 'undefined') return;
  let parsed;
  try {
    parsed = new URL(url, window.location.origin);
  } catch {
    return;
  }
  if (!SAFE_PROTOCOLS.has(parsed.protocol)) return;
  const opened = window.open(parsed.href, '_blank', 'noopener,noreferrer');
  if (opened) opened.opener = null;
};

export default openExternal;
