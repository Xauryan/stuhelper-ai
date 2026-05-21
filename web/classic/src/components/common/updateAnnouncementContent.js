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

import { marked } from 'marked';

export const shouldRenderFrame = (raw) =>
  /<!doctype|<html[\s>]|<head[\s>]|<body[\s>]|<style[\s>]|<script[\s>]/i.test(
    String(raw || ''),
  );

export const buildFrameHtml = (raw) => {
  const source = String(raw || '');
  if (!source.trim()) {
    return '';
  }
  return source;
};

export const renderMarkdownHtml = (raw) => marked.parse(String(raw || ''));

export const getUpdateAnnouncementContent = (item) =>
  String(item?.content || '').trim() ||
  String(item?.extra || '').trim() ||
  String(item?.title || '').trim();
