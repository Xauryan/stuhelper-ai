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

import { useCallback, useEffect, useMemo, useState } from 'react';
import { API } from '../../helpers';

const NOTIFICATION_READ_KEYS = 'notice_notification_read_keys';
const UPDATE_ANNOUNCEMENT_READ_KEYS = 'update_announcement_read_keys';
const LEGACY_NOTICE_READ_KEYS = 'notice_read_keys';
const LEGACY_SYSTEM_NOTIFICATION_READ_KEYS = 'system_notification_read_keys';

const readStoredKeys = (storageKey) => {
  try {
    const parsed = JSON.parse(localStorage.getItem(storageKey)) || [];
    return Array.isArray(parsed) ? parsed : [];
  } catch (_) {
    return [];
  }
};

const writeStoredKeys = (storageKey, keys) => {
  localStorage.setItem(storageKey, JSON.stringify(Array.from(new Set(keys))));
};

const hashText = (value) => {
  const text = String(value || '');
  let hash = 2166136261;
  for (let i = 0; i < text.length; i += 1) {
    hash ^= text.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(36);
};

export const normalizeNoticeNotifications = (rawNotice) => {
  const raw = String(rawNotice || '').trim();
  if (!raw) {
    return [];
  }

  let parsed = raw;
  try {
    parsed = JSON.parse(raw);
  } catch (_) {
    parsed = raw;
  }

  const list = Array.isArray(parsed) ? parsed : [parsed];
  return list
    .map((item, index) => {
      const source =
        item && typeof item === 'object' && !Array.isArray(item)
          ? item
          : { content: String(item || '') };
      return {
        id: source.id || index + 1,
        title: source.title || '',
        content: source.content || '',
        publishDate: source.publishDate || source.createdAt || '',
        type: source.type || 'default',
        extra: source.extra || '',
      };
    })
    .filter((item) => String(item.content || '').trim());
};

export const getNoticeNotificationKey = (item) => {
  const signature = [
    item?.id || '',
    item?.publishDate || '',
    item?.title || '',
    item?.content || '',
    item?.extra || '',
    item?.type || '',
  ].join('|');
  return `notice-notification:${item?.id || 'no-id'}:${item?.publishDate || 'no-date'}:${hashText(signature)}`;
};

export const getSystemNotificationKey = getNoticeNotificationKey;

export const getUpdateAnnouncementKey = (item) => {
  const signature = [
    item?.id || '',
    item?.publishDate || '',
    item?.title || '',
    item?.content || '',
    item?.extra || '',
    item?.type || '',
  ].join('|');
  return `update-announcement:${item?.id || 'no-id'}:${item?.publishDate || 'no-date'}:${hashText(signature)}`;
};

const getLegacyNoticeKey = (item) =>
  `${item?.publishDate || ''}-${(item?.content || '').slice(0, 30)}`;

const getLegacySystemAnnouncementKey = (item) => {
  const signature = [
    item?.id || '',
    item?.publishDate || '',
    item?.content || '',
    item?.extra || '',
    item?.type || '',
  ].join('|');
  return `system-notification:${item?.id || 'no-id'}:${item?.publishDate || 'no-date'}:${hashText(signature)}`;
};

const getPublishTime = (item) => {
  const time = item?.publishDate ? new Date(item.publishDate).getTime() : 0;
  return Number.isFinite(time) ? time : 0;
};

const getReadKeySet = (notifications) => {
  const currentKeys = readStoredKeys(NOTIFICATION_READ_KEYS);
  const legacyKeys = readStoredKeys(LEGACY_NOTICE_READ_KEYS);
  const readSet = new Set(currentKeys);

  if (legacyKeys.length > 0) {
    const legacySet = new Set(legacyKeys);
    const migratedKeys = notifications
      .filter((item) => legacySet.has(getLegacyNoticeKey(item)))
      .map(getNoticeNotificationKey);
    if (migratedKeys.length > 0) {
      let changed = false;
      migratedKeys.forEach((key) => {
        if (!readSet.has(key)) {
          readSet.add(key);
          changed = true;
        }
      });
      if (changed) {
        writeStoredKeys(NOTIFICATION_READ_KEYS, Array.from(readSet));
      }
    }
  }

  return readSet;
};

const getUpdateAnnouncementReadKeySet = (announcements) => {
  const currentKeys = readStoredKeys(UPDATE_ANNOUNCEMENT_READ_KEYS);
  const legacyKeys = readStoredKeys(LEGACY_SYSTEM_NOTIFICATION_READ_KEYS);
  const readSet = new Set(currentKeys);

  if (legacyKeys.length > 0) {
    const legacySet = new Set(legacyKeys);
    const migratedKeys = announcements
      .filter((item) => legacySet.has(getLegacySystemAnnouncementKey(item)))
      .map(getUpdateAnnouncementKey);
    if (migratedKeys.length > 0) {
      let changed = false;
      migratedKeys.forEach((key) => {
        if (!readSet.has(key)) {
          readSet.add(key);
          changed = true;
        }
      });
      if (changed) {
        writeStoredKeys(UPDATE_ANNOUNCEMENT_READ_KEYS, Array.from(readSet));
      }
    }
  }

  return readSet;
};

export const useNotifications = (statusState) => {
  const [noticeVisible, setNoticeVisible] = useState(false);
  const [notifications, setNotifications] = useState([]);
  const [readVersion, setReadVersion] = useState(0);

  const loadNotifications = useCallback(async () => {
    try {
      const res = await API.get('/api/notice');
      const { success, data } = res.data;
      setNotifications(success ? normalizeNoticeNotifications(data) : []);
    } catch (_) {
      setNotifications([]);
    }
  }, []);

  useEffect(() => {
    loadNotifications();
  }, [loadNotifications, statusState?.status]);

  const updateAnnouncements = useMemo(
    () => (statusState?.status?.announcements || []).slice(0, 20),
    [statusState?.status?.announcements],
  );

  const unreadNotificationItems = useMemo(() => {
    if (!notifications.length) return [];
    const readSet = getReadKeySet(notifications);
    return notifications
      .map((item, index) => {
        const key = getNoticeNotificationKey(item);
        return {
          kind: 'notification',
          key,
          item,
          publishTime: getPublishTime(item),
          order: index,
        };
      })
      .filter((entry) => !readSet.has(entry.key));
  }, [notifications, readVersion]);

  const unreadUpdateAnnouncementItems = useMemo(() => {
    if (!updateAnnouncements.length) return [];
    const readSet = getUpdateAnnouncementReadKeySet(updateAnnouncements);
    return updateAnnouncements
      .map((item, index) => {
        const key = getUpdateAnnouncementKey(item);
        return {
          kind: 'updateAnnouncement',
          key,
          item,
          publishTime: getPublishTime(item),
          order: notifications.length + index,
        };
      })
      .filter((entry) => !readSet.has(entry.key));
  }, [notifications.length, readVersion, updateAnnouncements]);

  const promptQueue = useMemo(
    () =>
      [...unreadNotificationItems, ...unreadUpdateAnnouncementItems].sort(
        (a, b) => a.publishTime - b.publishTime || a.order - b.order,
      ),
    [unreadNotificationItems, unreadUpdateAnnouncementItems],
  );

  const unreadCount = promptQueue.length;
  const autoPromptItem = noticeVisible ? null : promptQueue[0] || null;
  const autoPromptRemainingCount = noticeVisible ? 0 : promptQueue.length;

  const getUnreadKeys = () => {
    if (!notifications.length) return [];
    const readSet = getReadKeySet(notifications);
    return notifications
      .filter((item) => !readSet.has(getNoticeNotificationKey(item)))
      .map(getNoticeNotificationKey);
  };

  const handleNoticeOpen = () => {
    setNoticeVisible(true);
    loadNotifications();
  };

  const handleNoticeClose = () => {
    setNoticeVisible(false);
    if (notifications.length) {
      const readKeys = readStoredKeys(NOTIFICATION_READ_KEYS);
      const mergedKeys = Array.from(
        new Set([...readKeys, ...notifications.map(getNoticeNotificationKey)]),
      );
      writeStoredKeys(NOTIFICATION_READ_KEYS, mergedKeys);
    }
    if (updateAnnouncements.length) {
      const readKeys = readStoredKeys(UPDATE_ANNOUNCEMENT_READ_KEYS);
      const mergedKeys = Array.from(
        new Set([
          ...readKeys,
          ...updateAnnouncements.map(getUpdateAnnouncementKey),
        ]),
      );
      writeStoredKeys(UPDATE_ANNOUNCEMENT_READ_KEYS, mergedKeys);
    }
    setReadVersion((version) => version + 1);
  };

  const handleAutoPromptClose = () => {
    if (!autoPromptItem) {
      return;
    }

    const storageKey =
      autoPromptItem.kind === 'updateAnnouncement'
        ? UPDATE_ANNOUNCEMENT_READ_KEYS
        : NOTIFICATION_READ_KEYS;
    const readKeys = readStoredKeys(storageKey);
    writeStoredKeys(storageKey, [...readKeys, autoPromptItem.key]);
    setReadVersion((version) => version + 1);
  };

  return {
    noticeVisible,
    unreadCount,
    notifications,
    autoPromptItem,
    autoPromptRemainingCount,
    handleNoticeOpen,
    handleNoticeClose,
    handleAutoPromptClose,
    getUnreadKeys,
  };
};
