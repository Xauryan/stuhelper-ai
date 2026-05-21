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

import React, { useMemo, useState } from 'react';
import { Button, Card, Tag, Timeline, Empty, Modal } from '@douyinfe/semi-ui';
import { Clock3, FileClock } from 'lucide-react';
import { marked } from 'marked';
import {
  IllustrationConstruction,
  IllustrationConstructionDark,
} from '@douyinfe/semi-illustrations';
import ScrollableContainer from '../common/ui/ScrollableContainer';

const shouldRenderFrame = (raw) =>
  /<!doctype|<html[\s>]|<head[\s>]|<body[\s>]|<style[\s>]|<script[\s>]/i.test(
    String(raw || ''),
  );

const getAnnouncementContent = (item) =>
  String(item?.content || '').trim() ||
  String(item?.extra || '').trim() ||
  String(item?.title || '').trim();

const getDisplayTime = (item) => {
  const relative = String(item?.relative || '').trim();
  const absolute = String(item?.time || '').trim();

  if (relative && absolute && relative !== absolute) {
    if (absolute.startsWith(`${relative} `)) {
      return absolute;
    }
    return `${relative} · ${absolute}`;
  }
  return relative || absolute || '';
};

const AnnouncementsPanel = ({
  announcementData,
  announcementLegendData,
  CARD_PROPS,
  ILLUSTRATION_SIZE,
  t,
}) => {
  const [selectedAnnouncement, setSelectedAnnouncement] = useState(null);

  const processedAnnouncementData = useMemo(
    () =>
      (announcementData || []).map((item, index) => {
        const content = getAnnouncementContent(item);
        const usesFrame = shouldRenderFrame(content);
        return {
          ...item,
          id: item?.id || `dashboard-announcement-${index}`,
          content,
          title: String(item?.title || '').trim(),
          usesFrame,
          displayTime: getDisplayTime(item),
          htmlExtra:
            item.extra && !shouldRenderFrame(item.extra)
              ? marked.parse(item.extra)
              : '',
        };
      }),
    [announcementData],
  );

  return (
    <>
      <Card
        {...CARD_PROPS}
        className='shadow-sm !rounded-2xl lg:col-span-2'
        title={
          <div className='flex flex-col lg:flex-row lg:items-center lg:justify-between gap-2 w-full'>
            <div className='flex items-center gap-2'>
              <FileClock size={16} />
              {t('更新公告')}
              <Tag color='white' shape='circle'>
                {t('显示最新20条')}
              </Tag>
            </div>
            {/* 图例 */}
            <div className='flex flex-wrap gap-3 text-xs'>
              {announcementLegendData.map((legend, index) => (
                <div key={index} className='flex items-center gap-1'>
                  <div
                    className='w-2 h-2 rounded-full'
                    style={{
                      backgroundColor:
                        legend.color === 'grey'
                          ? '#8b9aa7'
                          : legend.color === 'blue'
                            ? '#3b82f6'
                            : legend.color === 'green'
                              ? '#10b981'
                              : legend.color === 'orange'
                                ? '#f59e0b'
                                : legend.color === 'red'
                                  ? '#ef4444'
                                  : '#8b9aa7',
                    }}
                  />
                  <span className='text-gray-600'>{legend.label}</span>
                </div>
              ))}
            </div>
          </div>
        }
        bodyStyle={{ padding: 0 }}
      >
        <ScrollableContainer maxHeight='24rem'>
          {processedAnnouncementData.length > 0 ? (
            <Timeline mode='left'>
              {processedAnnouncementData.map((item) => (
                <Timeline.Item
                  key={item.id}
                  type={item.type || 'default'}
                  time={`${item.relative ? item.relative + ' ' : ''}${item.time}`}
                  extra={
                    item.htmlExtra ? (
                      <div
                        className='text-xs text-gray-500'
                        dangerouslySetInnerHTML={{ __html: item.htmlExtra }}
                      />
                    ) : null
                  }
                >
                  <div className='update-log-item'>
                    {item.title && (
                      <div className='update-log-title'>{item.title}</div>
                    )}
                    {item.usesFrame ? (
                      <Button
                        theme='borderless'
                        type='primary'
                        size='small'
                        className='update-announcement-detail-button'
                        onClick={() => setSelectedAnnouncement(item)}
                      >
                        {t('完整 HTML 内容，点击查看详情')}
                      </Button>
                    ) : (
                      <div
                        className='update-log-content'
                        dangerouslySetInnerHTML={{
                          __html: marked.parse(item.content || ''),
                        }}
                      />
                    )}
                  </div>
                </Timeline.Item>
              ))}
            </Timeline>
          ) : (
            <div className='flex justify-center items-center py-8'>
              <Empty
                image={<IllustrationConstruction style={ILLUSTRATION_SIZE} />}
                darkModeImage={
                  <IllustrationConstructionDark style={ILLUSTRATION_SIZE} />
                }
                title={t('暂无更新公告')}
                description={t('请联系管理员在系统设置中配置更新公告')}
              />
            </div>
          )}
        </ScrollableContainer>
      </Card>
      <Modal
        title={selectedAnnouncement?.title || t('更新公告')}
        visible={Boolean(selectedAnnouncement)}
        onCancel={() => setSelectedAnnouncement(null)}
        className='html-announcement-modal'
        bodyStyle={{ padding: 12 }}
        footer={
          <Button type='primary' onClick={() => setSelectedAnnouncement(null)}>
            {t('关闭')}
          </Button>
        }
        size='large'
      >
        {selectedAnnouncement && (
          <>
            {selectedAnnouncement.displayTime && (
              <div className='notification-detail-meta'>
                <Clock3 size={13} />
                <span>{selectedAnnouncement.displayTime}</span>
              </div>
            )}
            <div className='update-log-html-frame-shell notification-detail-frame-shell'>
              <iframe
                className='update-log-html-frame'
                title={selectedAnnouncement.title || t('更新公告')}
                sandbox='allow-scripts'
                srcDoc={selectedAnnouncement.content}
              />
            </div>
          </>
        )}
      </Modal>
    </>
  );
};

export default AnnouncementsPanel;
