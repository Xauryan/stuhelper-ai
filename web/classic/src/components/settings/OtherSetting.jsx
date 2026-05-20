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

import React, { useContext, useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Row,
  Modal,
  Space,
  Card,
  Checkbox,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess, timestamp2string } from '../../helpers';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../context/Status';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import {
  FOOTER_TEMPLATE_DEFAULTS,
  TELECOM_LICENSE_TYPE_LABELS,
  buildFooterTemplateHTML,
  joinFooterLicenseTypes,
  parseFooterLicenseTypes,
} from '../layout/footerTemplate';

const LEGAL_USER_AGREEMENT_KEY = 'legal.user_agreement';
const LEGAL_PRIVACY_POLICY_KEY = 'legal.privacy_policy';
const FOOTER_TEMPLATE_OPTION_KEYS = [
  'FooterTemplateCopyrightYear',
  'FooterTemplateCopyrightOwner',
  'FooterTemplateIcpBeianNumber',
  'FooterTemplateIcpBeianUrl',
  'FooterTemplateTelecomLicenseNumber',
  'FooterTemplateTelecomLicenseUrl',
  'FooterTemplateTelecomLicenseTypes',
];

const OtherSetting = () => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
    [LEGAL_USER_AGREEMENT_KEY]: '',
    [LEGAL_PRIVACY_POLICY_KEY]: '',
    SystemName: '',
    Logo: '',
    Footer: '',
    FooterTemplateCopyrightYear: '',
    FooterTemplateCopyrightOwner: '',
    FooterTemplateIcpBeianNumber: '',
    FooterTemplateIcpBeianUrl: '',
    FooterTemplateTelecomLicenseNumber: '',
    FooterTemplateTelecomLicenseUrl: '',
    FooterTemplateTelecomLicenseTypes: '',
    About: '',
    HomePageContent: '',
  });
  let [loading, setLoading] = useState(false);
  const [showUpdateModal, setShowUpdateModal] = useState(false);
  const [statusState, statusDispatch] = useContext(StatusContext);
  const [updateData, setUpdateData] = useState({
    tag_name: '',
    content: '',
  });

  const updateOption = async (key, value) => {
    setLoading(true);
    try {
      const res = await API.put('/api/option/', {
        key,
        value,
      });
      const { success, message } = res.data;
      if (success) {
        setInputs((inputs) => ({ ...inputs, [key]: value }));
        return true;
      } else {
        showError(message);
        return false;
      }
    } finally {
      setLoading(false);
    }
  };

  const updateOptions = async (options) => {
    setLoading(true);
    try {
      for (const option of options) {
        const res = await API.put('/api/option/', option);
        const { success, message } = res.data;
        if (!success) {
          showError(message);
          return false;
        }
      }
      setInputs((inputs) => ({
        ...inputs,
        ...Object.fromEntries(options.map(({ key, value }) => [key, value])),
      }));
      return true;
    } finally {
      setLoading(false);
    }
  };

  const [loadingInput, setLoadingInput] = useState({
    [LEGAL_USER_AGREEMENT_KEY]: false,
    [LEGAL_PRIVACY_POLICY_KEY]: false,
    SystemName: false,
    Logo: false,
    HomePageContent: false,
    About: false,
    Footer: false,
    FooterTemplate: false,
    CheckUpdate: false,
    FrontendTheme: false,
  });
  const handleInputChange = async (value, e) => {
    const name = e.target.id;
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const handleFooterLicenseTypesChange = (values) => {
    setInputs((inputs) => ({
      ...inputs,
      FooterTemplateTelecomLicenseTypes: joinFooterLicenseTypes(values),
    }));
  };

  const getFooterTemplateConfig = (source = inputs) => ({
    icpBeianNumber: source.FooterTemplateIcpBeianNumber,
    icpBeianUrl: source.FooterTemplateIcpBeianUrl,
    telecomLicenseNumber: source.FooterTemplateTelecomLicenseNumber,
    telecomLicenseUrl: source.FooterTemplateTelecomLicenseUrl,
    telecomLicenseTypes: parseFooterLicenseTypes(
      source.FooterTemplateTelecomLicenseTypes,
    ),
    copyrightYear: source.FooterTemplateCopyrightYear,
    copyrightOwner: source.FooterTemplateCopyrightOwner,
  });

  const getFooterTemplateOptions = () =>
    FOOTER_TEMPLATE_OPTION_KEYS.map((key) => ({
      key,
      value: inputs[key] || '',
    }));

  const footerTemplatePreview = buildFooterTemplateHTML(
    getFooterTemplateConfig(),
  );

  const refreshFooterStatus = (overrides = {}) => {
    const currentStatus = statusState?.status || {};
    const getStatusValue = (optionKey, statusKey) => {
      if (Object.prototype.hasOwnProperty.call(overrides, optionKey)) {
        return overrides[optionKey] || '';
      }
      return currentStatus[statusKey] ?? inputs[optionKey] ?? '';
    };
    const payload = {
      ...currentStatus,
      footer_html: getStatusValue('Footer', 'footer_html'),
      footer_template_copyright_year: getStatusValue(
        'FooterTemplateCopyrightYear',
        'footer_template_copyright_year',
      ),
      footer_template_copyright_owner: getStatusValue(
        'FooterTemplateCopyrightOwner',
        'footer_template_copyright_owner',
      ),
      footer_template_icp_beian_number: getStatusValue(
        'FooterTemplateIcpBeianNumber',
        'footer_template_icp_beian_number',
      ),
      footer_template_icp_beian_url: getStatusValue(
        'FooterTemplateIcpBeianUrl',
        'footer_template_icp_beian_url',
      ),
      footer_template_telecom_license_number: getStatusValue(
        'FooterTemplateTelecomLicenseNumber',
        'footer_template_telecom_license_number',
      ),
      footer_template_telecom_license_url: getStatusValue(
        'FooterTemplateTelecomLicenseUrl',
        'footer_template_telecom_license_url',
      ),
      footer_template_telecom_license_types: getStatusValue(
        'FooterTemplateTelecomLicenseTypes',
        'footer_template_telecom_license_types',
      ),
    };
    statusDispatch({ type: 'set', payload });
  };

  // 通用设置
  const formAPISettingGeneral = useRef();
  // 通用设置 - UserAgreement
  const submitUserAgreement = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_USER_AGREEMENT_KEY]: true,
      }));
      await updateOption(
        LEGAL_USER_AGREEMENT_KEY,
        inputs[LEGAL_USER_AGREEMENT_KEY],
      );
      showSuccess(t('用户协议已更新'));
    } catch (error) {
      console.error(t('用户协议更新失败'), error);
      showError(t('用户协议更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_USER_AGREEMENT_KEY]: false,
      }));
    }
  };
  // 通用设置 - PrivacyPolicy
  const submitPrivacyPolicy = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_PRIVACY_POLICY_KEY]: true,
      }));
      await updateOption(
        LEGAL_PRIVACY_POLICY_KEY,
        inputs[LEGAL_PRIVACY_POLICY_KEY],
      );
      showSuccess(t('隐私政策已更新'));
    } catch (error) {
      console.error(t('隐私政策更新失败'), error);
      showError(t('隐私政策更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_PRIVACY_POLICY_KEY]: false,
      }));
    }
  };
  // 个性化设置
  const formAPIPersonalization = useRef();
  //  个性化设置 - SystemName
  const submitSystemName = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        SystemName: true,
      }));
      await updateOption('SystemName', inputs.SystemName);
      showSuccess(t('系统名称已更新'));
    } catch (error) {
      console.error(t('系统名称更新失败'), error);
      showError(t('系统名称更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        SystemName: false,
      }));
    }
  };

  // 个性化设置 - Logo
  const submitLogo = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Logo: true }));
      await updateOption('Logo', inputs.Logo);
      showSuccess('Logo 已更新');
    } catch (error) {
      console.error('Logo 更新失败', error);
      showError('Logo 更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Logo: false }));
    }
  };
  // 个性化设置 - 首页内容
  const submitOption = async (key) => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        HomePageContent: true,
      }));
      await updateOption(key, inputs[key]);
      showSuccess('首页内容已更新');
    } catch (error) {
      console.error('首页内容更新失败', error);
      showError('首页内容更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        HomePageContent: false,
      }));
    }
  };
  // 个性化设置 - 关于
  const submitAbout = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, About: true }));
      await updateOption('About', inputs.About);
      showSuccess('关于内容已更新');
    } catch (error) {
      console.error('关于内容更新失败', error);
      showError('关于内容更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, About: false }));
    }
  };
  // 个性化设置 - 页脚
  const submitFooter = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Footer: true }));
      const success = await updateOption('Footer', inputs.Footer);
      if (!success) {
        return;
      }
      refreshFooterStatus({ Footer: inputs.Footer });
      showSuccess('页脚内容已更新');
    } catch (error) {
      console.error('页脚内容更新失败', error);
      showError('页脚内容更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Footer: false }));
    }
  };

  const submitFooterTemplate = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        FooterTemplate: true,
      }));
      const success = await updateOptions(getFooterTemplateOptions());
      if (!success) {
        return;
      }
      refreshFooterStatus(
        Object.fromEntries(
          getFooterTemplateOptions().map(({ key, value }) => [key, value]),
        ),
      );
      showSuccess(t('页脚模板已更新'));
    } catch (error) {
      console.error(t('页脚模板更新失败'), error);
      showError(t('页脚模板更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        FooterTemplate: false,
      }));
    }
  };

  const checkUpdate = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        CheckUpdate: true,
      }));
      // Use a CORS proxy to avoid direct cross-origin requests to GitHub API
      // Option 1: Use a public CORS proxy service
      // const proxyUrl = 'https://cors-anywhere.herokuapp.com/';
      // const res = await API.get(
      //   `${proxyUrl}https://api.github.com/repos/Xauryan/stuhelper-ai/releases/latest`,
      // );

      // Option 2: Use the JSON proxy approach which often works better with GitHub API
      const res = await fetch(
        'https://api.github.com/repos/Xauryan/stuhelper-ai/releases/latest',
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
            // Adding User-Agent which is often required by GitHub API
            'User-Agent': 'stuhelper-ai-update-checker',
          },
        },
      ).then((response) => response.json());

      // Option 3: Use a local proxy endpoint
      // Create a cached version of the response to avoid frequent GitHub API calls
      // const res = await API.get('/api/status/github-latest-release');

      const { tag_name, body } = res;
      if (tag_name === statusState?.status?.version) {
        showSuccess(`已是最新版本：${tag_name}`);
      } else {
        setUpdateData({
          tag_name: tag_name,
          content: marked.parse(body),
        });
        setShowUpdateModal(true);
      }
    } catch (error) {
      console.error('Failed to check for updates:', error);
      showError('检查更新失败，请稍后再试');
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        CheckUpdate: false,
      }));
    }
  };

  const switchToDefaultFrontend = () => {
    Modal.confirm({
      title: t('切换到新版前端'),
      content: t('切换后页面会自动刷新，并进入新版前端。是否继续？'),
      okText: t('确认切换'),
      cancelText: t('取消'),
      onOk: async () => {
        try {
          setLoadingInput((loadingInput) => ({
            ...loadingInput,
            FrontendTheme: true,
          }));
          const res = await API.put('/api/option/', {
            key: 'theme.frontend',
            value: 'default',
          });
          const { success, message } = res.data;
          if (!success) {
            showError(message);
            return;
          }
          showSuccess(t('已切换到新版前端，正在刷新页面'));
          setTimeout(() => {
            window.location.reload();
          }, 600);
        } catch (error) {
          console.error('切换新版前端失败', error);
          showError(t('切换失败，请稍后重试'));
        } finally {
          setLoadingInput((loadingInput) => ({
            ...loadingInput,
            FrontendTheme: false,
          }));
        }
      },
    });
  };

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key in inputs) {
          newInputs[item.key] = item.value;
        }
      });
      setInputs(newInputs);
      formAPISettingGeneral.current.setValues(newInputs);
      formAPIPersonalization.current.setValues(newInputs);
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    getOptions();
  }, []);

  // Function to open GitHub release page
  const openGitHubRelease = () => {
    window.open(
      `https://github.com/Xauryan/stuhelper-ai/releases/tag/${updateData.tag_name}`,
      '_blank',
    );
  };

  const getStartTimeString = () => {
    const timestamp = statusState?.status?.start_time;
    return statusState.status ? timestamp2string(timestamp) : '';
  };

  return (
    <Row>
      <Col
        span={24}
        style={{
          marginTop: '10px',
          display: 'flex',
          flexDirection: 'column',
          gap: '10px',
        }}
      >
        {/* 版本信息 */}
        <Form>
          <Card>
            <Form.Section text={t('系统信息')}>
              <Row>
                <Col span={16}>
                  <Space>
                    <Text>
                      {t('当前版本')}：
                      {statusState?.status?.version || t('未知')}
                    </Text>
                    <Button
                      type='primary'
                      onClick={checkUpdate}
                      loading={loadingInput['CheckUpdate']}
                    >
                      {t('检查更新')}
                    </Button>
                    <Button
                      onClick={switchToDefaultFrontend}
                      loading={loadingInput['FrontendTheme']}
                    >
                      {t('切换到新版前端')}
                    </Button>
                  </Space>
                </Col>
              </Row>
              <Row>
                <Col span={16}>
                  <Text>
                    {t('启动时间')}：{getStartTimeString()}
                  </Text>
                </Col>
              </Row>
            </Form.Section>
          </Card>
        </Form>
        {/* 通用设置 */}
        <Form
          values={inputs}
          getFormApi={(formAPI) => (formAPISettingGeneral.current = formAPI)}
        >
          <Card>
            <Form.Section text={t('通用设置')}>
              <Form.TextArea
                label={t('用户协议')}
                placeholder={t(
                  '在此输入用户协议内容，支持 Markdown & HTML 代码',
                )}
                field={LEGAL_USER_AGREEMENT_KEY}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
                helpText={t(
                  '填写用户协议内容后，用户注册时将被要求勾选已阅读用户协议',
                )}
              />
              <Button
                onClick={submitUserAgreement}
                loading={loadingInput[LEGAL_USER_AGREEMENT_KEY]}
              >
                {t('设置用户协议')}
              </Button>
              <Form.TextArea
                label={t('隐私政策')}
                placeholder={t(
                  '在此输入隐私政策内容，支持 Markdown & HTML 代码',
                )}
                field={LEGAL_PRIVACY_POLICY_KEY}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
                helpText={t(
                  '填写隐私政策内容后，用户注册时将被要求勾选已阅读隐私政策',
                )}
              />
              <Button
                onClick={submitPrivacyPolicy}
                loading={loadingInput[LEGAL_PRIVACY_POLICY_KEY]}
              >
                {t('设置隐私政策')}
              </Button>
            </Form.Section>
          </Card>
        </Form>
        {/* 个性化设置 */}
        <Form
          values={inputs}
          getFormApi={(formAPI) => (formAPIPersonalization.current = formAPI)}
        >
          <Card>
            <Form.Section text={t('个性化设置')}>
              <Form.Input
                label={t('系统名称')}
                placeholder={t('在此输入系统名称')}
                field={'SystemName'}
                onChange={handleInputChange}
              />
              <Button
                onClick={submitSystemName}
                loading={loadingInput['SystemName']}
              >
                {t('设置系统名称')}
              </Button>
              <Form.Input
                label={t('Logo 图片地址')}
                placeholder={t('在此输入 Logo 图片地址')}
                field={'Logo'}
                onChange={handleInputChange}
              />
              <Button onClick={submitLogo} loading={loadingInput['Logo']}>
                {t('设置 Logo')}
              </Button>
              <Form.TextArea
                label={t('首页内容')}
                placeholder={t(
                  '在此输入首页内容，支持 Markdown & HTML 代码，设置后首页的状态信息将不再显示。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为首页',
                )}
                field={'HomePageContent'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button
                onClick={() => submitOption('HomePageContent')}
                loading={loadingInput['HomePageContent']}
              >
                {t('设置首页内容')}
              </Button>
              <Form.TextArea
                label={t('关于')}
                placeholder={t(
                  '在此输入新的关于内容，支持 Markdown & HTML 代码。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为关于页面',
                )}
                field={'About'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button onClick={submitAbout} loading={loadingInput['About']}>
                {t('设置关于')}
              </Button>
              {/*  */}
              <Banner
                fullMode={false}
                type='info'
                description={t(
                  '自定义页脚支持 HTML。留空时将使用默认页脚模板；默认页脚模板也为空时展示系统默认页脚。右侧设计与开发信息会保持显示。',
                )}
                closeIcon={null}
                style={{ marginTop: 15 }}
              />
              <Form.TextArea
                label={t('自定义页脚')}
                placeholder={t(
                  '在此输入自定义页脚 HTML。留空时使用下方默认模板，模板也为空时使用系统默认页脚',
                )}
                field={'Footer'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 3, maxRows: 8 }}
              />
              <Button onClick={submitFooter} loading={loadingInput['Footer']}>
                {t('设置页脚')}
              </Button>
              <div className='footer-template-settings'>
                <Text strong>{t('默认页脚模板')}</Text>
                <Row
                  gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                >
                  <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                    <Form.Input
                      label={t('版权年份')}
                      placeholder='2026'
                      field={'FooterTemplateCopyrightYear'}
                      onChange={handleInputChange}
                      extraText={t('例如：2026 或 2025-2026')}
                    />
                  </Col>
                  <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                    <Form.Input
                      label={t('版权方')}
                      placeholder='StuHelper AI.'
                      field={'FooterTemplateCopyrightOwner'}
                      onChange={handleInputChange}
                      extraText={t('例如：StuHelper AI.')}
                    />
                  </Col>
                </Row>
                <Row
                  gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                >
                  <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                    <Form.Input
                      label={t('ICP备案号')}
                      field={'FooterTemplateIcpBeianNumber'}
                      onChange={handleInputChange}
                      extraText={t('留空则不显示')}
                    />
                  </Col>
                  <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                    <Form.Input
                      label={t('ICP备案链接')}
                      placeholder={FOOTER_TEMPLATE_DEFAULTS.icpBeianUrl}
                      field={'FooterTemplateIcpBeianUrl'}
                      onChange={handleInputChange}
                      extraText={t('留空则链接到工信部备案系统')}
                    />
                  </Col>
                </Row>
                <Row
                  gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                >
                  <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                    <Form.Input
                      label={t('增值电信业务经营许可证号')}
                      field={'FooterTemplateTelecomLicenseNumber'}
                      onChange={handleInputChange}
                      extraText={t('留空则不显示')}
                    />
                  </Col>
                  <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                    <Form.Input
                      label={t('增值电信业务经营许可证链接')}
                      placeholder={FOOTER_TEMPLATE_DEFAULTS.telecomLicenseUrl}
                      field={'FooterTemplateTelecomLicenseUrl'}
                      onChange={handleInputChange}
                      extraText={t(
                        '留空则链接到工信部电信业务市场综合管理信息系统',
                      )}
                    />
                  </Col>
                </Row>
                <div className='footer-template-license-types'>
                  <Text strong>{t('许可证类型')}</Text>
                  <Checkbox.Group
                    value={parseFooterLicenseTypes(
                      inputs.FooterTemplateTelecomLicenseTypes,
                    )}
                    onChange={handleFooterLicenseTypesChange}
                  >
                    <Checkbox value='ICP'>
                      {TELECOM_LICENSE_TYPE_LABELS.ICP}
                    </Checkbox>
                    <Checkbox value='EDI'>
                      {TELECOM_LICENSE_TYPE_LABELS.EDI}
                    </Checkbox>
                  </Checkbox.Group>
                </div>
                {footerTemplatePreview && (
                  <div className='footer-template-preview'>
                    <Text type='tertiary'>{t('预览')}</Text>
                    <div
                      className='classic-footer-template-slot'
                      dangerouslySetInnerHTML={{
                        __html: footerTemplatePreview,
                      }}
                    />
                  </div>
                )}
                <Button
                  onClick={submitFooterTemplate}
                  loading={loadingInput['FooterTemplate']}
                >
                  {t('设置默认页脚模板')}
                </Button>
              </div>
            </Form.Section>
          </Card>
        </Form>
      </Col>
      <Modal
        title={t('新版本') + '：' + updateData.tag_name}
        visible={showUpdateModal}
        onCancel={() => setShowUpdateModal(false)}
        footer={[
          <Button
            key='details'
            type='primary'
            onClick={() => {
              setShowUpdateModal(false);
              openGitHubRelease();
            }}
          >
            {t('详情')}
          </Button>,
        ]}
      >
        <div dangerouslySetInnerHTML={{ __html: updateData.content }}></div>
      </Modal>
    </Row>
  );
};

export default OtherSetting;
