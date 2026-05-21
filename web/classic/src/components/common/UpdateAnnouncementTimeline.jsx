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

import React, { useMemo } from 'react';
import { Timeline } from '@douyinfe/semi-ui';
import { formatDateTimeString, getRelativeTime } from '../../helpers';
import {
  buildFrameHtml,
  getUpdateAnnouncementContent,
  renderMarkdownHtml,
  shouldRenderFrame,
} from './updateAnnouncementContent';

export {
  buildFrameHtml,
  getUpdateAnnouncementContent,
  renderMarkdownHtml,
  shouldRenderFrame,
} from './updateAnnouncementContent';

export const formatAbsoluteTime = (dateValue) => {
  const date = dateValue ? new Date(dateValue) : null;
  if (!date || isNaN(date.getTime())) {
    return dateValue || '';
  }
  return formatDateTimeString(date);
};

const joinDisplayTime = (relative, absolute) => {
  if (relative && absolute && relative !== absolute) {
    if (absolute.startsWith(`${relative} `)) {
      return absolute;
    }
    return `${relative} · ${absolute}`;
  }
  return relative || absolute || '';
};

export const formatDisplayTime = (dateValue) => {
  const relative = getRelativeTime(dateValue);
  const absolute = formatAbsoluteTime(dateValue);
  return joinDisplayTime(relative, absolute);
};

const getUpdateAnnouncementTime = (item) => {
  if (item?.publishDate) {
    return formatDisplayTime(item.publishDate);
  }

  return joinDisplayTime(
    String(item?.relative || '').trim(),
    String(item?.time || '').trim(),
  );
};

export const normalizeUpdateAnnouncementItems = (items) =>
  (items || []).map((item, index) => {
    const content = getUpdateAnnouncementContent(item);
    const time = getUpdateAnnouncementTime(item);

    return {
      ...item,
      id: item?.id || `update-announcement-${index}`,
      title: String(item?.title || '').trim(),
      content,
      usesFrame: shouldRenderFrame(content),
      frameHtml: buildFrameHtml(content),
      type: item?.type || (index === 0 ? 'success' : 'default'),
      time,
      displayTime: time,
      absoluteTime: item?.publishDate
        ? formatAbsoluteTime(item.publishDate)
        : '',
    };
  });

const UpdateAnnouncementTimeline = ({
  items,
  t,
  className = '',
  onSelectItem,
}) => {
  const processedItems = useMemo(
    () => normalizeUpdateAnnouncementItems(items),
    [items],
  );

  return (
    <div className={`update-log-timeline ${className}`.trim()}>
      <Timeline mode='alternate'>
        {processedItems.map((item, index) => (
          <Timeline.Item
            key={item.id}
            type={item.type}
            time={index === 0 ? `${t('最新')} · ${item.time}` : item.time}
          >
            <div className='update-log-item'>
              {item.title && (
                <div className='update-log-title'>{item.title}</div>
              )}
              {item.usesFrame ? (
                <button
                  className='update-announcement-detail-button'
                  type='button'
                  onClick={() => onSelectItem?.(item)}
                >
                  {t('完整 HTML 内容，点击查看详情')}
                </button>
              ) : (
                <div
                  className='update-log-content'
                  dangerouslySetInnerHTML={{
                    __html: renderMarkdownHtml(item.content || ''),
                  }}
                />
              )}
            </div>
          </Timeline.Item>
        ))}
      </Timeline>
    </div>
  );
};

export default UpdateAnnouncementTimeline;
