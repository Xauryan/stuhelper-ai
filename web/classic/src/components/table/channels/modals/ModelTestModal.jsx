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

import React, { useState } from 'react';
import {
  Modal,
  Button,
  Input,
  Progress,
  Table,
  Tag,
  Typography,
  Select,
  Switch,
  Banner,
} from '@douyinfe/semi-ui';
import { IconSearch, IconInfoCircle, IconCopy } from '@douyinfe/semi-icons';
import { Settings, Trash2 } from 'lucide-react';
import { copy, showError, showInfo, showSuccess } from '../../../../helpers';
import { MODEL_TABLE_PAGE_SIZE } from '../../../../constants';

const MODEL_PRICE_ERROR_CODE = 'model_price_error';
const FAILURE_SUMMARY_MAX_LENGTH = 96;

const normalizeInlineError = (errorText) =>
  String(errorText || '')
    .replace(/\s+/g, ' ')
    .trim();

const getFirstErrorLine = (errorText) =>
  String(errorText || '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find(Boolean);

const truncateFailureSummary = (summary) => {
  if (summary.length <= FAILURE_SUMMARY_MAX_LENGTH) {
    return summary;
  }
  return `${summary.slice(0, FAILURE_SUMMARY_MAX_LENGTH).trimEnd()}...`;
};

const getFailureStatusDisplay = ({
  errorText,
  fallbackSummary,
  isModelPriceError,
  modelPriceSummary,
}) => {
  const rawError = String(errorText || '').trim();

  if (!rawError) {
    return { summary: fallbackSummary, details: '' };
  }

  if (isModelPriceError) {
    return {
      summary: modelPriceSummary,
      details: rawError === modelPriceSummary ? '' : rawError,
    };
  }

  const firstLine = getFirstErrorLine(rawError) || rawError;
  const summary = truncateFailureSummary(normalizeInlineError(firstLine));
  const normalizedRawError = normalizeInlineError(rawError);

  return {
    summary,
    details: summary === normalizedRawError ? '' : rawError,
  };
};

const ModelTestModal = ({
  showModelTestModal,
  currentTestChannel,
  handleCloseModal,
  isBatchTesting,
  batchStopRequested,
  batchTestProgress,
  batchTestModels,
  stopBatchTesting,
  modelSearchKeyword,
  setModelSearchKeyword,
  selectedModelKeys,
  setSelectedModelKeys,
  modelTestResults,
  testingModels,
  testChannel,
  deleteFailedModels,
  modelTablePage,
  setModelTablePage,
  selectedEndpointType,
  setSelectedEndpointType,
  isStreamTest,
  setIsStreamTest,
  allSelectingRef,
  isMobile,
  t,
}) => {
  const hasChannel = Boolean(currentTestChannel);
  const [failureDetails, setFailureDetails] = useState(null);
  const streamToggleDisabled = [
    'embeddings',
    'image-generation',
    'jina-rerank',
    'openai-response-compact',
  ].includes(selectedEndpointType);

  const handleEndpointTypeChange = (value) => {
    setSelectedEndpointType(value);
    if (
      [
        'embeddings',
        'image-generation',
        'jina-rerank',
        'openai-response-compact',
      ].includes(value)
    ) {
      setIsStreamTest(false);
    }
  };

  const filteredModels = hasChannel
    ? currentTestChannel.models
        .split(',')
        .map((model) => model.trim())
        .filter(Boolean)
        .filter((model) =>
          model.toLowerCase().includes(modelSearchKeyword.toLowerCase()),
        )
    : [];

  const failedModels = hasChannel
    ? filteredModels.filter((model) => {
        const result = modelTestResults[`${currentTestChannel.id}-${model}`];
        return result && result.success === false;
      })
    : [];
  const progressPercent =
    batchTestProgress && batchTestProgress.total > 0
      ? Math.min(
          100,
          Math.round(
            (batchTestProgress.completed / batchTestProgress.total) * 100,
          ),
        )
      : 0;

  const endpointTypeOptions = [
    { value: '', label: t('自动检测') },
    { value: 'openai', label: 'OpenAI (/v1/chat/completions)' },
    { value: 'openai-response', label: 'OpenAI Response (/v1/responses)' },
    {
      value: 'openai-response-compact',
      label: 'OpenAI Response Compaction (/v1/responses/compact)',
    },
    { value: 'anthropic', label: 'Anthropic (/v1/messages)' },
    {
      value: 'gemini',
      label: 'Gemini (/v1beta/models/{model}:generateContent)',
    },
    { value: 'jina-rerank', label: 'Jina Rerank (/v1/rerank)' },
    {
      value: 'image-generation',
      label: t('图像生成') + ' (/v1/images/generations)',
    },
    { value: 'embeddings', label: 'Embeddings (/v1/embeddings)' },
  ];

  const handleCopySelected = () => {
    if (selectedModelKeys.length === 0) {
      showError(t('请先选择模型！'));
      return;
    }
    copy(selectedModelKeys.join(',')).then((ok) => {
      if (ok) {
        showSuccess(
          t('已复制 ${count} 个模型').replace(
            '${count}',
            selectedModelKeys.length,
          ),
        );
      } else {
        showError(t('复制失败，请手动复制'));
      }
    });
  };

  const handleSelectSuccess = () => {
    if (!currentTestChannel) return;
    const successKeys = currentTestChannel.models
      .split(',')
      .map((model) => model.trim())
      .filter(Boolean)
      .filter((m) => m.toLowerCase().includes(modelSearchKeyword.toLowerCase()))
      .filter((m) => {
        const result = modelTestResults[`${currentTestChannel.id}-${m}`];
        return result && result.success;
      });
    if (successKeys.length === 0) {
      showInfo(t('暂无成功模型'));
    }
    setSelectedModelKeys(successKeys);
  };

  const handleDeleteFailed = () => {
    if (!currentTestChannel || failedModels.length === 0) {
      showInfo(t('暂无失败模型'));
      return;
    }

    Modal.confirm({
      title: t('删除失败模型'),
      content: t(
        '将从该渠道模型列表中删除 ${count} 个测试失败的模型，此操作会保存渠道配置。',
      ).replace('${count}', failedModels.length),
      okText: t('删除'),
      cancelText: t('取消'),
      okButtonProps: { type: 'danger' },
      onOk: () => deleteFailedModels(currentTestChannel, failedModels),
    });
  };

  const handleCopyFailureDetails = () => {
    if (!failureDetails?.details) {
      return;
    }
    copy(failureDetails.details).then((ok) => {
      if (ok) {
        showSuccess(t('复制成功'));
      } else {
        showError(t('复制失败，请手动复制'));
      }
    });
  };

  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'model',
      render: (text) => (
        <div className='flex items-center'>
          <Typography.Text strong>{text}</Typography.Text>
        </div>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (text, record) => {
        const testResult =
          modelTestResults[`${currentTestChannel.id}-${record.model}`];
        const isTesting = testingModels.has(record.model);

        if (isTesting) {
          return (
            <Tag color='blue' shape='circle'>
              {t('测试中')}
            </Tag>
          );
        }

        if (!testResult) {
          return (
            <Tag color='grey' shape='circle'>
              {t('未开始')}
            </Tag>
          );
        }

        if (testResult.success) {
          return (
            <div className='flex flex-col gap-1 min-w-0'>
              <div className='flex items-center gap-2'>
                <Tag color='green' shape='circle'>
                  {t('成功')}
                </Tag>
                <Typography.Text type='tertiary'>
                  {t('请求时长: ${time}s').replace(
                    '${time}',
                    testResult.time.toFixed(2),
                  )}
                </Typography.Text>
              </div>
            </div>
          );
        }

        const isModelPriceError =
          testResult.errorCode === MODEL_PRICE_ERROR_CODE;
        const modelPriceSummary = t('模型价格未配置，请前往设置补充模型价格。');
        const rawMessage = testResult.message || t('测试失败');
        const errorText = testResult.errorCode
          ? `${rawMessage}\n\n${t('错误码')}: ${testResult.errorCode}`
          : rawMessage;
        const { summary, details } = getFailureStatusDisplay({
          errorText,
          fallbackSummary: t('测试失败'),
          isModelPriceError,
          modelPriceSummary,
        });

        return (
          <div className='flex flex-col gap-1 min-w-0'>
            <div className='flex items-center gap-2'>
              <Tag color='red' shape='circle'>
                {t('失败')}
              </Tag>
            </div>
            <Typography.Text
              type='danger'
              size='small'
              className='break-all'
              style={{
                maxWidth: isMobile ? '100%' : '420px',
                fontSize: '12px',
                lineHeight: 1.4,
              }}
            >
              {summary}
            </Typography.Text>
            <div className='flex flex-wrap items-center gap-2'>
              {isModelPriceError && (
                <Button
                  size='small'
                  theme='light'
                  type='warning'
                  icon={<Settings size={12} />}
                  onClick={() =>
                    window.open('/console/setting?tab=ratio', '_blank')
                  }
                  style={{ width: 'fit-content' }}
                >
                  {t('前往设置')}
                </Button>
              )}
              {details && (
                <Button
                  size='small'
                  theme='borderless'
                  type='tertiary'
                  icon={<IconInfoCircle />}
                  onClick={() =>
                    setFailureDetails({
                      model: record.model,
                      summary,
                      details,
                    })
                  }
                >
                  {t('详情')}
                </Button>
              )}
            </div>
          </div>
        );
      },
    },
    {
      title: '',
      dataIndex: 'operate',
      render: (text, record) => {
        const isTesting = testingModels.has(record.model);
        return (
          <Button
            type='tertiary'
            onClick={() =>
              testChannel(
                currentTestChannel,
                record.model,
                selectedEndpointType,
                isStreamTest,
              )
            }
            loading={isTesting}
            size='small'
          >
            {t('测试')}
          </Button>
        );
      },
    },
  ];

  const dataSource = (() => {
    if (!hasChannel) return [];
    const start = (modelTablePage - 1) * MODEL_TABLE_PAGE_SIZE;
    const end = start + MODEL_TABLE_PAGE_SIZE;
    return filteredModels.slice(start, end).map((model) => ({
      model,
      key: model,
    }));
  })();

  return (
    <>
      <Modal
        title={
          hasChannel ? (
            <div className='flex flex-col gap-2 w-full'>
              <div className='flex items-center gap-2'>
                <Typography.Text
                  strong
                  className='!text-[var(--semi-color-text-0)] !text-base'
                >
                  {currentTestChannel.name} {t('渠道的模型测试')}
                </Typography.Text>
                <Typography.Text type='tertiary' size='small'>
                  {t('共')} {currentTestChannel.models.split(',').length}{' '}
                  {t('个模型')}
                </Typography.Text>
              </div>
            </div>
          ) : null
        }
        visible={showModelTestModal}
        onCancel={handleCloseModal}
        footer={
          hasChannel ? (
            <div className='flex justify-end'>
              {isBatchTesting ? (
                <Button
                  type='danger'
                  onClick={stopBatchTesting}
                  disabled={batchStopRequested}
                >
                  {batchStopRequested ? t('停止中...') : t('停止测试')}
                </Button>
              ) : (
                <Button type='tertiary' onClick={handleCloseModal}>
                  {t('取消')}
                </Button>
              )}
              <Button
                onClick={batchTestModels}
                loading={isBatchTesting}
                disabled={isBatchTesting}
              >
                {isBatchTesting
                  ? t('测试中...')
                  : t('批量测试${count}个模型').replace(
                      '${count}',
                      filteredModels.length,
                    )}
              </Button>
            </div>
          ) : null
        }
        maskClosable={!isBatchTesting}
        className='!rounded-lg'
        size={isMobile ? 'full-width' : 'large'}
      >
        {hasChannel && (
          <div className='model-test-scroll'>
            {/* Endpoint toolbar */}
            <div className='flex flex-col sm:flex-row sm:items-center gap-2 w-full mb-2'>
              <div className='flex items-center gap-2 flex-1 min-w-0'>
                <Typography.Text strong className='shrink-0'>
                  {t('端点类型')}:
                </Typography.Text>
                <Select
                  value={selectedEndpointType}
                  onChange={handleEndpointTypeChange}
                  optionList={endpointTypeOptions}
                  className='!w-full min-w-0'
                  dropdownClassName='!max-w-[calc(100vw-2rem)]'
                  dropdownStyle={{ width: 460, maxWidth: 'calc(100vw - 2rem)' }}
                  renderOptionItem={(option) => (
                    <div className='whitespace-normal break-words leading-snug py-1'>
                      {option.label}
                    </div>
                  )}
                  placeholder={t('选择端点类型')}
                />
              </div>
              <div className='flex items-center justify-between sm:justify-end gap-2 shrink-0'>
                <Typography.Text strong className='shrink-0'>
                  {t('流式')}:
                </Typography.Text>
                <Switch
                  checked={isStreamTest}
                  onChange={setIsStreamTest}
                  size='small'
                  disabled={streamToggleDisabled}
                  aria-label={t('流式')}
                />
              </div>
            </div>

            <Banner
              type='info'
              closeIcon={null}
              icon={<IconInfoCircle />}
              className='!rounded-lg mb-2'
              description={t(
                '说明：本页测试为非流式请求；若渠道仅支持流式返回，可能出现测试失败，请以实际使用为准。',
              )}
            />

            {batchTestProgress && (
              <div className='rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-3 mb-2'>
                <div className='flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 mb-2'>
                  <Typography.Text strong>
                    {batchStopRequested
                      ? t('正在停止批量测试')
                      : t('批量测试进度')}
                  </Typography.Text>
                  <Typography.Text type='tertiary' size='small'>
                    {t('已完成 ${completed}/${total}')
                      .replace('${completed}', batchTestProgress.completed)
                      .replace('${total}', batchTestProgress.total)}
                  </Typography.Text>
                </div>
                <Progress
                  percent={progressPercent}
                  showInfo={false}
                  stroke='#1890ff'
                  size='small'
                />
                <div className='flex flex-wrap gap-3 mt-2 text-xs text-[var(--semi-color-text-2)]'>
                  <span>
                    {t('成功 ${count}').replace(
                      '${count}',
                      batchTestProgress.success,
                    )}
                  </span>
                  <span>
                    {t('失败 ${count}').replace(
                      '${count}',
                      batchTestProgress.failed,
                    )}
                  </span>
                </div>
              </div>
            )}

            {/* 搜索与操作按钮 */}
            <div className='flex flex-col sm:flex-row sm:items-center gap-2 w-full mb-2'>
              <Input
                placeholder={t('搜索模型...')}
                value={modelSearchKeyword}
                onChange={(v) => {
                  setModelSearchKeyword(v);
                  setModelTablePage(1);
                }}
                className='!w-full sm:!flex-1'
                prefix={<IconSearch />}
                showClear
              />

              <div className='flex items-center justify-end gap-2'>
                <Button onClick={handleCopySelected}>{t('复制已选')}</Button>
                <Button type='tertiary' onClick={handleSelectSuccess}>
                  {t('选择成功')}
                </Button>
                {failedModels.length > 0 && (
                  <Button
                    type='danger'
                    icon={<Trash2 size={14} />}
                    onClick={handleDeleteFailed}
                    disabled={isBatchTesting}
                  >
                    {t('删除失败 ${count}').replace(
                      '${count}',
                      failedModels.length,
                    )}
                  </Button>
                )}
              </div>
            </div>

            <Table
              columns={columns}
              dataSource={dataSource}
              rowSelection={{
                selectedRowKeys: selectedModelKeys,
                onChange: (keys) => {
                  if (allSelectingRef.current) {
                    allSelectingRef.current = false;
                    return;
                  }
                  setSelectedModelKeys(keys);
                },
                onSelectAll: (checked) => {
                  allSelectingRef.current = true;
                  setSelectedModelKeys(checked ? filteredModels : []);
                },
              }}
              pagination={{
                currentPage: modelTablePage,
                pageSize: MODEL_TABLE_PAGE_SIZE,
                total: filteredModels.length,
                showSizeChanger: false,
                onPageChange: (page) => setModelTablePage(page),
              }}
            />
          </div>
        )}
      </Modal>
      <Modal
        title={t('错误详情')}
        visible={Boolean(failureDetails)}
        onCancel={() => setFailureDetails(null)}
        footer={
          <div className='flex justify-end gap-2'>
            <Button type='tertiary' onClick={() => setFailureDetails(null)}>
              {t('关闭')}
            </Button>
            <Button
              icon={<IconCopy />}
              onClick={handleCopyFailureDetails}
              disabled={!failureDetails?.details}
            >
              {t('复制')}
            </Button>
          </div>
        }
        width={isMobile ? '100%' : 720}
        bodyStyle={{
          maxHeight: isMobile ? '70vh' : '64vh',
          overflowY: 'auto',
        }}
      >
        {failureDetails && (
          <div className='flex flex-col gap-3'>
            <div>
              <Typography.Text type='tertiary' size='small'>
                {t('模型名称')}
              </Typography.Text>
              <Typography.Paragraph
                strong
                className='!mb-0 break-all'
                copyable={false}
              >
                {failureDetails.model}
              </Typography.Paragraph>
            </div>
            <div>
              <Typography.Text type='tertiary' size='small'>
                {t('失败')}
              </Typography.Text>
              <Typography.Paragraph className='!mb-0 break-all'>
                {failureDetails.summary}
              </Typography.Paragraph>
            </div>
            <pre className='m-0 max-h-[44vh] overflow-auto whitespace-pre-wrap break-all rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-3 text-xs leading-5 text-[var(--semi-color-text-0)]'>
              {failureDetails.details}
            </pre>
          </div>
        )}
      </Modal>
    </>
  );
};

export default ModelTestModal;
