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
import {
  Button,
  Divider,
  Form,
  Space,
  Table,
  Typography,
  Empty,
  Modal,
  Tag,
  TextArea,
  Tooltip,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Bell, Edit, Maximize2, Plus, Save, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  API,
  formatDateTimeString,
  getRelativeTime,
  showError,
  showSuccess,
} from '../../../helpers';

const { Text } = Typography;

const SettingsUpdateLog = ({ options, refresh }) => {
  const { t } = useTranslation();
  const [notifications, setNotifications] = useState([]);
  const [showNotificationModal, setShowNotificationModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showContentModal, setShowContentModal] = useState(false);
  const [editingNotification, setEditingNotification] = useState(null);
  const [deletingNotification, setDeletingNotification] = useState(null);
  const [notificationForm, setNotificationForm] = useState({
    title: '',
    content: '',
    publishDate: new Date(),
    type: 'default',
    extra: '',
  });
  const [loading, setLoading] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);

  useEffect(() => {
    setNotifications(parseNotifications(options.Notice));
  }, [options.Notice]);

  const typeOptions = [
    { value: 'default', label: t('默认') },
    { value: 'success', label: t('成功') },
    { value: 'warning', label: t('警告') },
    { value: 'error', label: t('错误') },
  ];

  const getTypeColor = (type) => {
    const colorMap = {
      default: 'grey',
      success: 'green',
      warning: 'orange',
      error: 'red',
    };
    return colorMap[type] || 'grey';
  };

  const parseNotifications = (value) => {
    const raw = String(value || '').trim();
    if (!raw) {
      return [];
    }

    try {
      const parsed = JSON.parse(raw);
      const list = Array.isArray(parsed) ? parsed : [{ content: String(raw) }];
      return list.map((item, index) => ({
        id: item.id || index + 1,
        title: item.title || '',
        content: item.content || '',
        publishDate: item.publishDate || '',
        type: item.type || 'default',
        extra: item.extra || '',
      }));
    } catch (_) {
      return [
        {
          id: 1,
          title: '',
          content: raw,
          publishDate: new Date().toISOString(),
          type: 'default',
          extra: '',
        },
      ];
    }
  };

  const submitNotifications = async () => {
    try {
      setLoading(true);
      const res = await API.put('/api/option/', {
        key: 'Notice',
        value: JSON.stringify(notifications),
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('通知已更新'));
        setHasChanges(false);
        refresh?.();
      } else {
        showError(message);
      }
    } catch (error) {
      console.error(t('通知更新失败'), error);
      showError(t('通知更新失败'));
    } finally {
      setLoading(false);
    }
  };

  const handleAddNotification = () => {
    setEditingNotification(null);
    setNotificationForm({
      title: '',
      content: '',
      publishDate: new Date(),
      type: 'default',
      extra: '',
    });
    setShowNotificationModal(true);
  };

  const handleEditNotification = (notification) => {
    setEditingNotification(notification);
    setNotificationForm({
      title: notification.title || '',
      content: notification.content || '',
      publishDate: notification.publishDate
        ? new Date(notification.publishDate)
        : new Date(),
      type: notification.type || 'default',
      extra: notification.extra || '',
    });
    setShowNotificationModal(true);
  };

  const handleSaveNotification = async () => {
    if (!notificationForm.content || !notificationForm.publishDate) {
      showError(t('请填写完整的通知信息'));
      return;
    }

    try {
      setModalLoading(true);
      const formData = {
        ...notificationForm,
        publishDate: notificationForm.publishDate.toISOString(),
      };

      let newList;
      if (editingNotification) {
        newList = notifications.map((item) =>
          item.id === editingNotification.id ? { ...item, ...formData } : item,
        );
      } else {
        const newId = Math.max(...notifications.map((item) => item.id), 0) + 1;
        newList = [...notifications, { id: newId, ...formData }];
      }

      setNotifications(newList);
      setHasChanges(true);
      setShowNotificationModal(false);
      showSuccess(
        editingNotification
          ? t('通知已更新，请及时点击“保存设置”进行保存')
          : t('通知已添加，请及时点击“保存设置”进行保存'),
      );
    } catch (error) {
      showError(t('操作失败: ') + error.message);
    } finally {
      setModalLoading(false);
    }
  };

  const handleDeleteNotification = (notification) => {
    setDeletingNotification(notification);
    setShowDeleteModal(true);
  };

  const confirmDeleteNotification = () => {
    if (deletingNotification) {
      setNotifications(
        notifications.filter((item) => item.id !== deletingNotification.id),
      );
      setHasChanges(true);
      showSuccess(t('通知已删除，请及时点击“保存设置”进行保存'));
    }
    setShowDeleteModal(false);
    setDeletingNotification(null);
  };

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      showError(t('请先选择要删除的通知'));
      return;
    }
    setNotifications(
      notifications.filter((item) => !selectedRowKeys.includes(item.id)),
    );
    setSelectedRowKeys([]);
    setHasChanges(true);
    showSuccess(
      t('已删除 {{count}} 个通知，请及时点击“保存设置”进行保存', {
        count: selectedRowKeys.length,
      }),
    );
  };

  const columns = [
    {
      title: t('通知标题'),
      dataIndex: 'title',
      key: 'title',
      width: 180,
      render: (text) => (
        <Tooltip content={text || '-'} position='topLeft' showArrow>
          <div className='truncate max-w-[180px]'>{text || '-'}</div>
        </Tooltip>
      ),
    },
    {
      title: t('通知内容'),
      dataIndex: 'content',
      key: 'content',
      render: (text) => (
        <Tooltip content={text} position='topLeft' showArrow>
          <div className='truncate max-w-[300px]'>{text}</div>
        </Tooltip>
      ),
    },
    {
      title: t('发布时间'),
      dataIndex: 'publishDate',
      key: 'publishDate',
      width: 180,
      render: (publishDate) => (
        <div>
          <div style={{ fontWeight: 'bold' }}>
            {getRelativeTime(publishDate)}
          </div>
          <div
            style={{
              fontSize: '12px',
              color: 'var(--semi-color-text-2)',
              marginTop: '2px',
            }}
          >
            {publishDate ? formatDateTimeString(new Date(publishDate)) : '-'}
          </div>
        </div>
      ),
    },
    {
      title: t('通知类型'),
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type) => (
        <Tag color={getTypeColor(type)} shape='circle'>
          {typeOptions.find((opt) => opt.value === type)?.label || type}
        </Tag>
      ),
    },
    {
      title: t('操作'),
      key: 'action',
      fixed: 'right',
      width: 150,
      render: (_, record) => (
        <Space>
          <Button
            icon={<Edit size={14} />}
            theme='light'
            type='tertiary'
            size='small'
            onClick={() => handleEditNotification(record)}
          >
            {t('编辑')}
          </Button>
          <Button
            icon={<Trash2 size={14} />}
            type='danger'
            theme='light'
            size='small'
            onClick={() => handleDeleteNotification(record)}
          >
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  const renderHeader = () => (
    <div className='flex flex-col w-full'>
      <div className='mb-2'>
        <div className='flex items-center text-blue-500'>
          <Bell size={16} className='mr-2' />
          <Text>
            {t(
              '通知管理，可以发布多条通知；用户已读后不会重复弹窗，新通知或内容变更会重新提示。',
            )}
          </Text>
        </div>
      </div>
      <Divider margin='12px' />
      <div className='flex gap-2 w-full md:w-auto'>
        <Button
          theme='light'
          type='primary'
          icon={<Plus size={14} />}
          onClick={handleAddNotification}
        >
          {t('添加通知')}
        </Button>
        <Button
          icon={<Trash2 size={14} />}
          type='danger'
          theme='light'
          onClick={handleBatchDelete}
          disabled={selectedRowKeys.length === 0}
        >
          {t('批量删除')}{' '}
          {selectedRowKeys.length > 0 && `(${selectedRowKeys.length})`}
        </Button>
        <Button
          icon={<Save size={14} />}
          onClick={submitNotifications}
          loading={loading}
          disabled={!hasChanges}
          type='secondary'
        >
          {t('保存设置')}
        </Button>
      </div>
    </div>
  );

  return (
    <>
      <Form.Section text={renderHeader()}>
        <Table
          columns={columns}
          dataSource={notifications}
          rowKey='id'
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          scroll={{ x: 'max-content' }}
          pagination={{
            pageSize: 10,
            total: notifications.length,
            showSizeChanger: true,
            pageSizeOptions: ['5', '10', '20', '50'],
          }}
          size='middle'
          loading={loading}
          empty={
            <Empty
              image={
                <IllustrationNoResult style={{ width: 150, height: 150 }} />
              }
              darkModeImage={
                <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
              }
              description={t('暂无通知')}
              style={{ padding: 30 }}
            />
          }
          className='overflow-hidden'
        />
      </Form.Section>

      <Modal
        title={editingNotification ? t('编辑通知') : t('添加通知')}
        visible={showNotificationModal}
        onOk={handleSaveNotification}
        onCancel={() => setShowNotificationModal(false)}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={modalLoading}
      >
        <Form
          layout='vertical'
          initValues={notificationForm}
          key={editingNotification ? editingNotification.id : 'new'}
        >
          <Form.Input
            field='title'
            label={t('通知标题')}
            placeholder={t('可选，通知标题')}
            onChange={(value) =>
              setNotificationForm({ ...notificationForm, title: value })
            }
          />
          <Form.TextArea
            field='content'
            label={t('通知内容')}
            placeholder={t('请输入通知内容（支持 Markdown/HTML）')}
            rows={4}
            rules={[{ required: true, message: t('请输入通知内容') }]}
            onChange={(value) =>
              setNotificationForm({ ...notificationForm, content: value })
            }
          />
          <Button
            theme='light'
            type='tertiary'
            size='small'
            icon={<Maximize2 size={14} />}
            style={{ marginBottom: 16 }}
            onClick={() => setShowContentModal(true)}
          >
            {t('放大编辑')}
          </Button>
          <Form.DatePicker
            field='publishDate'
            label={t('发布日期')}
            type='dateTime'
            rules={[{ required: true, message: t('请选择发布日期') }]}
            onChange={(value) =>
              setNotificationForm({ ...notificationForm, publishDate: value })
            }
          />
          <Form.Select
            field='type'
            label={t('通知类型')}
            optionList={typeOptions}
            onChange={(value) =>
              setNotificationForm({ ...notificationForm, type: value })
            }
          />
          <Form.Input
            field='extra'
            label={t('说明信息')}
            placeholder={t('可选，通知的补充说明')}
            onChange={(value) =>
              setNotificationForm({ ...notificationForm, extra: value })
            }
          />
        </Form>
      </Modal>

      <Modal
        title={t('确认删除')}
        visible={showDeleteModal}
        onOk={confirmDeleteNotification}
        onCancel={() => {
          setShowDeleteModal(false);
          setDeletingNotification(null);
        }}
        okText={t('确认删除')}
        cancelText={t('取消')}
        type='warning'
        okButtonProps={{
          type: 'danger',
          theme: 'solid',
        }}
      >
        <Text>{t('确定要删除此通知吗？')}</Text>
      </Modal>

      <Modal
        title={t('编辑通知内容')}
        visible={showContentModal}
        onOk={() => setShowContentModal(false)}
        onCancel={() => setShowContentModal(false)}
        okText={t('确定')}
        cancelText={t('取消')}
        width={800}
      >
        <TextArea
          value={notificationForm.content}
          placeholder={t('请输入通知内容（支持 Markdown/HTML）')}
          rows={15}
          style={{ width: '100%' }}
          onChange={(value) =>
            setNotificationForm({ ...notificationForm, content: value })
          }
        />
      </Modal>
    </>
  );
};

export default SettingsUpdateLog;
