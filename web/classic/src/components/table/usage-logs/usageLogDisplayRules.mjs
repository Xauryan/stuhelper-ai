export function isModelBillingLog(record) {
  return record?.type === 0 || record?.type === 2 || record?.type === 5;
}

export function shouldShowLogIp(record, isAdminUser) {
  return Boolean(
    record?.ip &&
      (record?.type === 2 ||
        record?.type === 5 ||
        (isAdminUser && (record?.type === 1 || record?.type === 6))),
  );
}

export function getRefundLogDetailText(record, fallback) {
  return record?.type === 6 ? record?.content || fallback : null;
}
