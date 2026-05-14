export function shouldHighlightSubscriptionPlan(plan) {
  return Boolean(plan?.recommended);
}
