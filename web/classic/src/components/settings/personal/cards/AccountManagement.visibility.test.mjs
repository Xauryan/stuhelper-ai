import assert from 'node:assert/strict';
import { getVisibleAccountBindingKeys } from './AccountManagement.visibility.js';

const visibleKeys = getVisibleAccountBindingKeys({
  wechat_login: false,
  github_oauth: false,
  discord_oauth: false,
  oidc_enabled: false,
  telegram_oauth: false,
  linuxdo_oauth: false,
  custom_oauth_providers: [{ id: 1, slug: 'custom-one', name: 'Custom One' }],
});

assert.deepEqual(visibleKeys, ['email', 'custom:custom-one']);

assert.deepEqual(
  getVisibleAccountBindingKeys({
    wechat_login: true,
    github_oauth: true,
    discord_oauth: true,
    oidc_enabled: true,
    telegram_oauth: true,
    linuxdo_oauth: true,
    custom_oauth_providers: [],
  }),
  ['email', 'wechat', 'github', 'discord', 'oidc', 'telegram', 'linuxdo'],
);
