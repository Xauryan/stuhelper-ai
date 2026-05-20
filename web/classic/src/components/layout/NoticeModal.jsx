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

import React, { useEffect, useState, useContext, useMemo } from 'react';
import {
  Button,
  Modal,
  Empty,
  Tabs,
  TabPane,
  Timeline,
  Tag,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { getRelativeTime } from '../../helpers';
import { marked } from 'marked';
import {
  IllustrationNoContent,
  IllustrationNoContentDark,
} from '@douyinfe/semi-illustrations';
import { StatusContext } from '../../context/Status';
import { Bell, CheckCircle2, FileClock } from 'lucide-react';
import { getNoticeNotificationKey } from '../../hooks/common/useNotifications';

const stripHtml = (html) => {
  const div = document.createElement('div');
  div.innerHTML = html || '';
  return div.textContent || div.innerText || '';
};

const formatAbsoluteTime = (dateValue) => {
  const date = dateValue ? new Date(dateValue) : null;
  if (!date || isNaN(date.getTime())) {
    return dateValue || '';
  }
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
};

const shouldRenderFrame = (raw) =>
  /<!doctype|<html[\s>]|<head[\s>]|<body[\s>]|<style[\s>]|<script[\s>]/i.test(
    String(raw || ''),
  );

const buildFrameHtml = (raw) => {
  const source = String(raw || '');
  if (!source.trim()) {
    return '';
  }
  return source;
};

const getTimelineContent = (item) =>
  String(item?.content || '').trim() ||
  String(item?.extra || '').trim() ||
  String(item?.title || '').trim();

const splitUpdateAnnouncementItems = (items) =>
  (items || []).map((item, index) => ({
    id: item?.id || `update-announcement-${index}`,
    title:
      item?.title ||
      stripHtml(marked.parse(getTimelineContent(item))).split('\n')[0] ||
      item?.publishDate ||
      '',
    content: getTimelineContent(item),
    usesFrame: shouldRenderFrame(getTimelineContent(item)),
    frameHtml: buildFrameHtml(getTimelineContent(item)),
    type: item?.type || (index === 0 ? 'success' : 'default'),
    time:
      getRelativeTime(item?.publishDate) ||
      formatAbsoluteTime(item?.publishDate) ||
      '',
  }));

const NoticeModal = ({
  visible,
  onClose,
  isMobile,
  defaultTab = 'inApp',
  unreadKeys = [],
  notifications = [],
}) => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState(defaultTab);
  const [selectedNotification, setSelectedNotification] = useState(null);
  const [selectedUpdateAnnouncement, setSelectedUpdateAnnouncement] =
    useState(null);

  const [statusState] = useContext(StatusContext);

  const updateAnnouncements = statusState?.status?.announcements || [];

  const unreadSet = useMemo(() => new Set(unreadKeys), [unreadKeys]);

  const processedNotifications = useMemo(() => {
    return (notifications || []).slice(0, 20).map((item) => {
      const htmlContent = marked.parse(item.content || '');
      const plainContent = stripHtml(htmlContent).replace(/\s+/g, ' ').trim();
      const key = getNoticeNotificationKey(item);
      return {
        key,
        title: item.title || '',
        type: item.type || 'default',
        time: formatAbsoluteTime(item.publishDate),
        content: item.content,
        htmlContent,
        usesFrame: shouldRenderFrame(item.content || ''),
        frameHtml: buildFrameHtml(item.content || ''),
        plainContent,
        extra: item.extra,
        relative: getRelativeTime(item.publishDate),
        isUnread: unreadSet.has(key),
      };
    });
  }, [notifications, unreadSet]);

  const renderNotificationDetail = () => {
    if (!selectedNotification) {
      return null;
    }

    if (selectedNotification.usesFrame) {
      return (
        <div className='update-log-html-frame-shell notification-detail-frame-shell'>
          <iframe
            className='update-log-html-frame'
            title={selectedNotification.title || t('通知内容')}
            sandbox='allow-scripts'
            srcDoc={selectedNotification.frameHtml}
          />
        </div>
      );
    }

    return (
      <div
        className='notification-detail-content card-content-scroll'
        dangerouslySetInnerHTML={{ __html: selectedNotification.htmlContent }}
      />
    );
  };

  const updateAnnouncementItems = useMemo(
    () => splitUpdateAnnouncementItems(updateAnnouncements),
    [updateAnnouncements],
  );

  const renderUpdateAnnouncementDetail = () => {
    if (!selectedUpdateAnnouncement) {
      return null;
    }

    if (selectedUpdateAnnouncement.usesFrame) {
      return (
        <div className='update-log-html-frame-shell notification-detail-frame-shell'>
          <iframe
            className='update-log-html-frame'
            title={selectedUpdateAnnouncement.title || t('更新公告')}
            sandbox='allow-scripts'
            srcDoc={selectedUpdateAnnouncement.frameHtml}
          />
        </div>
      );
    }

    return (
      <div
        className='notification-detail-content card-content-scroll'
        dangerouslySetInnerHTML={{
          __html: marked.parse(selectedUpdateAnnouncement.content || ''),
        }}
      />
    );
  };

  useEffect(() => {
    if (visible) {
      setActiveTab(defaultTab);
    }
  }, [defaultTab, visible]);

  const renderNotificationList = () => {
    if (processedNotifications.length === 0) {
      return (
        <div className='py-12'>
          <Empty
            image={
              <IllustrationNoContent style={{ width: 150, height: 150 }} />
            }
            darkModeImage={
              <IllustrationNoContentDark style={{ width: 150, height: 150 }} />
            }
            description={t('暂无通知')}
          />
        </div>
      );
    }

    return (
      <div className='system-notification-list max-h-[58vh] overflow-y-auto card-content-scroll'>
        {processedNotifications.map((item) => (
          <div
            key={item.key}
            className={`system-notification-item ${item.isUnread ? 'is-unread' : ''}`}
            role='button'
            tabIndex={0}
            onClick={() => setSelectedNotification(item)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                setSelectedNotification(item);
              }
            }}
          >
            <div className='system-notification-status'>
              <CheckCircle2 size={18} />
            </div>
            <div className='system-notification-main'>
              <div className='system-notification-heading'>
                <div className='system-notification-title'>
                  {item.title || item.plainContent || t('通知内容')}
                </div>
                <Tag color={item.isUnread ? 'blue' : 'white'} shape='circle'>
                  {item.isUnread ? t('未读') : t('已读')}
                </Tag>
              </div>
              <div className='system-notification-meta'>
                {item.relative || item.time}
              </div>
              {item.usesFrame ? (
                <div className='system-notification-content'>
                  {item.plainContent || t('完整 HTML 内容，点击查看详情')}
                </div>
              ) : (
                <div
                  className='system-notification-content'
                  dangerouslySetInnerHTML={{ __html: item.htmlContent }}
                />
              )}
              {item.extra && (
                <div
                  className='system-notification-extra'
                  dangerouslySetInnerHTML={{ __html: marked.parse(item.extra) }}
                />
              )}
            </div>
          </div>
        ))}
      </div>
    );
  };

  const renderUpdateAnnouncementTimeline = () => {
    if (!updateAnnouncements.length) {
      return (
        <div className='py-12'>
          <Empty
            image={
              <IllustrationNoContent style={{ width: 150, height: 150 }} />
            }
            darkModeImage={
              <IllustrationNoContentDark style={{ width: 150, height: 150 }} />
            }
            description={t('暂无更新公告')}
          />
        </div>
      );
    }

    return (
      <div className='update-log-timeline max-h-[58vh] overflow-y-auto pr-2 card-content-scroll'>
        <Timeline mode='alternate'>
          {updateAnnouncementItems.map((item, idx) => {
            return (
              <Timeline.Item
                key={item.id}
                type={item.type}
                time={idx === 0 ? t('最新') : item.time}
              >
                <div className='update-log-item'>
                  <div className='update-log-title'>{item.title}</div>
                  {item.usesFrame ? (
                    <button
                      className='update-announcement-detail-button'
                      type='button'
                      onClick={() => setSelectedUpdateAnnouncement(item)}
                    >
                      {t('完整 HTML 内容，点击查看详情')}
                    </button>
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
            );
          })}
        </Timeline>
      </div>
    );
  };

  const renderBody = () => {
    if (activeTab === 'inApp') {
      return renderNotificationList();
    }
    return renderUpdateAnnouncementTimeline();
  };

  return (
    <>
      <Modal
        title={
          <div className='flex items-center justify-between w-full'>
            <span>{activeTab === 'inApp' ? t('通知') : t('更新公告')}</span>
            <Tabs activeKey={activeTab} onChange={setActiveTab} type='button'>
              <TabPane
                tab={
                  <span className='flex items-center gap-1'>
                    <Bell size={14} /> {t('通知')}
                  </span>
                }
                itemKey='inApp'
              />
              <TabPane
                tab={
                  <span className='flex items-center gap-1'>
                    <FileClock size={14} /> {t('更新公告')}
                  </span>
                }
                itemKey='system'
              />
            </Tabs>
          </div>
        }
        visible={visible}
        onCancel={onClose}
        footer={
          <div className='flex justify-end'>
            <Button type='primary' onClick={onClose}>
              {t('关闭')}
            </Button>
          </div>
        }
        size={isMobile ? 'full-width' : 'large'}
      >
        {renderBody()}
      </Modal>
      <Modal
        title={selectedNotification?.title || t('通知内容')}
        visible={Boolean(selectedNotification)}
        onCancel={() => setSelectedNotification(null)}
        footer={
          <Button type='primary' onClick={() => setSelectedNotification(null)}>
            {t('关闭')}
          </Button>
        }
        size={isMobile ? 'full-width' : 'large'}
      >
        {renderNotificationDetail()}
      </Modal>
      <Modal
        title={selectedUpdateAnnouncement?.title || t('更新公告')}
        visible={Boolean(selectedUpdateAnnouncement)}
        onCancel={() => setSelectedUpdateAnnouncement(null)}
        footer={
          <Button
            type='primary'
            onClick={() => setSelectedUpdateAnnouncement(null)}
          >
            {t('关闭')}
          </Button>
        }
        size={isMobile ? 'full-width' : 'large'}
      >
        {renderUpdateAnnouncementDetail()}
      </Modal>
    </>
  );
};

export default NoticeModal;
