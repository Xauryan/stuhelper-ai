import assert from 'node:assert/strict';
import {
  buildFooterTemplateHTML,
  buildFooterCopyrightText,
  formatFooterLicenseName,
  joinFooterLicenseTypes,
  parseFooterLicenseTypes,
} from './footerTemplate.js';

assert.equal(
  buildFooterCopyrightText({
    copyrightYear: '2025-2026',
    copyrightOwner: 'StuHelper AI.',
  }),
  '© 2025-2026 StuHelper AI. 版权所有',
);

assert.deepEqual(parseFooterLicenseTypes('icp,edi,ICP,unknown'), [
  'ICP',
  'EDI',
]);
assert.equal(joinFooterLicenseTypes(['ICP', 'EDI']), 'ICP,EDI');
assert.equal(joinFooterLicenseTypes(['EDI', 'ICP']), 'ICP,EDI');
assert.equal(
  formatFooterLicenseName(['ICP', 'EDI']),
  '互联网信息服务业务经营许可证、增值电信业务经营许可证—在线数据处理与交易处理业务',
);

const fullFooter = buildFooterTemplateHTML({
  icpBeianNumber: '京ICP备2025154912号',
  telecomLicenseNumber: '京B2-20253869',
  telecomLicenseTypes: ['ICP', 'EDI'],
  copyrightYear: '2026',
  copyrightOwner: 'StuHelper AI.',
});

assert.match(fullFooter, /https:\/\/beian\.miit\.gov\.cn\//);
assert.match(fullFooter, /京ICP备2025154912号/);
assert.match(fullFooter, /https:\/\/tsm\.miit\.gov\.cn\//);
assert.match(
  fullFooter,
  /互联网信息服务业务经营许可证、增值电信业务经营许可证—在线数据处理与交易处理业务：京B2-20253869/,
);
assert.match(fullFooter, /© 2026 StuHelper AI\. 版权所有/);
assert.equal((fullFooter.match(/stuhelper-footer-separator/g) || []).length, 2);

const partialFooter = buildFooterTemplateHTML({
  icpBeianNumber: '',
  telecomLicenseNumber: '京B2-20253869',
  telecomLicenseTypes: ['ICP'],
  copyrightYear: '',
  copyrightOwner: 'StuHelper AI.',
});

assert.doesNotMatch(partialFooter, /beian\.miit/);
assert.match(partialFooter, /互联网信息服务业务经营许可证：京B2-20253869/);
assert.doesNotMatch(partialFooter, /版权所有/);
assert.equal(
  (partialFooter.match(/stuhelper-footer-separator/g) || []).length,
  0,
);

const escapedFooter = buildFooterTemplateHTML({
  icpBeianNumber: '<script>alert(1)</script>',
  telecomLicenseNumber: '',
  copyrightYear: '2026',
  copyrightOwner: '<StuHelper>',
});

assert.match(escapedFooter, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/);
assert.match(escapedFooter, /&lt;StuHelper&gt;/);
assert.doesNotMatch(escapedFooter, /<script>/);
