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

import { useEffect, useMemo, useState } from 'react';

const NOTIFICATION_READ_KEYS = 'system_notification_read_keys';
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

export const getSystemNotificationKey = (item) => {
  const signature = [
    item?.id || '',
    item?.publishDate || '',
    item?.content || '',
    item?.extra || '',
    item?.type || '',
  ].join('|');
  return `system-notification:${item?.id || 'no-id'}:${item?.publishDate || 'no-date'}:${hashText(signature)}`;
};

const getLegacyAnnouncementKey = (item) =>
  `${item?.publishDate || ''}-${(item?.content || '').slice(0, 30)}`;

const getReadKeySet = (announcements) => {
  const currentKeys = readStoredKeys(NOTIFICATION_READ_KEYS);
  const legacyKeys = readStoredKeys(LEGACY_NOTICE_READ_KEYS);
  const readSet = new Set(currentKeys);

  if (currentKeys.length === 0 && legacyKeys.length > 0) {
    const legacySet = new Set(legacyKeys);
    const migratedKeys = announcements
      .filter((item) => legacySet.has(getLegacyAnnouncementKey(item)))
      .map(getSystemNotificationKey);
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

  const announcements = useMemo(
    () => statusState?.status?.announcements || [],
    [statusState?.status?.announcements],
  );

  const calculateUnreadCount = () => {
    if (!announcements.length) return 0;
    const readSet = getReadKeySet(announcements);
    return announcements.filter(
      (item) => !readSet.has(getSystemNotificationKey(item)),
    ).length;
  };

  const getUnreadKeys = () => {
    if (!announcements.length) return [];
    const readSet = getReadKeySet(announcements);
    return announcements
      .filter((item) => !readSet.has(getSystemNotificationKey(item)))
      .map(getSystemNotificationKey);
  };

  useEffect(() => {
    setUnreadCount(calculateUnreadCount());
  }, [announcements]);

  useEffect(() => {
    if (unreadCount > 0 && !noticeVisible) {
      setNoticeVisible(true);
    }
  }, [noticeVisible, unreadCount]);

  const handleNoticeOpen = () => {
    setNoticeVisible(true);
  };

  const handleNoticeClose = () => {
    setNoticeVisible(false);
    if (announcements.length) {
      const readKeys = readStoredKeys(NOTIFICATION_READ_KEYS);
      const mergedKeys = Array.from(
        new Set([...readKeys, ...announcements.map(getSystemNotificationKey)]),
      );
      writeStoredKeys(NOTIFICATION_READ_KEYS, mergedKeys);
    }
    setUnreadCount(0);
  };

  return {
    noticeVisible,
    unreadCount,
    announcements,
    handleNoticeOpen,
    handleNoticeClose,
    getUnreadKeys,
  };
};
