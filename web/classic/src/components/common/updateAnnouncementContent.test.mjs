import assert from 'node:assert/strict';
import {
  buildFrameHtml,
  renderMarkdownHtml,
  shouldRenderFrame,
} from './updateAnnouncementContent.js';

const markdownHtml = renderMarkdownHtml(
  '# StuHelperAPI User Group\nQQ 群号：1060573532',
);
assert.match(markdownHtml, /<h1[^>]*>StuHelperAPI User Group<\/h1>/);
assert.match(markdownHtml, /<p>QQ 群号：1060573532<\/p>/);
assert.equal(
  shouldRenderFrame('# StuHelperAPI User Group\nQQ 群号：1060573532'),
  false,
);

const htmlFragment = '<section><strong>HTML 片段</strong></section>';
assert.match(renderMarkdownHtml(htmlFragment), /<section>/);
assert.match(renderMarkdownHtml(htmlFragment), /<strong>HTML 片段<\/strong>/);
assert.equal(shouldRenderFrame(htmlFragment), false);

const fullHtml =
  '<!DOCTYPE html><html><head><style>body{}</style></head></html>';
assert.equal(shouldRenderFrame(fullHtml), true);
assert.equal(buildFrameHtml(fullHtml), fullHtml);

console.log('updateAnnouncementContent tests passed');
