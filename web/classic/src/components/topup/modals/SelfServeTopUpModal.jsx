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
import {
  Banner,
  Checkbox,
  Input,
  InputNumber,
  Modal,
  Skeleton,
  Typography,
} from '@douyinfe/semi-ui';
import { SiAlipay, SiWechat } from 'react-icons/si';
import { ShieldAlert } from 'lucide-react';

const { Text } = Typography;

const getSelfServePaymentLabel = (paymentMethod, t) => {
  if (paymentMethod === 'alipay_self_serve') return t('支付宝自助');
  if (paymentMethod === 'wxpay_self_serve') return t('微信自助');
  return t('自助充值');
};

const getSelfServePaymentIcon = (paymentMethod) => {
  if (paymentMethod === 'alipay_self_serve') {
    return <SiAlipay size={18} color='#1677FF' />;
  }
  return <SiWechat size={18} color='#07C160' />;
};

const positiveMoney = (value) => {
  const money = Number(value);
  return Number.isFinite(money) && money > 0 ? money : 0;
};

const nonNegativeMoney = (value) => {
  const money = Number(value);
  return Number.isFinite(money) && money >= 0 ? money : 0;
};

const SelfServeTopUpModal = ({
  t,
  visible,
  paymentMethod,
  qrCode,
  declaredMoney,
  setDeclaredMoney,
  transactionNo,
  setTransactionNo,
  confirmed,
  setConfirmed,
  preview,
  previewLoading,
  submitLoading,
  limits,
  renderQuota,
  onSubmit,
  onCancel,
}) => {
  const label = getSelfServePaymentLabel(paymentMethod, t);
  const singleMax = positiveMoney(limits?.single_max_money);
  const dailyMax = positiveMoney(limits?.daily_max_money);
  const dailyRemain = nonNegativeMoney(limits?.daily_remain_money ?? dailyMax);

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
      okText={t('提交并实时到账')}
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
            '提交后余额会实时到账。请填写真实充值金额和真实交易订单号，虚假填写、重复提交或金额不符会被拒绝、扣回余额，账户可能被封禁，概不退款。',
          )}
          closeIcon={null}
        />

        <div className='rounded-lg border border-[var(--semi-color-border)] p-3'>
          <div className='text-sm text-[var(--semi-color-text-1)]'>
            {t('每人单笔最高 {{amount}} 元，每日最高 {{daily}} 元。', {
              amount: singleMax.toFixed(2),
              daily: dailyMax.toFixed(2),
            })}
          </div>
          <div className='text-sm text-[var(--semi-color-text-2)] mt-1'>
            {t('今日剩余可提交金额：{{amount}} 元', {
              amount: Math.max(0, dailyRemain).toFixed(2),
            })}
          </div>
        </div>

        <div className='flex flex-col items-center gap-2'>
          {qrCode ? (
            <img
              src={qrCode}
              alt={label}
              className='rounded-lg border border-[var(--semi-color-border)] bg-white p-2'
              style={{ width: 220, height: 220, objectFit: 'contain' }}
            />
          ) : (
            <div className='rounded-lg border border-dashed border-[var(--semi-color-border)] p-6 text-sm text-[var(--semi-color-text-2)]'>
              {t('管理员未配置收款码')}
            </div>
          )}
          <Text type='secondary'>{t('请先扫码支付，再填写下方表单')}</Text>
        </div>

        <div>
          <Text type='tertiary'>{t('充值金额（元）')}</Text>
          <InputNumber
            prefix='¥'
            min={0.01}
            max={Math.min(singleMax, dailyRemain)}
            step={0.01}
            precision={2}
            value={declaredMoney}
            onChange={setDeclaredMoney}
            placeholder={t('请输入实际支付金额')}
            style={{ width: '100%', marginTop: 6 }}
          />
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

        <div className='rounded-lg bg-[var(--semi-color-fill-0)] px-3 py-2'>
          <Text type='tertiary'>{t('预计到账额度')}</Text>
          <div className='mt-1'>
            {previewLoading ? (
              <Skeleton.Title style={{ width: 120, height: 18 }} />
            ) : preview?.credited_quota ? (
              <Text strong>{renderQuota(preview.credited_quota)}</Text>
            ) : (
              <Text type='tertiary'>-</Text>
            )}
          </div>
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

export default SelfServeTopUpModal;
