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
import { Button, Modal, Empty, Tabs, TabPane, Tag } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { getRelativeTime } from '../../helpers';
import {
  IllustrationNoContent,
  IllustrationNoContentDark,
} from '@douyinfe/semi-illustrations';
import { StatusContext } from '../../context/Status';
import {
  Bell,
  CheckCircle2,
  ChevronRight,
  CircleDot,
  Clock3,
  FileClock,
} from 'lucide-react';
import { getNoticeNotificationKey } from '../../hooks/common/useNotifications';
import UpdateAnnouncementTimeline, {
  buildFrameHtml,
  formatAbsoluteTime,
  formatDisplayTime,
  getUpdateAnnouncementContent,
  renderMarkdownHtml,
  shouldRenderFrame,
} from '../common/UpdateAnnouncementTimeline';

const stripHtml = (html) => {
  const div = document.createElement('div');
  div.innerHTML = html || '';
  div.querySelectorAll('script, style, title').forEach((node) => node.remove());
  return div.textContent || div.innerText || '';
};

const getHtmlDocumentTitle = (html) => {
  const match = String(html || '').match(/<title[^>]*>([\s\S]*?)<\/title>/i);
  return stripHtml(match?.[1] || '')
    .replace(/\s+/g, ' ')
    .trim();
};

