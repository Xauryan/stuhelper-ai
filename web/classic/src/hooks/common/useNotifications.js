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
const LEGACY_NOTICE_READ_KEYS = 'notice_read_keys';

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

const getLegacyNoticeKey = (item) =>
  `${item?.publishDate || ''}-${(item?.content || '').slice(0, 30)}`;

const getReadKeySet = (notifications) => {
  const currentKeys = readStoredKeys(NOTIFICATION_READ_KEYS);
  const legacyKeys = readStoredKeys(LEGACY_NOTICE_READ_KEYS);
  const readSet = new Set(currentKeys);

  if (currentKeys.length === 0 && legacyKeys.length > 0) {
    const legacySet = new Set(legacyKeys);
    const migratedKeys = notifications
      .filter((item) => legacySet.has(getLegacyNoticeKey(item)))
      .map(getNoticeNotificationKey);
    if (migratedKeys.length > 0) {
      migratedKeys.forEach((key) => readSet.add(key));
      writeStoredKeys(NOTIFICATION_READ_KEYS, Array.from(readSet));
    }
  }

  return readSet;
};

export const useNotifications = (statusState) => {
  const [noticeVisible, setNoticeVisible] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [notifications, setNotifications] = useState([]);

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

  const calculateUnreadCount = () => {
    if (!notifications.length) return 0;
    const readSet = getReadKeySet(notifications);
    return notifications.filter(
      (item) => !readSet.has(getNoticeNotificationKey(item)),
    ).length;
  };

  const getUnreadKeys = () => {
    if (!notifications.length) return [];
    const readSet = getReadKeySet(notifications);
    return notifications
      .filter((item) => !readSet.has(getNoticeNotificationKey(item)))
      .map(getNoticeNotificationKey);
  };

  useEffect(() => {
    setUnreadCount(calculateUnreadCount());
  }, [notifications]);

  useEffect(() => {
    if (unreadCount > 0 && !noticeVisible) {
      setNoticeVisible(true);
    }
  }, [noticeVisible, unreadCount]);

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
    setUnreadCount(0);
    setNotifications((current) => [...current]);
  };

  return {
    noticeVisible,
    unreadCount,
    notifications,
    handleNoticeOpen,
    handleNoticeClose,
    getUnreadKeys,
  };
};
