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
import { API, showError, getRelativeTime } from '../../helpers';
import { marked } from 'marked';
import {
  IllustrationNoContent,
  IllustrationNoContentDark,
} from '@douyinfe/semi-illustrations';
import { StatusContext } from '../../context/Status';
import { Bell, CheckCircle2, ChevronRight, History } from 'lucide-react';
import { getSystemNotificationKey } from '../../hooks/common/useNotifications';

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

const shouldRenderUpdateLogFrame = (raw) =>
  /<!doctype|<html[\s>]|<head[\s>]|<body[\s>]|<style[\s>]/i.test(
    String(raw || ''),
  );

const buildUpdateLogFrameHtml = (raw) => {
  const source = String(raw || '');
  if (!source.trim()) {
    return '';
  }
  return source;
};

const splitUpdateLogItems = (raw) => {
  const content = String(raw || '').trim();
  if (!content) {
    return [];
  }

  const lines = content.split(/\r?\n/);
  const headingIndexes = [];

  lines.forEach((line, index) => {
    if (/^#{1,3}\s+\S/.test(line.trim())) {
      headingIndexes.push(index);
    }
  });

  if (headingIndexes.length === 0) {
    return [
      {
        id: 'update-log-0',
        content,
        title: stripHtml(marked.parse(content)).split('\n')[0] || content,
      },
    ];
  }

  return headingIndexes.map((startIndex, index) => {
    const endIndex = headingIndexes[index + 1] ?? lines.length;
    const block = lines.slice(startIndex, endIndex).join('\n').trim();
    const title = lines[startIndex].replace(/^#{1,3}\s+/, '').trim();
    return {
      id: `update-log-${index}`,
      content: block,
      title,
    };
  });
};

const NoticeModal = ({
  visible,
  onClose,
  isMobile,
  defaultTab = 'inApp',
  unreadKeys = [],
}) => {
  const { t } = useTranslation();
  const [updateLogRaw, setUpdateLogRaw] = useState('');
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState(defaultTab);

  const [statusState] = useContext(StatusContext);

  const announcements = statusState?.status?.announcements || [];

  const unreadSet = useMemo(() => new Set(unreadKeys), [unreadKeys]);
  const updateLogUsesFrame = useMemo(
    () => shouldRenderUpdateLogFrame(updateLogRaw),
    [updateLogRaw],
  );
  const updateLogHtmlSrcDoc = useMemo(
    () => (updateLogUsesFrame ? buildUpdateLogFrameHtml(updateLogRaw) : ''),
    [updateLogRaw, updateLogUsesFrame],
  );

  const processedAnnouncements = useMemo(() => {
    return (announcements || []).slice(0, 20).map((item) => {
      const htmlContent = marked.parse(item.content || '');
      const plainContent = stripHtml(htmlContent).replace(/\s+/g, ' ').trim();
      const key = getSystemNotificationKey(item);
      return {
        key,
        type: item.type || 'default',
        time: formatAbsoluteTime(item.publishDate),
        content: item.content,
        htmlContent,
        plainContent,
        extra: item.extra,
        relative: getRelativeTime(item.publishDate),
        isUnread: unreadSet.has(key),
      };
    });
  }, [announcements, unreadSet]);

  const updateLogItems = useMemo(
    () => (updateLogUsesFrame ? [] : splitUpdateLogItems(updateLogRaw)),
    [updateLogRaw, updateLogUsesFrame],
  );

  const displayNotice = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/notice');
      const { success, message, data } = res.data;
      if (success) {
        setUpdateLogRaw(data || '');
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible) {
      displayNotice();
    }
  }, [visible]);

  useEffect(() => {
    if (visible) {
      setActiveTab(defaultTab);
    }
  }, [defaultTab, visible]);

  const renderNotificationList = () => {
    if (processedAnnouncements.length === 0) {
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
      <div className='system-notification-list max-h-[58vh] overflow-y-auto'>
        {processedAnnouncements.map((item) => (
          <div
            key={item.key}
            className={`system-notification-item ${item.isUnread ? 'is-unread' : ''}`}
          >
            <div className='system-notification-status'>
              <CheckCircle2 size={18} />
            </div>
            <div className='system-notification-main'>
              <div className='system-notification-title'>
                {item.plainContent || t('通知内容')}
              </div>
              <div className='system-notification-meta'>
                {item.relative || item.time}
              </div>
              {item.extra && (
                <div
                  className='system-notification-extra'
                  dangerouslySetInnerHTML={{ __html: marked.parse(item.extra) }}
                />
              )}
            </div>
            <Tag color={item.isUnread ? 'blue' : 'white'} shape='circle'>
              {item.isUnread ? t('未读') : t('已读')}
            </Tag>
            <ChevronRight className='system-notification-chevron' size={20} />
          </div>
        ))}
      </div>
    );
  };

  const renderUpdateLogTimeline = () => {
    if (loading) {
      return (
        <div className='py-12'>
          <Empty description={t('加载中...')} />
        </div>
      );
    }

    if (!String(updateLogRaw || '').trim()) {
      return (
        <div className='py-12'>
          <Empty
            image={
              <IllustrationNoContent style={{ width: 150, height: 150 }} />
            }
            darkModeImage={
              <IllustrationNoContentDark style={{ width: 150, height: 150 }} />
            }
            description={t('暂无更新日志')}
          />
        </div>
      );
    }

    if (updateLogUsesFrame) {
      return (
        <div className='update-log-html-frame-shell'>
          <iframe
            className='update-log-html-frame'
            title={t('更新日志')}
            sandbox='allow-scripts'
            srcDoc={updateLogHtmlSrcDoc}
          />
        </div>
      );
    }

    return (
      <div className='update-log-timeline max-h-[58vh] overflow-y-auto pr-2 card-content-scroll'>
        <Timeline mode='alternate'>
          {updateLogItems.map((item, idx) => {
            return (
              <Timeline.Item
                key={item.id}
                type={idx === 0 ? 'success' : 'default'}
                time={idx === 0 ? t('最新') : ''}
              >
                <div className='update-log-item'>
                  <div className='update-log-title'>{item.title}</div>
                  <div
                    className='update-log-content'
                    dangerouslySetInnerHTML={{
                      __html: marked.parse(item.content || ''),
                    }}
                  />
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
    return renderUpdateLogTimeline();
  };

  return (
    <Modal
      title={
        <div className='flex items-center justify-between w-full'>
          <span>{activeTab === 'inApp' ? t('公告') : t('更新日志')}</span>
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
                  <History size={14} /> {t('更新日志')}
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
  );
};

export default NoticeModal;
