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
import { Button, Dropdown, Popover, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { IconMore } from '@douyinfe/semi-icons';
import { renderQuota, timestamp2string } from '../../../helpers';
import {
  REDEMPTION_ACTIONS,
  REDEMPTION_STATUS,
  REDEMPTION_STATUS_MAP,
} from '../../../constants/redemption.constants';

const { Text } = Typography;

const formatUSD = (value) => {
  const amount = Number(value || 0);
  if (!Number.isFinite(amount)) {
    return '$0.000000';
  }
  return `$${amount.toFixed(6)}`;
};

export const isExpired = (record) => {
  return (
    record.status === REDEMPTION_STATUS.UNUSED &&
    record.expired_time !== 0 &&
    record.expired_time < Math.floor(Date.now() / 1000)
  );
};

const renderTimestamp = (timestamp) => {
  return <>{timestamp2string(timestamp)}</>;
};

const renderStatus = (status, record, t) => {
  if (isExpired(record)) {
    return (
      <Tag color='orange' shape='circle'>
        {t('已过期')}
      </Tag>
    );
  }

  const statusConfig = REDEMPTION_STATUS_MAP[status];
  if (statusConfig) {
    return (
      <Tag color={statusConfig.color} shape='circle'>
        {t(statusConfig.text)}
      </Tag>
    );
  }

  return (
    <Tag color='black' shape='circle'>
      {t('未知状态')}
    </Tag>
  );
};

const renderFundingType = (record, t) => {
  const isPaid = record.funding_type === 'paid';
  return (
    <Tag color={isPaid ? 'blue' : 'green'} shape='circle'>
      {isPaid ? t('付费') : t('免费')}
    </Tag>
  );
};

const renderAmountInfo = (record, t) => {
  const isPaid = record.funding_type === 'paid';
  const recognizedRevenue = Number(record.recognized_revenue_usd || 0);
  return (
    <Space vertical align='start' spacing='tight'>
      <Tag color='grey' shape='circle'>
        {formatUSD(record.amount_usd)}
      </Tag>
      {isPaid && recognizedRevenue > 0 && (
        <Text type='secondary' size='small'>
          {`${t('确认收入')} ${formatUSD(recognizedRevenue)}`}
        </Text>
      )}
    </Space>
  );
};

const renderRemark = (record, t) => {
  if (!record.remark) {
    return <Text type='tertiary'>{t('无')}</Text>;
  }
  return (
    <Popover
      content={
        <div style={{ maxWidth: 320, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
          {record.remark}
        </div>
      }
      position='top'
    >
      <Text
        style={{
          maxWidth: 180,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          display: 'inline-block',
          cursor: 'pointer',
        }}
      >
        {record.remark}
      </Text>
    </Popover>
  );
};

export const getRedemptionsColumns = ({
  t,
  manageRedemption,
  copyText,
  setEditingRedemption,
  setShowEdit,
  showDeleteRedemptionModal,
}) => {
  return [
    {
      title: t('ID'),
      dataIndex: 'id',
    },
    {
      title: t('名称'),
      dataIndex: 'name',
      minWidth: 160,
    },
    {
      title: t('类型'),
      dataIndex: 'funding_type',
      render: (text, record) => renderFundingType(record, t),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
      render: (text, record) => {
        return <div>{renderStatus(text, record, t)}</div>;
      },
    },
    {
      title: t('面值 USD'),
      dataIndex: 'amount_usd',
      minWidth: 150,
      render: (text, record) => renderAmountInfo(record, t),
    },
    {
      title: t('平台显示值'),
      dataIndex: 'quota',
      render: (text) => {
        return (
          <Tag color='light-blue' shape='circle'>
            {renderQuota(Number(text) || 0)}
          </Tag>
        );
      },
    },
    {
      title: t('备注'),
      dataIndex: 'remark',
      minWidth: 180,
      render: (text, record) => renderRemark(record, t),
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_time',
      render: (text) => {
        return <div>{renderTimestamp(text)}</div>;
      },
    },
    {
      title: t('过期时间'),
      dataIndex: 'expired_time',
      render: (text) => {
        return <div>{text === 0 ? t('永不过期') : renderTimestamp(text)}</div>;
      },
    },
    {
      title: t('兑换用户 ID'),
      dataIndex: 'used_user_id',
      render: (text) => {
        return <div>{text === 0 ? t('未使用') : text}</div>;
      },
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      width: 205,
      render: (text, record) => {
        const moreMenuItems = [
          {
            node: 'item',
            name: t('删除'),
            type: 'danger',
            onClick: () => {
              showDeleteRedemptionModal(record);
            },
          },
        ];

        if (record.status === REDEMPTION_STATUS.UNUSED && !isExpired(record)) {
          moreMenuItems.push({
            node: 'item',
            name: t('禁用'),
            type: 'warning',
            onClick: () => {
              manageRedemption(record.id, REDEMPTION_ACTIONS.DISABLE, record);
            },
          });
        } else if (!isExpired(record)) {
          moreMenuItems.push({
            node: 'item',
            name: t('启用'),
            type: 'secondary',
            onClick: () => {
              manageRedemption(record.id, REDEMPTION_ACTIONS.ENABLE, record);
            },
            disabled: record.status === REDEMPTION_STATUS.USED,
          });
        }

        return (
          <Space>
            <Popover
              content={record.key}
              style={{ padding: 20 }}
              position='top'
            >
              <Button type='tertiary' size='small'>
                {t('查看')}
              </Button>
            </Popover>
            <Button
              size='small'
              onClick={async () => {
                await copyText(record.key);
              }}
            >
              {t('复制')}
            </Button>
            <Button
              type='tertiary'
              size='small'
              onClick={() => {
                setEditingRedemption(record);
                setShowEdit(true);
              }}
              disabled={record.status !== REDEMPTION_STATUS.UNUSED}
            >
              {t('编辑')}
            </Button>
            <Dropdown
              trigger='click'
              position='bottomRight'
              menu={moreMenuItems}
            >
              <Button type='tertiary' size='small' icon={<IconMore />} />
            </Dropdown>
          </Space>
        );
      },
    },
  ];
};
