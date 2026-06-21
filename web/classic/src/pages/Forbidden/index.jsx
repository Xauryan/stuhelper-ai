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

import React, { useContext } from 'react';
import { Empty, Typography } from '@douyinfe/semi-ui';
import {
  IllustrationNoAccess,
  IllustrationNoAccessDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';
import PropTypes from 'prop-types';
import { StatusContext } from '../../context/Status';

const { Text } = Typography;

const Forbidden = ({ accessLimited = false }) => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const accessControl = statusState?.status?.access_control;
  const currentIp = accessControl?.request_ip || t('未知');
  const countryLabel =
    accessControl?.request_country_label ||
    accessControl?.request_country_code ||
    t('未知');

  return (
    <div className='classic-page-fill flex justify-center items-center p-8'>
      <Empty
        image={<IllustrationNoAccess style={{ width: 250, height: 250 }} />}
        darkModeImage={
          <IllustrationNoAccessDark style={{ width: 250, height: 250 }} />
        }
        description={
          accessLimited ? (
            <div className='flex flex-col items-center gap-3 text-center'>
              <Text strong>{t('本站不对您所在的地区开放。')}</Text>
              <div className='flex flex-col gap-1'>
                <Text type='secondary'>
                  {t('您当前 IP：{{ip}}', { ip: currentIp })}
                </Text>
                <Text type='secondary'>
                  {t('IP 归属地：{{location}}', { location: countryLabel })}
                </Text>
              </div>
              <Text type='tertiary'>{t('您无权访问此页面，请联系管理员')}</Text>
            </div>
          ) : (
            t('您无权访问此页面，请联系管理员')
          )
        }
      />
    </div>
  );
};

Forbidden.propTypes = {
  accessLimited: PropTypes.bool,
};

export default Forbidden;
