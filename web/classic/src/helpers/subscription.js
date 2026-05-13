export function getSubscriptionModelLimits(plan) {
  if (!plan?.model_limits_enabled || !plan?.model_limits) {
    return [];
  }
  const seen = new Set();
  return plan.model_limits
    .split(',')
    .map((item) => item.trim())
    .filter((item) => {
      if (!item || seen.has(item)) {
        return false;
      }
      seen.add(item);
      return true;
    });
}

export function getSubscriptionModelLimitsCsv(models) {
  const seen = new Set();
  return (models || [])
    .map((item) => String(item || '').trim())
    .filter((item) => {
      if (!item || seen.has(item)) {
        return false;
      }
      seen.add(item);
      return true;
    })
    .join(',');
}
