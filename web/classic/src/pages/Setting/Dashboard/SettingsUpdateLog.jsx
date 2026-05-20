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

import React, { useEffect, useState } from 'react';
import { Button, Divider, Form, Typography } from '@douyinfe/semi-ui';
import { History, Save } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';

const { Text } = Typography;

const SettingsUpdateLog = ({ options, refresh }) => {
  const { t } = useTranslation();
  const [updateLog, setUpdateLog] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setUpdateLog(options.Notice || '');
  }, [options.Notice]);

  const submitUpdateLog = async () => {
    try {
      setLoading(true);
      const res = await API.put('/api/option/', {
        key: 'Notice',
        value: updateLog,
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('更新日志已更新'));
        refresh?.();
      } else {
        showError(message);
      }
    } catch (error) {
      console.error(t('更新日志更新失败'), error);
      showError(t('更新日志更新失败'));
    } finally {
      setLoading(false);
    }
  };

  const renderHeader = () => (
    <div className='flex flex-col w-full'>
      <div className='mb-2'>
        <div className='flex items-center text-blue-500'>
          <History size={16} className='mr-2' />
          <Text>
            {t('更新日志管理，支持发布版本更新、运营变更和完整 HTML 公告。')}
          </Text>
        </div>
      </div>
      <Divider margin='12px' />
    </div>
  );

  return (
    <Form.Section text={renderHeader()}>
      <Form values={{ Notice: updateLog }}>
        <Form.TextArea
          label={t('更新日志')}
          placeholder={t('在此输入更新日志内容，支持 Markdown & HTML 代码')}
          field='Notice'
          onChange={setUpdateLog}
          style={{ fontFamily: 'JetBrains Mono, Consolas' }}
          autosize={{ minRows: 8, maxRows: 18 }}
          helpText={t(
            '完整 HTML 会在沙盒 iframe 中展示，并允许脚本在 iframe 内运行。',
          )}
        />
        <Button
          icon={<Save size={14} />}
          onClick={submitUpdateLog}
          loading={loading}
          type='secondary'
        >
          {t('设置更新日志')}
        </Button>
      </Form>
    </Form.Section>
  );
};

export default SettingsUpdateLog;
