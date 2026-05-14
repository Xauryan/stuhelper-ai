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

export function getBusinessLogExpandedDetailText(record) {
  if (record?.type !== 6 || typeof record?.content !== 'string') {
    return null;
  }

  const content = record.content.trim();
  return content === '' ? null : content;
}
