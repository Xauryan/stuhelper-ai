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

import React from 'react';
import { Modal } from '@douyinfe/semi-ui';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import TopupBillingTable from './TopupBillingTable';

const TopupHistoryModal = ({ visible, onCancel, t }) => {
  const isMobile = useIsMobile();

  return (
    <Modal
      title={t('充值账单')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size={isMobile ? 'full-width' : 'large'}
      width={isMobile ? undefined : 'min(1280px, calc(100vw - 48px))'}
    >
      <TopupBillingTable active={visible} t={t} />
    </Modal>
  );
};

export default TopupHistoryModal;
