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
import { Banner, Checkbox, Input, Modal, Typography } from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import { SiAlipay, SiWechat } from 'react-icons/si';
import { ShieldAlert } from 'lucide-react';
import { isLegacyQRCodeImageValue } from '../qrCodeUtils';

const { Text } = Typography;

const getSelfServePaymentIcon = (paymentMethod) => {
  if (paymentMethod === 'alipay_self_serve') {
    return <SiAlipay size={18} color='#1677FF' />;
  }
  return <SiWechat size={18} color='#07C160' />;
};

const SelfServeSubscriptionModal = ({
  t,
  visible,
  selectedPlan,
  paymentMethod,
  paymentName,
  qrCode,
  expectedMoney,
  transactionNo,
  setTransactionNo,
  confirmed,
  setConfirmed,
  submitLoading,
  onSubmit,
  onCancel,
}) => {
  const plan = selectedPlan?.plan;
  const money = Number(expectedMoney || 0);
  const label = paymentName || t('自助充值');

  return (
    <Modal
      title={
        <div className='flex items-center gap-2'>
          {getSelfServePaymentIcon(paymentMethod)}
          <span>{label}</span>
        </div>
      }
      visible={visible}
      onOk={onSubmit}
      onCancel={onCancel}
      okText={t('提交并开通订阅')}
      cancelText={t('取消')}
      confirmLoading={submitLoading}
      okButtonProps={{ disabled: !confirmed }}
      maskClosable={false}
      centered
    >
      <div className='space-y-4'>
        <Banner
          type='warning'
          icon={<ShieldAlert size={16} />}
          description={t(
            '提交后订阅会立即开通。请按应付金额扫码付款并填写真实交易订单号，虚假填写、重复提交或金额不符会被拒绝、取消订阅，账户可能被封禁，概不退款。',
          )}
          closeIcon={null}
        />

        <div className='rounded-lg border border-[var(--semi-color-border)] p-3'>
          <Text type='tertiary'>{t('套餐名称')}</Text>
          <div className='mt-1'>
            <Text strong>{plan?.title || t('订阅套餐')}</Text>
          </div>
          <div className='mt-3'>
            <Text type='tertiary'>{t('应付金额')}</Text>
            <div className='text-2xl font-bold text-purple-600'>
              ¥{money.toFixed(2)}
            </div>
          </div>
        </div>

        <div className='flex flex-col items-center gap-2'>
          {qrCode ? (
            <div className='rounded-lg border border-[var(--semi-color-border)] bg-white p-2'>
              {isLegacyQRCodeImageValue(qrCode) ? (
                <img
                  src={qrCode}
                  alt={label}
                  style={{ width: 220, height: 220, objectFit: 'contain' }}
                />
              ) : (
                <QRCodeSVG value={qrCode} size={220} level='M' />
              )}
            </div>
          ) : (
            <div className='rounded-lg border border-dashed border-[var(--semi-color-border)] p-6 text-sm text-[var(--semi-color-text-2)]'>
              {t('管理员未配置收款码')}
            </div>
          )}
          <Text type='secondary'>{t('请先扫码支付，再填写下方表单')}</Text>
        </div>

        <div>
          <Text type='tertiary'>{t('交易订单号')}</Text>
          <Input
            value={transactionNo}
            onChange={setTransactionNo}
            placeholder={t('请输入微信或支付宝的交易订单号，不是商户订单号')}
            showClear
            style={{ marginTop: 6 }}
          />
        </div>

        <Checkbox
          checked={confirmed}
          onChange={(event) => setConfirmed(event.target.checked)}
        >
          {t('我确认已完成付款，并承诺金额和交易订单号真实有效')}
        </Checkbox>
      </div>
    </Modal>
  );
};

export default SelfServeSubscriptionModal;
