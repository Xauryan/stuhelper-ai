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
import { Banner, Button, Col, Form, Row, Spin, Tabs } from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
  toBoolean,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { BookOpen, TriangleAlert } from 'lucide-react';
import {
  buildOfficialChinaPaymentOptions,
  hasSubmittedOrStoredOfficialChinaPaymentValue,
} from './officialChinaPaymentSettings';

const defaultInputs = {
  AlipayOfficialEnabled: false,
  AlipayOfficialSandbox: false,
  AlipayOfficialAppID: '',
  AlipayOfficialPrivateKey: '',
  AlipayOfficialAlipayPublicKey: '',
  AlipayOfficialAppCertSN: '',
  AlipayOfficialRootCertSN: '',
  AlipayOfficialAlipayCertSN: '',
  AlipayOfficialNotifyURL: '',
  AlipayOfficialReturnURL: '',
  AlipayOfficialUnitPrice: 1.0,
  AlipayOfficialMinTopUp: 1,

  WechatPayOfficialEnabled: false,
  WechatPayOfficialAppID: '',
  WechatPayOfficialMchID: '',
  WechatPayOfficialCertificateSerial: '',
  WechatPayOfficialAPIv3Key: '',
  WechatPayOfficialPrivateKey: '',
  WechatPayOfficialPlatformPublicKey: '',
  WechatPayOfficialNotifyURL: '',
  WechatPayOfficialReturnURL: '',
  WechatPayOfficialUnitPrice: 1.0,
  WechatPayOfficialMinTopUp: 1,
};