const NoticeModal = ({
  visible,
  onClose,
  isMobile,
  defaultTab = 'inApp',
  unreadKeys = [],
  notifications = [],
  autoPromptItem = null,
  autoPromptRemainingCount = 0,
  onAutoPromptClose,
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
      const usesFrame = shouldRenderFrame(item.content || '');
      const htmlContent = renderMarkdownHtml(item.content || '');
      const plainContent = stripHtml(htmlContent).replace(/\s+/g, ' ').trim();
      const plainExtra = stripHtml(renderMarkdownHtml(item.extra || ''))
        .replace(/\s+/g, ' ')
        .trim();
      const key = getNoticeNotificationKey(item);
      return {
        key,
        title: String(item.title || '').trim(),
        type: item.type || 'default',
        time: formatAbsoluteTime(item.publishDate),
        content: item.content,
        htmlContent,
        usesFrame,
        frameHtml: buildFrameHtml(item.content || ''),
        plainContent,
        documentTitle: usesFrame
          ? getHtmlDocumentTitle(item.content || '')
          : '',
        extra: item.extra,
        plainExtra,
        relative: getRelativeTime(item.publishDate),
        displayTime: formatDisplayTime(item.publishDate),
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
            title={
              selectedNotification.title ||
              selectedNotification.documentTitle ||
              t('通知内容')
            }
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

  const autoPromptDetail = useMemo(() => {
    if (!autoPromptItem) {
      return null;
    }

    if (autoPromptItem.kind === 'updateAnnouncement') {
      const item = autoPromptItem.item;
      return {
        kind: 'updateAnnouncement',
        title: String(item?.title || '').trim() || t('更新公告'),
        content: getUpdateAnnouncementContent(item),
        usesFrame: shouldRenderFrame(getUpdateAnnouncementContent(item)),
        frameHtml: buildFrameHtml(getUpdateAnnouncementContent(item)),
        displayTime: formatDisplayTime(item?.publishDate),
      };
    }

    const item = autoPromptItem.item;
    const htmlContent = renderMarkdownHtml(item?.content || '');
    const usesFrame = shouldRenderFrame(item?.content || '');
    return {
      kind: 'notification',
      title:
        String(item?.title || '').trim() ||
        (usesFrame ? getHtmlDocumentTitle(item?.content || '') : '') ||
        t('通知内容'),
      content: item?.content || '',
      htmlContent,
      usesFrame,
      frameHtml: buildFrameHtml(item?.content || ''),
      displayTime: formatDisplayTime(item?.publishDate),
    };
  }, [autoPromptItem, t]);

  const renderDetailMeta = (displayTime) => {
    if (!displayTime) {
      return null;
    }

    return (
      <div className='notification-detail-meta'>
        <Clock3 size={13} />
        <span>{displayTime}</span>
      </div>
    );
  };

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
          __html: renderMarkdownHtml(selectedUpdateAnnouncement.content || ''),
        }}
      />
    );
  };

  const renderAutoPromptDetail = () => {
    if (!autoPromptDetail) {
      return null;
    }

    if (autoPromptDetail.usesFrame) {
      return (
        <div className='update-log-html-frame-shell notification-detail-frame-shell'>
          <iframe
            className='update-log-html-frame'
            title={autoPromptDetail.title}
            sandbox='allow-scripts'
            srcDoc={autoPromptDetail.frameHtml}
          />
        </div>
      );
    }

    return (
      <div
        className='notification-detail-content card-content-scroll'
        dangerouslySetInnerHTML={{
          __html:
            autoPromptDetail.kind === 'updateAnnouncement'
              ? renderMarkdownHtml(autoPromptDetail.content || '')
              : autoPromptDetail.htmlContent,
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
            aria-label={`${item.title || item.plainContent || t('通知内容')} ${item.isUnread ? t('未读') : t('已读')}`}
            onClick={() => setSelectedNotification(item)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                setSelectedNotification(item);
              }
            }}
          >
            <div className='system-notification-status' aria-hidden='true'>
              {item.isUnread ? (
                <CircleDot size={16} />
              ) : (
                <CheckCircle2 size={16} />
              )}
            </div>
            <div className='system-notification-main'>
              <div className='system-notification-heading'>
                <div className='system-notification-title-wrap'>
                  {item.isUnread && (
                    <span className='system-notification-unread-dot' />
                  )}
                  <div className='system-notification-title'>
                    {item.title || item.documentTitle || t('通知内容')}
                  </div>
                </div>
                <div className='system-notification-actions'>
                  <Tag color={item.isUnread ? 'blue' : 'white'} shape='circle'>
                    {item.isUnread ? t('未读') : t('已读')}
                  </Tag>
                  <ChevronRight
                    size={16}
                    className='system-notification-chevron'
                  />
                </div>
              </div>
              <div className='system-notification-meta'>
                <Clock3 size={13} />
                <span>{item.displayTime || item.relative || item.time}</span>
              </div>
              <div className='system-notification-content'>
                {item.plainContent || t('完整 HTML 内容，点击查看详情')}
              </div>
              {item.plainExtra && (
                <div className='system-notification-extra'>
                  {item.plainExtra}
                </div>
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
      <UpdateAnnouncementTimeline
        items={updateAnnouncements}
        t={t}
        className='max-h-[58vh] overflow-y-auto pr-2 card-content-scroll'
        onSelectItem={setSelectedUpdateAnnouncement}
      />
    );
  };

  const renderBody = () => {
    if (activeTab === 'inApp') {
      return renderNotificationList();
    }
    return renderUpdateAnnouncementTimeline();
  };

  const notificationDetailIsFrame = Boolean(selectedNotification?.usesFrame);
  const updateAnnouncementDetailIsFrame = Boolean(
    selectedUpdateAnnouncement?.usesFrame,
  );
  const autoPromptDetailIsFrame = Boolean(autoPromptDetail?.usesFrame);
  const htmlFrameModalProps = {
    className: 'html-announcement-modal',
    bodyStyle: { padding: 12 },
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
        title={
          selectedNotification?.title ||
          selectedNotification?.documentTitle ||
          t('通知内容')
        }
        visible={Boolean(selectedNotification)}
        onCancel={() => setSelectedNotification(null)}
        {...(notificationDetailIsFrame ? htmlFrameModalProps : {})}
        footer={
          <Button type='primary' onClick={() => setSelectedNotification(null)}>
            {t('关闭')}
          </Button>
        }
        size={isMobile ? 'full-width' : 'large'}
      >
        {renderDetailMeta(selectedNotification?.displayTime)}
        {renderNotificationDetail()}
      </Modal>
      <Modal
        title={selectedUpdateAnnouncement?.title || t('更新公告')}
        visible={Boolean(selectedUpdateAnnouncement)}
        onCancel={() => setSelectedUpdateAnnouncement(null)}
        {...(updateAnnouncementDetailIsFrame ? htmlFrameModalProps : {})}
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
        {renderDetailMeta(selectedUpdateAnnouncement?.time)}
        {renderUpdateAnnouncementDetail()}
      </Modal>
      <Modal
        title={autoPromptDetail?.title || t('通知内容')}
        visible={Boolean(autoPromptDetail)}
        onCancel={onAutoPromptClose}
        {...(autoPromptDetailIsFrame ? htmlFrameModalProps : {})}
        footer={
          <div className='flex items-center justify-between w-full'>
            <span className='text-xs text-semi-color-text-2'>
              {autoPromptRemainingCount > 1 ? t('关闭后继续显示下一条') : ''}
            </span>
            <Button type='primary' onClick={onAutoPromptClose}>
              {autoPromptRemainingCount > 1 ? t('下一条') : t('关闭')}
            </Button>
          </div>
        }
        size={isMobile ? 'full-width' : 'large'}
      >
        {renderDetailMeta(autoPromptDetail?.displayTime)}
        {renderAutoPromptDetail()}
      </Modal>
    </>
  );
};

export default NoticeModal;
