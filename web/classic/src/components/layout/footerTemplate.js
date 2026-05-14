export const FOOTER_TEMPLATE_DEFAULTS = Object.freeze({
  copyrightYear: '2026',
  copyrightOwner: 'StuHelper AI.',
  icpBeianUrl: 'https://beian.miit.gov.cn/',
  telecomLicenseUrl: 'https://tsm.miit.gov.cn/',
});

export const TELECOM_LICENSE_TYPES = Object.freeze({
  icp: 'ICP',
  edi: 'EDI',
});

const escapeHtml = (value) =>
  String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');

const normalizeText = (value) => String(value ?? '').trim();

const normalizeUrl = (value, fallback) => normalizeText(value) || fallback;

const normalizeLicenseTypes = (value) => {
  const rawTypes = Array.isArray(value)
    ? value
    : normalizeText(value)
        .split(',')
        .map((item) => item.trim());
  const normalized = [];
  for (const type of rawTypes) {
    const upperType = String(type ?? '')
      .trim()
      .toUpperCase();
    if (
      (upperType === TELECOM_LICENSE_TYPES.icp ||
        upperType === TELECOM_LICENSE_TYPES.edi) &&
      !normalized.includes(upperType)
    ) {
      normalized.push(upperType);
    }
  }
  return normalized;
};

export const joinFooterLicenseTypes = (value) =>
  [TELECOM_LICENSE_TYPES.icp, TELECOM_LICENSE_TYPES.edi]
    .filter((type) => normalizeLicenseTypes(value).includes(type))
    .join(',');

export const parseFooterLicenseTypes = normalizeLicenseTypes;

export const buildFooterCopyrightText = ({
  copyrightYear,
  copyrightOwner,
} = {}) => {
  const year = normalizeText(copyrightYear);
  const owner = normalizeText(copyrightOwner);
  if (!year || !owner) {
    return '';
  }
  return `© ${year} ${owner} 版权所有`;
};

export const buildFooterTemplateHTML = ({
  icpBeianNumber = '',
  icpBeianUrl = FOOTER_TEMPLATE_DEFAULTS.icpBeianUrl,
  telecomLicenseNumber = '',
  telecomLicenseUrl = FOOTER_TEMPLATE_DEFAULTS.telecomLicenseUrl,
  telecomLicenseTypes = '',
  copyrightYear = FOOTER_TEMPLATE_DEFAULTS.copyrightYear,
  copyrightOwner = FOOTER_TEMPLATE_DEFAULTS.copyrightOwner,
} = {}) => {
  const items = [];
  const icpNumber = normalizeText(icpBeianNumber);
  if (icpNumber) {
    items.push(
      `<a href="${escapeHtml(normalizeUrl(icpBeianUrl, FOOTER_TEMPLATE_DEFAULTS.icpBeianUrl))}" target="_blank" rel="noopener noreferrer">${escapeHtml(icpNumber)}</a>`,
    );
  }

  const licenseNumber = normalizeText(telecomLicenseNumber);
  const licenseTypes = normalizeLicenseTypes(telecomLicenseTypes);
  if (licenseNumber && licenseTypes.length > 0) {
    items.push(
      `<a href="${escapeHtml(normalizeUrl(telecomLicenseUrl, FOOTER_TEMPLATE_DEFAULTS.telecomLicenseUrl))}" target="_blank" rel="noopener noreferrer">增值电信业务经营许可证(${escapeHtml(licenseTypes.join(','))})：${escapeHtml(licenseNumber)}</a>`,
    );
  }

  const copyrightText = buildFooterCopyrightText({
    copyrightYear,
    copyrightOwner,
  });
  if (copyrightText) {
    items.push(
      `<span class="stuhelper-copyright">${escapeHtml(copyrightText)}</span>`,
    );
  }

  if (items.length === 0) {
    return '';
  }

  return `<footer class="stuhelper-footer"><div class="stuhelper-footer-content">${items.join('<span class="stuhelper-footer-separator">|</span>')}</div></footer>`;
};

export const hasFooterTemplateConfig = (config = {}) =>
  Boolean(
    normalizeText(config.icpBeianNumber) ||
      normalizeText(config.telecomLicenseNumber) ||
      normalizeText(config.copyrightYear) ||
      normalizeText(config.copyrightOwner),
  );
