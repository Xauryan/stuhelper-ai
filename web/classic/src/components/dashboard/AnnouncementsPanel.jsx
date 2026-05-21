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
import { Button, Card, Tag, Empty, Modal } from '@douyinfe/semi-ui';
import { Clock3, FileClock } from 'lucide-react';
import {
  IllustrationConstruction,
  IllustrationConstructionDark,
} from '@douyinfe/semi-illustrations';
import ScrollableContainer from '../common/ui/ScrollableContainer';
import UpdateAnnouncementTimeline, {
  normalizeUpdateAnnouncementItems,
} from '../common/UpdateAnnouncementTimeline';

const AnnouncementsPanel = ({
  announcementData,
  CARD_PROPS,
  ILLUSTRATION_SIZE,
  t,
}) => {
  const [selectedAnnouncement, setSelectedAnnouncement] = useState(null);

  const processedAnnouncementData = useMemo(
    () => normalizeUpdateAnnouncementItems(announcementData),
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
          </div>
        }
        bodyStyle={{ padding: 0 }}
      >
        <ScrollableContainer maxHeight='24rem'>
          {processedAnnouncementData.length > 0 ? (
            <UpdateAnnouncementTimeline
              items={announcementData}
              t={t}
              className='dashboard-update-log-timeline card-content-scroll'
              onSelectItem={setSelectedAnnouncement}
            />
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
            {selectedAnnouncement.time && (
              <div className='notification-detail-meta'>
                <Clock3 size={13} />
                <span>{selectedAnnouncement.time}</span>
              </div>
            )}
            <div className='update-log-html-frame-shell notification-detail-frame-shell'>
              <iframe
                className='update-log-html-frame'
                title={selectedAnnouncement.title || t('更新公告')}
                sandbox='allow-scripts'
                srcDoc={selectedAnnouncement.frameHtml}
              />
            </div>
          </>
        )}
      </Modal>
    </>
  );
};

export default AnnouncementsPanel;
