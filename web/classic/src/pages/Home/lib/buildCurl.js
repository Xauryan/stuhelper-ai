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

const normalizeBase = (raw) => {
  if (!raw) return 'https://api.example.com';
  return raw.replace(/\/+$/, '');
};

export const buildCurl = (serverAddress) => {
  const base = normalizeBase(serverAddress);
  return [
    `curl ${base}/v1/chat/completions \\`,
    `  -H "Authorization: Bearer $STUHELPER_API_KEY" \\`,
    `  -H "Content-Type: application/json" \\`,
    `  -d '{`,
    `    "model": "gpt-4o-mini",`,
    `    "messages": [{ "role": "user", "content": "Hello" }]`,
    `  }'`,
  ].join('\n');
};

export default buildCurl;
