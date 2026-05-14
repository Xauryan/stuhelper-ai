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
