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
export const RECHARGE_TAB_TOPUP = 'topup';
export const RECHARGE_TAB_SUBSCRIPTION = 'subscription';

export function getInitialRechargeTabKey() {
  return RECHARGE_TAB_TOPUP;
}

export function getRechargeTabKeys(shouldShowSubscription) {
  return shouldShowSubscription
    ? [RECHARGE_TAB_TOPUP, RECHARGE_TAB_SUBSCRIPTION]
    : [RECHARGE_TAB_TOPUP];
}

export function shouldShowSubscriptionTab({
  loading = false,
  plans = [],
  activeSubscriptions = [],
  allSubscriptions = [],
} = {}) {
  const hasSubscriptionRecords =
    (Array.isArray(activeSubscriptions) && activeSubscriptions.length > 0) ||
    (Array.isArray(allSubscriptions) && allSubscriptions.length > 0);
  if (hasSubscriptionRecords) {
    return true;
  }
  return !loading && Array.isArray(plans) && plans.length > 0;
}
