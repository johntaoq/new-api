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
import React, { useEffect, useState } from 'react';
import dayjs from 'dayjs';
import {
  Modal,
  Table,
  Empty,
  Button,
  DatePicker,
  Card,
  Typography,
  Space,
  Tag,
  Toast,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { IconDownload, IconRefresh } from '@douyinfe/semi-icons';
import { API, timestamp2string } from '../../../helpers';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text } = Typography;

const getCurrentBillMonth = () => {
  const now = new Date();
  const month = `${now.getMonth() + 1}`.padStart(2, '0');
  return `${now.getFullYear()}-${month}`;
};

const getMonthPickerValue = (billMonth) => {
  if (!billMonth) return undefined;
  const [year, month] = `${billMonth}`
    .split('-')
    .map((part) => Number.parseInt(part, 10));
  if (!Number.isFinite(year) || !Number.isFinite(month) || month < 1 || month > 12) {
    return undefined;
  }
  return new Date(year, month - 1, 1);
};

const getMonthPickerNextValue = (value) => {
  if (!value) return getCurrentBillMonth();
  if (typeof value === 'string') return value;
  return dayjs(value).format('YYYY-MM');
};

const formatMoney = (value, prefix = '') => {
  const amount = Number(value || 0);
  return `${prefix}${amount.toFixed(6).replace(/\.?0+$/, '')}`;
};

const entryTypeColorMap = {
  consume: 'orange',
  refund: 'green',
  topup: 'cyan',
  gift: 'violet',
  adjustment: 'red',
};

const entryTypeLabelMap = {
  consume: '消费',
  refund: '退款',
  topup: '充值',
  gift: '赠送',
  adjustment: '调整',
};

const extractFilenameFromDisposition = (disposition) => {
  if (!disposition) return '';

  const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1]);
    } catch (error) {
      return utf8Match[1];
    }
  }

  const plainMatch = disposition.match(/filename="?([^";]+)"?/i);
  return plainMatch?.[1] || '';
};