export default function SettingsPaymentGatewayOfficialChina(props) {
  const { t } = useTranslation();
  const sectionTitle = props.hideSectionTitle ? undefined : t('官方支付设置');
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState(defaultInputs);
  const formApiRef = useRef(null);

  useEffect(() => {
    if (!props.options || !formApiRef.current) return;

    const currentInputs = {
      ...defaultInputs,
      AlipayOfficialEnabled: toBoolean(props.options.AlipayOfficialEnabled),
      AlipayOfficialSandbox: toBoolean(props.options.AlipayOfficialSandbox),
      AlipayOfficialAppID: props.options.AlipayOfficialAppID || '',
      AlipayOfficialAppCertSN: props.options.AlipayOfficialAppCertSN || '',
      AlipayOfficialRootCertSN:
        props.options.AlipayOfficialRootCertSN || '',
      AlipayOfficialAlipayCertSN:
        props.options.AlipayOfficialAlipayCertSN || '',
      AlipayOfficialNotifyURL: props.options.AlipayOfficialNotifyURL || '',
      AlipayOfficialReturnURL: props.options.AlipayOfficialReturnURL || '',
      AlipayOfficialUnitPrice:
        props.options.AlipayOfficialUnitPrice !== undefined
          ? parseFloat(props.options.AlipayOfficialUnitPrice)
          : 1.0,
      AlipayOfficialMinTopUp:
        props.options.AlipayOfficialMinTopUp !== undefined
          ? parseFloat(props.options.AlipayOfficialMinTopUp)
          : 1,

      WechatPayOfficialEnabled: toBoolean(
        props.options.WechatPayOfficialEnabled,
      ),
      WechatPayOfficialAppID: props.options.WechatPayOfficialAppID || '',
      WechatPayOfficialMchID: props.options.WechatPayOfficialMchID || '',
      WechatPayOfficialCertificateSerial:
        props.options.WechatPayOfficialCertificateSerial || '',
      WechatPayOfficialPlatformPublicKey:
        props.options.WechatPayOfficialPlatformPublicKey || '',
      WechatPayOfficialNotifyURL:
        props.options.WechatPayOfficialNotifyURL || '',
      WechatPayOfficialReturnURL:
        props.options.WechatPayOfficialReturnURL || '',
      WechatPayOfficialUnitPrice:
        props.options.WechatPayOfficialUnitPrice !== undefined
          ? parseFloat(props.options.WechatPayOfficialUnitPrice)
          : 1.0,
      WechatPayOfficialMinTopUp:
        props.options.WechatPayOfficialMinTopUp !== undefined
          ? parseFloat(props.options.WechatPayOfficialMinTopUp)
          : 1,
    };

    setInputs(currentInputs);
    formApiRef.current.setValues(currentInputs);
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitOfficialSetting = async () => {
    const values = {
      ...inputs,
      ...(formApiRef.current?.getValues?.() || {}),
    };
    values.AlipayOfficialEnabled = toBoolean(values.AlipayOfficialEnabled);
    values.AlipayOfficialSandbox = toBoolean(values.AlipayOfficialSandbox);
    values.WechatPayOfficialEnabled = toBoolean(
      values.WechatPayOfficialEnabled,
    );

    if (values.AlipayOfficialEnabled) {
      if (!String(values.AlipayOfficialAppID || '').trim()) {
        showError(t('请输入支付宝 AppID'));
        return;
      }
      if (
        !hasSubmittedOrStoredOfficialChinaPaymentValue(
          values,
          props.options,
          'AlipayOfficialAlipayPublicKey',
        )
      ) {
        showError(t('请输入支付宝公钥'));
        return;
      }
      if (Number(values.AlipayOfficialUnitPrice) <= 0) {
        showError(t('充值价格必须大于 0'));
        return;
      }
      if (Number(values.AlipayOfficialMinTopUp) < 1) {
        showError(t('最低充值美元数量必须大于 0'));
        return;
      }
    }

    if (values.WechatPayOfficialEnabled) {
      if (!String(values.WechatPayOfficialAppID || '').trim()) {
        showError(t('请输入微信支付 AppID'));
        return;
      }
      if (!String(values.WechatPayOfficialMchID || '').trim()) {
        showError(t('请输入微信支付商户号'));
        return;
      }
      if (
        !String(values.WechatPayOfficialCertificateSerial || '').trim()
      ) {
        showError(t('请输入微信支付商户证书序列号'));
        return;
      }
      if (
        !hasSubmittedOrStoredOfficialChinaPaymentValue(
          values,
          props.options,
          'WechatPayOfficialPlatformPublicKey',
        )
      ) {
        showError(t('请输入微信支付平台公钥'));
        return;
      }
      if (Number(values.WechatPayOfficialUnitPrice) <= 0) {
        showError(t('充值价格必须大于 0'));
        return;
      }
      if (Number(values.WechatPayOfficialMinTopUp) < 1) {
        showError(t('最低充值美元数量必须大于 0'));
        return;
      }
    }

    const options = buildOfficialChinaPaymentOptions(values, props.options);

    setLoading(true);
    try {
      const results = await Promise.all(
        options.map((opt) =>
          API.put('/api/option/', {
            key: opt.key,
            value: opt.value,
          }),
        ),
      );
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => showError(res.data.message));
        return;
      }
      showSuccess(t('更新成功'));
      props.refresh?.();
    } catch (error) {
      showError(t('更新失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={sectionTitle}>
          <Banner
            type='info'
            icon={<BookOpen size={16} />}
            description={
              <>
                {t(
                  '官方支付接入使用支付宝电脑网站支付、支付宝手机网站支付、微信 Native 支付和微信 H5 支付。',
                )}
                <br />
                {t('支付宝回调地址')}：
                {props.options.ServerAddress
                  ? removeTrailingSlash(props.options.ServerAddress)
                  : t('网站地址')}
                /api/alipay/official/notify
                <br />
                {t('微信支付回调地址')}：
                {props.options.ServerAddress
                  ? removeTrailingSlash(props.options.ServerAddress)
                  : t('网站地址')}
                /api/wechat-pay/official/notify
              </>
            }
            style={{ marginBottom: 12 }}
          />
          <Banner
            type='warning'
            icon={<TriangleAlert size={16} />}
            description={t(
              '必须使用企业主体开通的官方产品能力；这不是易支付，也不是支付宝当面付。',
            )}
            style={{ marginBottom: 16 }}
          />

          <Tabs type='card' defaultActiveKey='alipay'>
            <Tabs.TabPane tab={t('支付宝官方')} itemKey='alipay'>
              <Row
                gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
              >
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Switch
                    field='AlipayOfficialEnabled'
                    label={t('启用支付宝官方支付')}
                    checkedText='｜'
                    uncheckedText='〇'
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Switch
                    field='AlipayOfficialSandbox'
                    label={t('支付宝沙盒模式')}
                    checkedText='｜'
                    uncheckedText='〇'
                  />
                </Col>
                <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                  <Form.Input
                    field='AlipayOfficialAppID'
                    label={t('支付宝 AppID')}
                    placeholder='2021000000000000'
                  />
                </Col>
              </Row>

              <Row
                gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                style={{ marginTop: 16 }}
              >
                <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                  <Form.TextArea
                    field='AlipayOfficialPrivateKey'
                    label={t('支付宝应用私钥')}
                    placeholder={t('填写后覆盖当前私钥，留空表示保持当前不变')}
                    type='password'
                    autosize={{ minRows: 4, maxRows: 8 }}
                  />
                </Col>
                <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                  <Form.TextArea
                    field='AlipayOfficialAlipayPublicKey'
                    label={t('支付宝公钥')}
                    placeholder={t(
                      '填写后覆盖当前支付宝公钥，留空表示保持当前不变',
                    )}
                    autosize={{ minRows: 4, maxRows: 8 }}
                  />
                </Col>
              </Row>

              <Row
                gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                style={{ marginTop: 16 }}
              >
                <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                  <Form.Input
                    field='AlipayOfficialAppCertSN'
                    label={t('应用公钥证书 SN')}
                    placeholder={t('普通公钥模式可留空')}
                  />
                </Col>
                <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                  <Form.Input
                    field='AlipayOfficialRootCertSN'
                    label={t('支付宝根证书 SN')}
                    placeholder={t('普通公钥模式可留空')}
                  />
                </Col>
                <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                  <Form.Input
                    field='AlipayOfficialAlipayCertSN'
                    label={t('支付宝公钥证书 SN')}
                    placeholder={t('普通公钥模式可留空')}
                  />
                </Col>
              </Row>

              <Row
                gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                style={{ marginTop: 16 }}
              >
                <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                  <Form.Input
                    field='AlipayOfficialNotifyURL'
                    label={t('支付宝异步通知地址')}
                    placeholder={t('留空则使用默认回调地址')}
                  />
                </Col>
                <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                  <Form.Input
                    field='AlipayOfficialReturnURL'
                    label={t('支付宝支付返回地址')}
                    placeholder={t('留空则使用默认充值页地址')}
                  />
                </Col>
              </Row>

              <Row
                gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                style={{ marginTop: 16 }}
              >
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.InputNumber
                    field='AlipayOfficialUnitPrice'
                    precision={2}
                    step={0.01}
                    label={t('充值价格（x元/美金）')}
                    min={0}
                    extraText={t('支持两位小数，例如：7.23')}
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.InputNumber
                    field='AlipayOfficialMinTopUp'
                    label={t('最低充值美元数量')}
                    min={1}
                  />
                </Col>
              </Row>
            </Tabs.TabPane>

            <Tabs.TabPane tab={t('微信支付官方')} itemKey='wechat'>
              <Row
                gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
              >
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Switch
                    field='WechatPayOfficialEnabled'
                    label={t('启用微信支付官方支付')}
                    checkedText='｜'
                    uncheckedText='〇'
                  />
                </Col>
                <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                  <Form.Input
                    field='WechatPayOfficialAppID'
                    label={t('微信支付 AppID')}
                    placeholder='wx0000000000000000'
                  />
                </Col>
                <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                  <Form.Input
                    field='WechatPayOfficialMchID'
                    label={t('微信支付商户号')}
                    placeholder='1900000000'
                  />
                </Col>
              </Row>

              <Row
                gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                style={{ marginTop: 16 }}
              >
                <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                  <Form.Input
                    field='WechatPayOfficialCertificateSerial'
                    label={t('微信支付商户证书序列号')}
                    placeholder={t('例如：7775B6A45ACD...')}
                  />
                </Col>
                <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                  <Form.Input
                    field='WechatPayOfficialAPIv3Key'
                    label={t('微信支付 APIv3 密钥')}
                    placeholder={t('填写后覆盖当前密钥，留空表示保持当前不变')}
                    type='password'
                  />
                </Col>
              </Row>

              <Row
                gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                style={{ marginTop: 16 }}
              >
                <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                  <Form.TextArea
                    field='WechatPayOfficialPrivateKey'
                    label={t('微信支付商户私钥')}
                    placeholder={t('填写后覆盖当前私钥，留空表示保持当前不变')}
                    type='password'
                    autosize={{ minRows: 4, maxRows: 8 }}
                  />
                </Col>
                <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                  <Form.TextArea
                    field='WechatPayOfficialPlatformPublicKey'
                    label={t('微信支付平台公钥')}
                    placeholder={t(
                      '填写后覆盖当前微信支付平台公钥，留空表示保持当前不变',
                    )}
                    extraText={t('用于校验微信支付回调签名')}
                    autosize={{ minRows: 4, maxRows: 8 }}
                  />
                  <Form.Input
                    field='WechatPayOfficialNotifyURL'
                    label={t('微信支付异步通知地址')}
                    placeholder={t('留空则使用默认回调地址')}
                    style={{ marginTop: 16 }}
                  />
                  <Form.Input
                    field='WechatPayOfficialReturnURL'
                    label={t('微信支付 H5 返回地址')}
                    placeholder={t('留空则使用默认充值页地址')}
                    style={{ marginTop: 16 }}
                  />
                </Col>
              </Row>

              <Row
                gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                style={{ marginTop: 16 }}
              >
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.InputNumber
                    field='WechatPayOfficialUnitPrice'
                    precision={2}
                    step={0.01}
                    label={t('充值价格（x元/美金）')}
                    min={0}
                    extraText={t('支持两位小数，例如：7.23')}
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.InputNumber
                    field='WechatPayOfficialMinTopUp'
                    label={t('最低充值美元数量')}
                    min={1}
                  />
                </Col>
              </Row>
            </Tabs.TabPane>
          </Tabs>

          <Button onClick={submitOfficialSetting} style={{ marginTop: 16 }}>
            {t('更新官方支付设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
