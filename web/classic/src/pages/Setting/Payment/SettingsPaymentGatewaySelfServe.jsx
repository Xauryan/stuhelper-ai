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

import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Row,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import { ImageUp, ShieldAlert } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';

const QR_MAX_BYTES = 300 * 1024;

const { Text } = Typography;

const normalizeLimitInput = (value) => {
  if (value === undefined || value === null || value === '') {
    return '';
  }
  return value;
};

const getPositiveLimit = (value) => {
  const amount = Number(value);
  return Number.isFinite(amount) && amount > 0 ? amount : 0;
};

export default function SettingsPaymentGatewaySelfServe(props) {
  const { t } = useTranslation();
  const sectionTitle = props.hideSectionTitle ? undefined : t('自助充值');
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    SelfServeTopUpEnabled: false,
    SelfServeAlipayEnabled: false,
    SelfServeWechatPayEnabled: false,
    SelfServeAlipayQRCode: '',
    SelfServeWechatPayQRCode: '',
    SelfServeTopUpSingleMaxAmount: '',
    SelfServeTopUpDailyMaxAmount: '',
    SelfServeRejectAutoBan: true,
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);
  const alipayFileRef = useRef(null);
  const wechatFileRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        SelfServeTopUpEnabled: Boolean(props.options.SelfServeTopUpEnabled),
        SelfServeAlipayEnabled: Boolean(props.options.SelfServeAlipayEnabled),
        SelfServeWechatPayEnabled: Boolean(
          props.options.SelfServeWechatPayEnabled,
        ),
        SelfServeAlipayQRCode: props.options.SelfServeAlipayQRCode || '',
        SelfServeWechatPayQRCode: props.options.SelfServeWechatPayQRCode || '',
        SelfServeTopUpSingleMaxAmount: normalizeLimitInput(
          props.options.SelfServeTopUpSingleMaxAmount,
        ),
        SelfServeTopUpDailyMaxAmount: normalizeLimitInput(
          props.options.SelfServeTopUpDailyMaxAmount,
        ),
        SelfServeRejectAutoBan:
          props.options.SelfServeRejectAutoBan !== undefined
            ? Boolean(props.options.SelfServeRejectAutoBan)
            : true,
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const setFieldValue = (field, value) => {
    setInputs((prev) => ({ ...prev, [field]: value }));
    formApiRef.current?.setValue(field, value);
  };

  const handleQRCodeFile = (field, event) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
      showError(t('仅支持 PNG、JPG 或 WebP 图片'));
      return;
    }
    if (file.size > QR_MAX_BYTES) {
      showError(t('二维码图片不能超过 300KB'));
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      setFieldValue(field, String(reader.result || ''));
    };
    reader.onerror = () => showError(t('读取图片失败'));
    reader.readAsDataURL(file);
  };

  const validateInputs = () => {
    if (inputs.SelfServeTopUpEnabled) {
      const singleMax = getPositiveLimit(inputs.SelfServeTopUpSingleMaxAmount);
      const dailyMax = getPositiveLimit(inputs.SelfServeTopUpDailyMaxAmount);
      if (!singleMax) {
        showError(t('请填写自助充值单笔限额'));
        return false;
      }
      if (!dailyMax) {
        showError(t('请填写自助充值每日限额'));
        return false;
      }
      if (dailyMax < singleMax) {
        showError(t('自助充值每日限额不能小于单笔限额'));
        return false;
      }
    }
    if (
      inputs.SelfServeTopUpEnabled &&
      inputs.SelfServeAlipayEnabled &&
      !inputs.SelfServeAlipayQRCode
    ) {
      showError(t('请上传或填写支付宝收款码'));
      return false;
    }
    if (
      inputs.SelfServeTopUpEnabled &&
      inputs.SelfServeWechatPayEnabled &&
      !inputs.SelfServeWechatPayQRCode
    ) {
      showError(t('请上传或填写微信收款码'));
      return false;
    }
    return true;
  };

  const submitSelfServeSetting = async () => {
    if (!validateInputs()) return;
    setLoading(true);
    try {
      const optionKeys = [
        'SelfServeTopUpEnabled',
        'SelfServeAlipayEnabled',
        'SelfServeWechatPayEnabled',
        'SelfServeAlipayQRCode',
        'SelfServeWechatPayQRCode',
        'SelfServeTopUpSingleMaxAmount',
        'SelfServeTopUpDailyMaxAmount',
        'SelfServeRejectAutoBan',
      ];
      const options = optionKeys
        .filter((key) => originInputs[key] !== inputs[key])
        .map((key) => ({
          key,
          value:
            typeof inputs[key] === 'boolean'
              ? inputs[key]
                ? 'true'
                : 'false'
              : normalizeLimitInput(inputs[key]).toString(),
        }));
      if (options.length === 0) {
        showSuccess(t('没有需要更新的设置'));
        return;
      }
      const results = await Promise.all(
        options.map((option) =>
          API.put('/api/option/', {
            key: option.key,
            value: option.value,
          }),
        ),
      );
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => showError(res.data.message));
        return;
      }
      showSuccess(t('更新成功'));
      setOriginInputs({ ...inputs });
      props.refresh?.();
    } catch (error) {
      showError(t('更新失败'));
    } finally {
      setLoading(false);
    }
  };

  const renderQRCodeInput = ({ field, label, fileRef }) => (
    <div className='space-y-3'>
      <Form.TextArea
        field={field}
        label={label}
        placeholder={t('可粘贴图片链接，或上传二维码图片自动填入')}
        autosize
        extraText={t(
          '支持 HTTPS 图片链接，或 300KB 以内的 PNG、JPG、WebP 图片',
        )}
      />
      <input
        ref={fileRef}
        type='file'
        accept='image/png,image/jpeg,image/webp'
        className='hidden'
        onChange={(event) => handleQRCodeFile(field, event)}
      />
      <div className='flex flex-wrap items-center gap-2'>
        <Button
          type='tertiary'
          icon={<ImageUp size={16} />}
          onClick={() => fileRef.current?.click()}
        >
          {t('上传收款码')}
        </Button>
        <Button type='tertiary' onClick={() => setFieldValue(field, '')}>
          {t('清空')}
        </Button>
      </div>
      {inputs[field] ? (
        <div className='rounded-lg border border-[var(--semi-color-border)] p-3 inline-block'>
          <img
            src={inputs[field]}
            alt={label}
            style={{ width: 160, height: 160, objectFit: 'contain' }}
          />
        </div>
      ) : null}
    </div>
  );

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={sectionTitle}>
          <Banner
            type='warning'
            icon={<ShieldAlert size={16} />}
            description={t(
              '自助充值会在用户提交交易订单号后实时增加余额，管理员需要每天在账单管理中核对待审核订单。虚假填写应拒绝并扣回余额，必要时封禁账户。',
            )}
            style={{ marginBottom: 16 }}
            closeIcon={null}
          />
          <div className='rounded-lg border border-[var(--semi-color-border)] px-4 py-3 mb-4'>
            <Text strong>{t('自助充值限额')}</Text>
            <div className='mt-2 text-sm text-[var(--semi-color-text-1)]'>
              {t(
                '请手动配置单笔和每日限额；例如单笔 199.99 元、每日 499.99 元。未配置完整时用户不能使用自助充值。',
              )}
            </div>
          </div>
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={6} lg={6} xl={6}>
              <Form.Switch
                field='SelfServeTopUpEnabled'
                label={t('启用自助充值')}
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={6} lg={6} xl={6}>
              <Form.Switch
                field='SelfServeAlipayEnabled'
                label={t('启用支付宝自助')}
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={6} lg={6} xl={6}>
              <Form.Switch
                field='SelfServeWechatPayEnabled'
                label={t('启用微信自助')}
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={6} lg={6} xl={6}>
              <Form.Switch
                field='SelfServeRejectAutoBan'
                label={t('拒绝时默认封禁')}
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 8 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.InputNumber
                field='SelfServeTopUpSingleMaxAmount'
                label={t('单笔限额（元）')}
                min={0.01}
                step={0.01}
                precision={2}
                placeholder={t('例如：199.99')}
                extraText={t('用户每次自助充值可提交的最高金额')}
                style={{ width: '100%' }}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.InputNumber
                field='SelfServeTopUpDailyMaxAmount'
                label={t('每日限额（元）')}
                min={0.01}
                step={0.01}
                precision={2}
                placeholder={t('例如：499.99')}
                extraText={t('单个用户每天自助充值可提交的最高累计金额')}
                style={{ width: '100%' }}
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              {renderQRCodeInput({
                field: 'SelfServeAlipayQRCode',
                label: t('支付宝收款码'),
                fileRef: alipayFileRef,
              })}
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              {renderQRCodeInput({
                field: 'SelfServeWechatPayQRCode',
                label: t('微信收款码'),
                fileRef: wechatFileRef,
              })}
            </Col>
          </Row>
          <Button onClick={submitSelfServeSetting} style={{ marginTop: 16 }}>
            {t('更新自助充值设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
