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

import React, { useEffect, useMemo, useState } from 'react';
import { Modal, Typography } from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import { Clock3, Loader2 } from 'lucide-react';
import {
  formatWechatQrCountdown,
  getWechatOfficialQrPaymentHint,
  normalizeOfficialPaymentOrderTimeoutSeconds,
} from '../wechatOfficialPaymentStatus.mjs';

const { Text } = Typography;

const WechatOfficialQrPaymentModal = ({
  t,
  visible,
  codeUrl,
  fallback,
  createdAt,
  orderTimeoutSeconds,
  onCancel,
}) => {
  const normalizedTimeout =
    normalizeOfficialPaymentOrderTimeoutSeconds(orderTimeoutSeconds);
  const deadline = useMemo(() => {
    const start = Number(createdAt) || Date.now();
    return start + normalizedTimeout * 1000;
  }, [createdAt, normalizedTimeout]);
  const [remainingSeconds, setRemainingSeconds] = useState(normalizedTimeout);

  useEffect(() => {
    if (!visible) {
      setRemainingSeconds(normalizedTimeout);
      return undefined;
    }

    const updateRemaining = () => {
      setRemainingSeconds(
        Math.max(0, Math.ceil((deadline - Date.now()) / 1000)),
      );
    };
    updateRemaining();
    const timer = window.setInterval(updateRemaining, 1000);
    return () => window.clearInterval(timer);
  }, [deadline, normalizedTimeout, visible]);

  const expired = remainingSeconds <= 0;

  return (
    <Modal
      title={t('微信支付扫码')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size='small'
      centered
      className='wechat-official-qr-modal'
      bodyStyle={{ padding: '18px 22px 22px' }}
    >
      <div className='flex flex-col items-center gap-4'>
        <div className='flex h-[248px] w-[248px] items-center justify-center rounded-lg border border-slate-200 bg-white p-3 shadow-sm dark:border-slate-700'>
          {codeUrl ? (
            <QRCodeSVG value={codeUrl} size={220} level='M' />
          ) : (
            <Text type='tertiary'>{t('二维码加载中')}</Text>
          )}
        </div>

        <div className='w-full space-y-2 text-center'>
          <Text strong className='text-slate-900 dark:text-slate-100'>
            {t(getWechatOfficialQrPaymentHint(fallback))}
          </Text>
          <div className='flex items-center justify-center gap-2 text-sm text-slate-600 dark:text-slate-300'>
            <Clock3 size={15} />
            <span>
              {expired
                ? t('订单已超时，请重新发起支付')
                : t('请在 {{duration}}内支付，超时无效。', {
                    duration: formatWechatQrCountdown(remainingSeconds),
                  })}
            </span>
          </div>
        </div>

        <div className='flex w-full items-center justify-center gap-2 rounded-md bg-slate-50 px-3 py-2 text-sm text-slate-600 dark:bg-slate-800 dark:text-slate-300'>
          <Loader2
            size={15}
            className={
              !expired
                ? 'wechat-official-waiting-spinner text-emerald-600'
                : 'text-slate-400'
            }
          />
          <span>
            {expired ? t('订单已停止等待支付结果') : t('支付完成后将自动刷新')}
          </span>
        </div>
      </div>
    </Modal>
  );
};

export default WechatOfficialQrPaymentModal;
