/*
Copyright (C) 2025 QuantumNous

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

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import {
  Button,
  Space,
  Tag,
  Tooltip,
  Progress,
  Popover,
  Typography,
  Dropdown,
} from '@douyinfe/semi-ui';
import { IconMore } from '@douyinfe/semi-icons';
import {
  hasAnyPermission,
  renderGroup,
  renderNumber,
  renderQuota,
} from '../../../helpers';

const renderRole = (record, t) => {
  const effectiveRole =
    record.effective_staff_role ||
    record.staff_role ||
    (record.role >= 100 ? 'root' : record.role >= 10 ? 'admin' : '');

  if (record.role >= 100) {
    return (
      <Tag color='orange' shape='circle'>
        {t('超级管理员')}
      </Tag>
    );
  }

  if (effectiveRole === 'admin') {
    return (
      <Tag color='yellow' shape='circle'>
        {t('运营管理员')}
      </Tag>
    );
  }
  if (effectiveRole === 'finance') {
    return (
      <Tag color='cyan' shape='circle'>
        {t('财务')}
      </Tag>
    );
  }
  if (effectiveRole === 'audit') {
    return (
      <Tag color='violet' shape='circle'>
        {t('审计')}
      </Tag>
    );
  }
  return (
    <Tag color='blue' shape='circle'>
      {t('普通用户')}
    </Tag>
  );
};

const renderUsername = (text, record) => {
  const remark = record.remark;
  if (!remark) {
    return <span>{text}</span>;
  }
  const maxLen = 10;
  const displayRemark =
    remark.length > maxLen ? `${remark.slice(0, maxLen)}...` : remark;
  return (
    <Space spacing={2}>
      <span>{text}</span>
      <Tooltip content={remark} position='top' showArrow>
        <Tag color='white' shape='circle' className='!text-xs'>
          <div className='flex items-center gap-1'>
            <div
              className='w-2 h-2 flex-shrink-0 rounded-full'
              style={{ backgroundColor: '#10b981' }}
            />
            {displayRemark}
          </div>
        </Tag>
      </Tooltip>
    </Space>
  );
};

const renderStatistics = (record, t) => {
  const isDeleted = record.DeletedAt !== null;

  let tagColor = 'grey';
  let tagText = t('未知状态');
  if (isDeleted) {
    tagColor = 'red';
    tagText = t('已注销');
  } else if (record.status === 1) {
    tagColor = 'green';
    tagText = t('已启用');
  } else if (record.status === 2) {
    tagColor = 'red';
    tagText = t('已禁用');
  }

  const tooltipContent = (
    <div className='text-xs'>
      <div>
        {t('调用次数')}: {renderNumber(record.request_count)}
      </div>
    </div>
  );

  return (
    <Tooltip content={tooltipContent} position='top'>
      <Tag color={tagColor} shape='circle' size='small'>
        {tagText}
      </Tag>
    </Tooltip>
  );
};

const renderQuotaUsage = (record, t) => {
  const { Paragraph } = Typography;
  const used = parseInt(record.used_quota, 10) || 0;
  const remain = parseInt(record.quota, 10) || 0;
  const total = used + remain;
  const percent = total > 0 ? (remain / total) * 100 : 0;
  const popoverContent = (
    <div className='text-xs p-2'>
      <Paragraph copyable={{ content: renderQuota(used) }}>
        {t('已用额度')}: {renderQuota(used)}
      </Paragraph>
      <Paragraph copyable={{ content: renderQuota(remain) }}>
        {t('剩余额度')}: {renderQuota(remain)} ({percent.toFixed(0)}%)
      </Paragraph>
      <Paragraph copyable={{ content: renderQuota(total) }}>
        {t('总额度')}: {renderQuota(total)}
      </Paragraph>
    </div>
  );
  return (
    <Popover content={popoverContent} position='top'>
      <Tag color='white' shape='circle'>
        <div className='flex flex-col items-end'>
          <span className='text-xs leading-none'>
            {`${renderQuota(remain)} / ${renderQuota(total)}`}
          </span>
          <Progress
            percent={percent}
            aria-label='quota usage'
            format={() => `${percent.toFixed(0)}%`}
            style={{ width: '100%', marginTop: '1px', marginBottom: 0 }}
          />
        </div>
      </Tag>
    </Popover>
  );
};

const renderInviteInfo = (record, t) => {
  return (
    <div>
      <Space spacing={1}>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('邀请')}: {renderNumber(record.aff_count)}
        </Tag>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('收益')}: {renderQuota(record.aff_history_quota)}
        </Tag>
        <Tag color='white' shape='circle' className='!text-xs'>
          {record.inviter_id === 0
            ? t('无邀请人')
            : `${t('邀请人')}: ${record.inviter_id}`}
        </Tag>
      </Space>
    </div>
  );
};

const renderOperations = (
  record,
  {
    setEditingUser,
    setShowEditUser,
    showEnableDisableModal,
    showDeleteModal,
    showResetPasskeyModal,
    showResetTwoFAModal,
    showUserSubscriptionsModal,
    t,
  },
) => {
  if (record.DeletedAt !== null) {
    return <></>;
  }

  const canManageOps = hasAnyPermission('ops.manage', 'system.manage');
  const canWriteFinance = hasAnyPermission('finance.write', 'system.manage');
  const canViewUserEditor = canManageOps || canWriteFinance;

  if (!canViewUserEditor) {
    return <></>;
  }

  const financeOnly = canWriteFinance && !canManageOps;
  const moreMenu = canManageOps
    ? [
        {
          node: 'item',
          name: t('订阅管理'),
          onClick: () => showUserSubscriptionsModal(record),
        },
        {
          node: 'divider',
        },
        {
          node: 'item',
          name: t('重置 Passkey'),
          onClick: () => showResetPasskeyModal(record),
        },
        {
          node: 'item',
          name: t('重置 2FA'),
          onClick: () => showResetTwoFAModal(record),
        },
        {
          node: 'divider',
        },
        {
          node: 'item',
          name: t('注销'),
          type: 'danger',
          onClick: () => showDeleteModal(record),
        },
      ]
    : [];

  return (
    <Space>
      {canManageOps ? (
        record.status === 1 ? (
          <Button
            type='danger'
            size='small'
            onClick={() => showEnableDisableModal(record, 'disable')}
          >
            {t('禁用')}
          </Button>
        ) : (
          <Button
            size='small'
            onClick={() => showEnableDisableModal(record, 'enable')}
          >
            {t('启用')}
          </Button>
        )
      ) : null}
      <Button
        type={financeOnly ? 'primary' : 'tertiary'}
        size='small'
        onClick={() => {
          setEditingUser(record);
          setShowEditUser(true);
        }}
      >
        {financeOnly ? t('查看/调账') : t('编辑')}
      </Button>
      {canManageOps ? (
        <Dropdown menu={moreMenu} trigger='click' position='bottomRight'>
          <Button type='tertiary' size='small' icon={<IconMore />} />
        </Dropdown>
      ) : null}
    </Space>
  );
};

export const getUsersColumns = ({
  t,
  setEditingUser,
  setShowEditUser,
  showEnableDisableModal,
  showDeleteModal,
  showResetPasskeyModal,
  showResetTwoFAModal,
  showUserSubscriptionsModal,
}) => {
  return [
    {
      title: 'ID',
      dataIndex: 'id',
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
      render: (text, record) => renderUsername(text, record),
    },
    {
      title: t('状态'),
      dataIndex: 'info',
      render: (text, record) => renderStatistics(record, t),
    },
    {
      title: t('剩余额度/总额度'),
      key: 'quota_usage',
      render: (text, record) => renderQuotaUsage(record, t),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      render: (text) => <div>{renderGroup(text)}</div>,
    },
    {
      title: t('后台职责'),
      dataIndex: 'staff_role',
      render: (text, record) => <div>{renderRole(record, t)}</div>,
    },
    {
      title: t('邀请信息'),
      dataIndex: 'invite',
      render: (text, record) => renderInviteInfo(record, t),
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      width: 220,
      render: (text, record) =>
        renderOperations(record, {
          setEditingUser,
          setShowEditUser,
          showEnableDisableModal,
          showDeleteModal,
          showResetPasskeyModal,
          showResetTwoFAModal,
          showUserSubscriptionsModal,
          t,
        }),
    },
  ];
};