const MonthlyStatementModal = ({ visible, onCancel, t }) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [regenerating, setRegenerating] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [billMonth, setBillMonth] = useState(getCurrentBillMonth());
  const [billMonthInput, setBillMonthInput] = useState(getCurrentBillMonth());
  const [statement, setStatement] = useState(null);
  const [items, setItems] = useState([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  const loadStatement = async (
    targetPage = page,
    targetPageSize = pageSize,
    targetBillMonth = billMonth,
  ) => {
    setLoading(true);
    try {
      const res = await API.get(
        `/api/billing/self?bill_month=${encodeURIComponent(
          targetBillMonth,
        )}&p=${targetPage}&page_size=${targetPageSize}`,
      );
      const { success, message, data } = res.data;
      if (!success) {
        Toast.error({ content: message || t('加载月账单失败') });
        return;
      }
      setStatement(data.statement || null);
      setItems(data.items?.items || []);
      setTotal(data.items?.total || 0);
    } catch (error) {
      Toast.error({ content: t('加载月账单失败') });
    } finally {
      setLoading(false);
    }
  };

  const regenerateStatement = async () => {
    setRegenerating(true);
    try {
      const res = await API.post(
        `/api/billing/self/generate?bill_month=${encodeURIComponent(
          billMonth,
        )}&p=${page}&page_size=${pageSize}`,
      );
      const { success, message, data } = res.data;
      if (!success) {
        Toast.error({ content: message || t('重新生成月账单失败') });
        return;
      }
      setStatement(data.statement || null);
      setItems(data.items?.items || []);
      setTotal(data.items?.total || 0);
      Toast.success({ content: t('月账单已更新') });
    } catch (error) {
      Toast.error({ content: t('重新生成月账单失败') });
    } finally {
      setRegenerating(false);
    }
  };

  const exportStatement = async () => {
    setExporting(true);
    try {
      const response = await API.get(
        `/api/billing/self/export?bill_month=${encodeURIComponent(billMonth)}`,
        {
          responseType: 'blob',
          disableDuplicate: true,
          skipErrorHandler: true,
        },
      );
      const filename =
        extractFilenameFromDisposition(
          response.headers?.['content-disposition'],
        ) || `statement-${billMonth}.csv`;
      const blob = new Blob([response.data], {
        type: 'text/csv;charset=utf-8',
      });
      const downloadUrl = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = downloadUrl;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(downloadUrl);
      Toast.success({ content: t('导出 CSV 成功') });
    } catch (error) {
      let errorMessage = t('导出 CSV 失败');
      if (error?.response?.data instanceof Blob) {
        try {
          const text = await error.response.data.text();
          const payload = JSON.parse(text);
          errorMessage =
            payload?.message || payload?.error?.message || errorMessage;
        } catch (parseError) {}
      }
      Toast.error({ content: errorMessage });
    } finally {
      setExporting(false);
    }
  };

  useEffect(() => {
    if (visible) {
      setBillMonthInput(billMonth);
      loadStatement(1, pageSize, billMonth);
      setPage(1);
    }
  }, [visible]);

  useEffect(() => {
    if (visible) {
      loadStatement(page, pageSize, billMonth);
    }
  }, [page, pageSize]);

  const handleQuery = () => {
    const nextBillMonth = billMonthInput || getCurrentBillMonth();
    setBillMonth(nextBillMonth);
    setPage(1);
    loadStatement(1, pageSize, nextBillMonth);
  };

  const displaySymbol = statement?.currency_symbol_snapshot || '';
  const isTokenDisplay =
    statement?.currency_display_type_snapshot === 'TOKENS';

  const columns = [
    {
      title: t('时间'),
      dataIndex: 'occurred_at',
      key: 'occurred_at',
      render: (value) => timestamp2string(value),
    },
    {
      title: t('令牌'),
      key: 'token',
      render: (_, record) => (
        <div>
          <div>{record.token_name_snapshot || '-'}</div>
          <Text type='tertiary' size='small'>
            {record.token_masked || '-'}
          </Text>
        </div>
      ),
    },
    {
      title: t('消费类型'),
      dataIndex: 'entry_type',
      key: 'entry_type',
      render: (value, record) => (
        <Space spacing={4}>
          <Tag color={entryTypeColorMap[value] || 'grey'}>
            {t(entryTypeLabelMap[value] || value)}
          </Tag>
          {record.operation_type ? (
            <Text type='tertiary'>{record.operation_type}</Text>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('模型'),
      dataIndex: 'model_name',
      key: 'model_name',
      render: (value) => value || '-',
    },
    {
      title: t('COS币变动'),
      dataIndex: 'display_currency_amount',
      key: 'display_currency_amount',
      render: (value) =>
        isTokenDisplay
          ? `${formatMoney(value)}`
          : `${displaySymbol}${formatMoney(value)}`,
    },
    {
      title: t('等价 USD'),
      dataIndex: 'usd_amount',
      key: 'usd_amount',
      render: (value) => formatMoney(value, '$'),
    },
    {
      title: t('请求 ID'),
      dataIndex: 'request_id',
      key: 'request_id',
      render: (value) =>
        value ? <Text copyable>{value}</Text> : <Text type='tertiary'>-</Text>,
    },
    {
      title: t('说明'),
      dataIndex: 'content_summary',
      key: 'content_summary',
      render: (value) => value || '-',
    },
  ];

  return (
    <Modal
      title={t('月账单')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size={isMobile ? 'full-width' : 'large'}
    >
      <div className='mb-4 flex flex-col gap-3 md:flex-row md:items-center md:justify-between'>
        <DatePicker
          type='month'
          value={getMonthPickerValue(billMonthInput)}
          inputReadOnly
          onChange={(value) => setBillMonthInput(getMonthPickerNextValue(value))}
          style={{ maxWidth: 220 }}
        />
        <Space>
          <Button theme='outline' onClick={handleQuery}>
            {t('查询')}
          </Button>
          <Button
            theme='outline'
            icon={<IconDownload />}
            loading={exporting}
            onClick={exportStatement}
          >
            {t('导出 CSV')}
          </Button>
          <Button
            icon={<IconRefresh />}
            loading={regenerating}
            onClick={regenerateStatement}
          >
            {t('重新生成')}
          </Button>
        </Space>
      </div>

      {statement ? (
        <div className='mb-4 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6'>
          <Card bodyStyle={{ padding: 16 }}>
            <Text type='tertiary'>{t('本月消费')}</Text>
            <div className='text-lg font-semibold mt-2'>
              {isTokenDisplay
                ? formatMoney(statement.total_consume_display_amount)
                : `${displaySymbol}${formatMoney(
                    statement.total_consume_display_amount,
                  )}`}
            </div>
          </Card>
          <Card bodyStyle={{ padding: 16 }}>
            <Text type='tertiary'>{t('本月退款')}</Text>
            <div className='text-lg font-semibold mt-2 text-green-600'>
              {isTokenDisplay
                ? formatMoney(statement.total_refund_display_amount)
                : `${displaySymbol}${formatMoney(
                    statement.total_refund_display_amount,
                  )}`}
            </div>
          </Card>
          <Card bodyStyle={{ padding: 16 }}>
            <Text type='tertiary'>{t('本月充值')}</Text>
            <div className='text-lg font-semibold mt-2'>
              {isTokenDisplay
                ? formatMoney(statement.total_topup_display_amount)
                : `${displaySymbol}${formatMoney(
                    statement.total_topup_display_amount,
                  )}`}
            </div>
          </Card>
          <Card bodyStyle={{ padding: 16 }}>
            <Text type='tertiary'>{t('本月赠送')}</Text>
            <div className='text-lg font-semibold mt-2'>
              {isTokenDisplay
                ? formatMoney(statement.total_gift_display_amount)
                : `${displaySymbol}${formatMoney(
                    statement.total_gift_display_amount,
                  )}`}
            </div>
          </Card>
          <Card bodyStyle={{ padding: 16 }}>
            <Text type='tertiary'>{t('本月调整')}</Text>
            <div className='text-lg font-semibold mt-2'>
              {isTokenDisplay
                ? formatMoney(statement.total_adjustment_display_amount)
                : `${displaySymbol}${formatMoney(
                    statement.total_adjustment_display_amount,
                  )}`}
            </div>
          </Card>
          <Card bodyStyle={{ padding: 16 }}>
            <Text type='tertiary'>{t('本月净变动')}</Text>
            <div className='text-lg font-semibold mt-2'>
              {isTokenDisplay
                ? formatMoney(statement.total_net_display_amount)
                : `${displaySymbol}${formatMoney(
                    statement.total_net_display_amount,
                  )}`}
              <Text type='tertiary' className='block mt-1'>
                {formatMoney(statement.total_net_usd, '$')}
              </Text>
            </div>
          </Card>
        </div>
      ) : null}

      <Table
        columns={columns}
        dataSource={items}
        loading={loading}
        rowKey='id'
        pagination={{
          currentPage: page,
          pageSize: pageSize,
          total: total,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50],
          onPageChange: setPage,
          onPageSizeChange: (size) => {
            setPageSize(size);
            setPage(1);
          },
        }}
        size='small'
        empty={
          <Empty
            image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
            darkModeImage={
              <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
            }
            description={t('暂无月账单明细')}
            style={{ padding: 30 }}
          />
        }
      />
    </Modal>
  );
};

export default MonthlyStatementModal;

