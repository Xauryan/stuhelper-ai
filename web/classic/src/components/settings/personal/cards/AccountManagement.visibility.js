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

const builtInAccountBindingVisibility = [
  { key: 'email', enabled: () => true },
  { key: 'wechat', enabled: (status) => Boolean(status.wechat_login) },
  { key: 'github', enabled: (status) => Boolean(status.github_oauth) },
  { key: 'discord', enabled: (status) => Boolean(status.discord_oauth) },
  { key: 'oidc', enabled: (status) => Boolean(status.oidc_enabled) },
  { key: 'telegram', enabled: (status) => Boolean(status.telegram_oauth) },
  { key: 'linuxdo', enabled: (status) => Boolean(status.linuxdo_oauth) },
];

export const isAccountBindingVisible = (status = {}, key) => {
  const item = builtInAccountBindingVisibility.find(
    (binding) => binding.key === key,
  );
  return item ? item.enabled(status) : false;
};

export const getVisibleAccountBindingKeys = (status = {}) => {
  const visibleKeys = builtInAccountBindingVisibility
    .filter((binding) => binding.enabled(status))
    .map((binding) => binding.key);

  return visibleKeys.concat(
    (status.custom_oauth_providers || []).map(
      (provider) => `custom:${provider.slug || provider.id}`,
    ),
  );
};
