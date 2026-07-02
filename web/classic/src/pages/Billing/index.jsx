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
  Button,
  Card,
  DatePicker,
  Empty,
  Input,
  Pagination,
  Select,
  Spin,
  TabPane,
  Tabs,
  Tag,
} from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import {
  Download,
  LayoutDashboard,
  Receipt,
  Shield,
  Users,
  WalletCards,
} from 'lucide-react';
import CardTable from '../../components/common/ui/CardTable';
import {
  API,
  hasPermission,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';
import { CHART_CONFIG } from '../../constants/dashboard.constants';
import './index.css';

const DEFAULT_PAGE_SIZE = 10;

const createPageState = (pageSize = DEFAULT_PAGE_SIZE) => ({
  page: 1,
  pageSize,
  total: 0,
  items: [],
});

const formatMoney = (value) => {
  const amount = Number(value || 0);
  return `$${amount.toFixed(2)}`;
};

const formatCount = (value) => Number(value || 0).toLocaleString();

const formatCOS = (value) =>
  Number(value || 0).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });

const formatTokens = (value) => Number(value || 0).toLocaleString();

const getCurrentPeriodValue = (periodType) =>
  periodType === 'year' ? dayjs().format('YYYY') : dayjs().format('YYYY-MM');

const getPickerType = (periodType) => (periodType === 'year' ? 'year' : 'month');

const getPickerValue = (periodType, period) => {
  if (!period) return undefined;
  if (periodType === 'year') {
    const year = Number.parseInt(`${period}`, 10);
    return Number.isFinite(year) ? new Date(year, 0, 1) : undefined;
  }
  const [year, month] = `${period}`.split('-').map((part) => Number.parseInt(part, 10));
  if (!Number.isFinite(year) || !Number.isFinite(month) || month < 1 || month > 12) {
    return undefined;
  }
  return new Date(year, month - 1, 1);
};

const getPickerNextValue = (periodType, value) => {
  if (!value) return getCurrentPeriodValue(periodType);
  if (typeof value === 'string') return value;
  return dayjs(value).format(periodType === 'year' ? 'YYYY' : 'YYYY-MM');
};

const getYearOptionList = (selectedYear) => {
  const currentYear = dayjs().year();
  const normalizedSelected = Number.parseInt(`${selectedYear || currentYear}`, 10);
  const anchorYear = Number.isFinite(normalizedSelected)
    ? normalizedSelected
    : currentYear;
  const minYear = Math.min(currentYear, anchorYear) - 6;
  const maxYear = Math.max(currentYear, anchorYear) + 1;
  const options = [];
  for (let year = maxYear; year >= minYear; year -= 1) {
    options.push({
      label: `${year}`,
      value: `${year}`,
    });
  }
  return options;
};

const getCustomCurrencyRate = () => {
  try {
    const statusStr = localStorage.getItem('status');
    if (statusStr) {
      const status = JSON.parse(statusStr);
      const rate = Number(status?.custom_currency_exchange_rate || 1);
      if (Number.isFinite(rate) && rate > 0) {
        return rate;
      }
    }
  } catch (e) {}
  return 1;
};

const getCosValue = (cos, usd) => {
  if (cos !== undefined && cos !== null) {
    return Number(cos || 0);
  }
  return Number(usd || 0) * getCustomCurrencyRate();
};

const getAmountText = (unit, usd, cos) =>
  unit === 'usd' ? formatMoney(usd) : formatCOS(getCosValue(cos, usd));

const getRankingValueText = (item, rankingView, unit) => {
  if (unit === 'usd') {
    return formatMoney(item?.value_usd);
  }
  if (rankingView === 'channel_cost') {
    return `${formatTokens(item?.value_tokens)} tokens`;
  }
  return formatCOS(getCosValue(item?.value_cos, item?.value_usd));
};

const dashboardTodoTypeLabelMap = {
  balance_alert: '余额告警',
  inactive_balance: '有余额未调用',
};

const billEntryTypeLabelMap = {
  consume: '消费',
  refund: '退款',
  topup: '充值',
  gift: '赠送',
  adjustment: '调整',
};

const billEntryTypeColorMap = {
  consume: 'orange',
  refund: 'green',
  topup: 'cyan',
  gift: 'violet',
  adjustment: 'red',
};

const activeTabLabelMap = {
  dashboard: '营运看板',
  revenue: '收入与赠送',
  channel: '渠道成本',
  customer: '客户账单',
  audit: '财务审计',
};

const auditModuleLabelMap = {
  redemption: '兑换码管理',
  user_quota: '用户额度调整',
};

const auditActionLabelMap = {
  create: '创建',
  update: '修改',
  delete: '删除',
  cleanup_invalid: '清理失效',
  adjust: '调整',
};

const auditActionColorMap = {
  create: 'green',
  update: 'cyan',
  delete: 'red',
  cleanup_invalid: 'orange',
  adjust: 'violet',
};

const auditTargetLabelMap = {
  redemption: '兑换码',
  redemption_batch: '批量兑换码',
  user_wallet: '用户钱包',
};

const formatAuditJSON = (value) => {
  if (!value) {
    return '-';
  }
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch (error) {
    return `${value}`;
  }
};

const unwrapResponse = (response) => {
  const { success, message, data } = response.data || {};
  if (!success) {
    throw new Error(message || '请求失败');
  }
  return data;
};

const buildPieSpec = (data, title) => ({
  type: 'pie',
  data: [{ id: 'finance-pie', values: data }],
  outerRadius: 0.82,
  innerRadius: 0.54,
  padAngle: 0.8,
  valueField: 'value',
  categoryField: 'type',
  legends: {
    visible: true,
    orient: 'bottom',
    item: { label: { style: { fill: '#cbd5f5', fontSize: 12 } } },
  },
  label: { visible: true, style: { fontSize: 12, fill: '#f8fafc' } },
  title: {
    visible: true,
    text: title,
    textStyle: { fill: '#e2e8f0', fontSize: 14, fontWeight: 600 },
  },
  pie: {
    style: { cornerRadius: 8 },
    state: {
      hover: { outerRadius: 0.86, stroke: '#93c5fd', lineWidth: 1.2 },
    },
  },
});

const MetricCard = ({ icon: Icon, title, value, helper, tone = 'slate' }) => {
  const toneClass = {
    slate:
      'border-cyan-400/20 from-[#121e5f]/95 via-[#0d163f]/96 to-[#09112e]/98 shadow-[0_22px_60px_rgba(34,211,238,0.16)]',
    emerald:
      'border-sky-400/20 from-[#10235e]/95 via-[#0d1b49]/96 to-[#0a122f]/98 shadow-[0_22px_60px_rgba(56,189,248,0.16)]',
    amber:
      'border-fuchsia-400/20 from-[#231759]/95 via-[#15133f]/96 to-[#0b102d]/98 shadow-[0_22px_60px_rgba(217,70,239,0.18)]',
    sky:
      'border-indigo-400/20 from-[#1c215c]/95 via-[#121744]/96 to-[#09102b]/98 shadow-[0_22px_60px_rgba(99,102,241,0.18)]',
    rose:
      'border-pink-400/20 from-[#281556]/95 via-[#1b123f]/96 to-[#0c102d]/98 shadow-[0_22px_60px_rgba(236,72,153,0.18)]',
  }[tone];

  return (
    <Card
      bordered
      className={`!rounded-[28px] border bg-gradient-to-br backdrop-blur-sm ${toneClass || ''}`}
      bodyStyle={{ padding: 18, position: 'relative', overflow: 'hidden' }}
    >
      <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(56,189,248,0.14),transparent_30%),radial-gradient(circle_at_bottom_left,rgba(236,72,153,0.12),transparent_28%)]' />
      <div className='pointer-events-none absolute inset-x-6 top-0 h-px bg-gradient-to-r from-transparent via-cyan-300/80 to-transparent' />
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <div className='text-[11px] uppercase tracking-[0.28em] text-slate-400'>{title}</div>
          <div className='mt-3 text-2xl font-semibold break-all text-slate-50'>{value}</div>
          {helper ? (
            <div className='mt-2 text-xs text-slate-400 break-all leading-5'>{helper}</div>
          ) : null}
        </div>
        <div className='rounded-2xl border border-cyan-300/20 bg-white/5 p-3 text-cyan-200 shadow-[0_10px_30px_rgba(8,47,73,0.35)] backdrop-blur-sm'>
          <Icon size={18} strokeWidth={2.2} />
        </div>
      </div>
    </Card>
  );
};

const TableCard = ({
  title,
  extra,
  loading,
  columns,
  dataSource,
  rowKey,
  pagination,
  onPageChange,
  onPageSizeChange,
}) => (
  <Card
    bordered
    className='!rounded-[28px] border border-cyan-400/15 bg-[linear-gradient(180deg,rgba(17,26,72,0.94),rgba(8,13,36,0.98))] shadow-[0_26px_70px_rgba(2,6,23,0.48)] backdrop-blur-xl'
    title={<span className='text-base font-semibold tracking-[0.08em] text-slate-100'>{title}</span>}
    headerExtraContent={extra}
    bodyStyle={{ padding: 0, position: 'relative', overflow: 'hidden' }}
  >
    <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(56,189,248,0.10),transparent_20%),radial-gradient(circle_at_bottom_right,rgba(217,70,239,0.08),transparent_22%)]' />
    <div className='pointer-events-none absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-cyan-300/70 to-transparent' />
    <CardTable
      columns={columns}
      dataSource={dataSource}
      loading={loading}
      rowKey={rowKey}
      pagination={false}
    />
    {pagination && pagination.total > 0 ? (
      <div className='flex justify-end border-t border-white/10 px-4 py-3'>
        <Pagination
          currentPage={pagination.page}
          pageSize={pagination.pageSize}
          total={pagination.total}
          showSizeChanger
          pageSizeOpts={[10, 20, 50, 100]}
          onPageChange={onPageChange}
          onPageSizeChange={onPageSizeChange}
        />
      </div>
    ) : null}
  </Card>
);

const ChartCard = ({ title, spec, loading, hasData }) => (
  <Card
    bordered
    className='!rounded-[28px] border border-fuchsia-400/15 bg-[linear-gradient(180deg,rgba(23,21,76,0.94),rgba(8,11,38,0.98))] shadow-[0_26px_70px_rgba(2,6,23,0.48)] backdrop-blur-xl'
    title={<span className='text-base font-semibold tracking-[0.08em] text-slate-100'>{title}</span>}
    bodyStyle={{ padding: 0, position: 'relative', overflow: 'hidden' }}
  >
    <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(236,72,153,0.10),transparent_22%),radial-gradient(circle_at_bottom_left,rgba(59,130,246,0.10),transparent_22%)]' />
    <div className='pointer-events-none absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-fuchsia-300/70 to-transparent' />
    <div className='h-[340px] p-3'>
      {loading ? (
        <div className='flex h-full items-center justify-center'>
          <Spin spinning />
        </div>
      ) : hasData ? (
        <VChart spec={spec} option={CHART_CONFIG} />
      ) : (
        <div className='flex h-full items-center justify-center'>
          <Empty description='暂无图表数据' />
        </div>
      )}
    </div>
  </Card>
);

const Billing = () => {
  const canWriteFinance = hasPermission('finance.write') || hasPermission('system.manage');
  const canViewAudit = hasPermission('finance.audit.view') || hasPermission('system.manage');
  const [activeTab, setActiveTab] = useState('dashboard');
  const [periodType, setPeriodType] = useState('month');
  const [periodValue, setPeriodValue] = useState(getCurrentPeriodValue('month'));
  const [dashboardUnit, setDashboardUnit] = useState('usd');
  const [rankingView, setRankingView] = useState('income');
  const [dashboardLoading, setDashboardLoading] = useState(false);
  const [dashboardSummary, setDashboardSummary] = useState(null);
  const [dashboardTodos, setDashboardTodos] = useState([]);
  const [dashboardRankings, setDashboardRankings] = useState([]);

  const [revenueUnit, setRevenueUnit] = useState('usd');
  const [paidSourceView, setPaidSourceView] = useState('summary');
  const [revenueLoading, setRevenueLoading] = useState(false);
  const [revenueSummary, setRevenueSummary] = useState(null);
  const [paidSourcesPage, setPaidSourcesPage] = useState(createPageState());
  const [giftAuditView, setGiftAuditView] = useState('summary');
  const [giftAuditSummaryPage, setGiftAuditSummaryPage] = useState(
    createPageState(),
  );
  const [giftAuditPage, setGiftAuditPage] = useState(createPageState());

  const [channelMetric, setChannelMetric] = useState('usd');
  const [channelLoading, setChannelLoading] = useState(false);
  const [channelSummary, setChannelSummary] = useState(null);
  const [channelPage, setChannelPage] = useState(createPageState());
  const [modelPage, setModelPage] = useState(createPageState());

  const [customerUnit, setCustomerUnit] = useState('usd');
  const [billMonthInput, setBillMonthInput] = useState(dayjs().format('YYYY-MM'));
  const [billKeywordInput, setBillKeywordInput] = useState('');
  const [billQuery, setBillQuery] = useState({
    billMonth: dayjs().format('YYYY-MM'),
    userKeyword: '',
  });
  const [customerLoading, setCustomerLoading] = useState(false);
  const [customerDetailLoading, setCustomerDetailLoading] = useState(false);
  const [customerGenerating, setCustomerGenerating] = useState(false);
  const [customerRefreshVersion, setCustomerRefreshVersion] = useState(0);
  const [customerSummaryPage, setCustomerSummaryPage] = useState(
    createPageState(),
  );
  const [customerBillDetails, setCustomerBillDetails] = useState([]);
  const [selectedBillUserId, setSelectedBillUserId] = useState(null);
  const [selectedBillUserName, setSelectedBillUserName] = useState('');

  const [auditLoading, setAuditLoading] = useState(false);
  const [auditModuleInput, setAuditModuleInput] = useState('');
  const [auditActionInput, setAuditActionInput] = useState('');
  const [auditOperatorInput, setAuditOperatorInput] = useState('');
  const [auditTargetInput, setAuditTargetInput] = useState('');
  const [auditQuery, setAuditQuery] = useState({
    module: '',
    action: '',
    operatorKeyword: '',
    targetKeyword: '',
  });
  const [auditPage, setAuditPage] = useState(createPageState());
  const [selectedAuditLog, setSelectedAuditLog] = useState(null);

  const periodParams = {
    period_type: periodType,
    period: periodValue,
  };

  useEffect(() => {
    if (activeTab === 'audit' && !canViewAudit) {
      setActiveTab('dashboard');
    }
  }, [activeTab, canViewAudit]);

  useEffect(() => {
    initVChartSemiTheme({
      isWatchingThemeSwitch: true,
    });
  }, []);

  useEffect(() => {
    if (activeTab === 'dashboard') {
      loadDashboardData();
    }
  }, [activeTab, periodType, periodValue, rankingView]);

  useEffect(() => {
    if (activeTab === 'revenue') {
      loadRevenueData();
    }
  }, [
    activeTab,
    periodType,
    periodValue,
    paidSourceView,
    paidSourcesPage.page,
    paidSourcesPage.pageSize,
    giftAuditSummaryPage.page,
    giftAuditSummaryPage.pageSize,
    giftAuditPage.page,
    giftAuditPage.pageSize,
  ]);

  useEffect(() => {
    if (activeTab === 'channel') {
      loadChannelData();
    }
  }, [
    activeTab,
    periodType,
    periodValue,
    channelPage.page,
    channelPage.pageSize,
    modelPage.page,
    modelPage.pageSize,
  ]);

  useEffect(() => {
    setPaidSourcesPage((prev) => ({ ...prev, page: 1 }));
    setGiftAuditSummaryPage((prev) => ({ ...prev, page: 1 }));
    setGiftAuditPage((prev) => ({ ...prev, page: 1 }));
    setChannelPage((prev) => ({ ...prev, page: 1 }));
    setModelPage((prev) => ({ ...prev, page: 1 }));
    setAuditPage((prev) => ({ ...prev, page: 1 }));
  }, [periodType, periodValue]);

  useEffect(() => {
    if (activeTab === 'customer') {
      loadCustomerSummary();
    }
  }, [
    activeTab,
    billQuery.billMonth,
    billQuery.userKeyword,
    customerSummaryPage.page,
    customerSummaryPage.pageSize,
    customerRefreshVersion,
  ]);

  useEffect(() => {
    if (activeTab === 'customer' && selectedBillUserId) {
      loadCustomerDetails(selectedBillUserId);
    }
  }, [activeTab, selectedBillUserId, billQuery.billMonth, customerRefreshVersion]);

  useEffect(() => {
    if (activeTab === 'audit') {
      loadAuditData();
    }
  }, [
    activeTab,
    periodType,
    periodValue,
    auditQuery.module,
    auditQuery.action,
    auditQuery.operatorKeyword,
    auditQuery.targetKeyword,
    auditPage.page,
    auditPage.pageSize,
  ]);

  const loadDashboardData = async () => {
    setDashboardLoading(true);
    try {
      const [summary, todos, rankings] = await Promise.all([
        API.get('/api/finance/dashboard/summary', { params: periodParams }),
        API.get('/api/finance/dashboard/todos', {
          params: { ...periodParams, limit: 10 },
        }),
        API.get('/api/finance/dashboard/rankings', {
          params: { ...periodParams, limit: 10, view: rankingView },
        }),
      ]);
      setDashboardSummary(unwrapResponse(summary));
      setDashboardTodos(unwrapResponse(todos)?.items || []);
      setDashboardRankings(unwrapResponse(rankings)?.items || []);
    } catch (error) {
      showError(error.message);
    } finally {
      setDashboardLoading(false);
    }
  };

  const loadRevenueData = async () => {
    setRevenueLoading(true);
    try {
      const [summary, paidSources, giftAuditSummary, giftAudit] = await Promise.all([
        API.get('/api/finance/revenue/summary', { params: periodParams }),
        API.get('/api/finance/revenue/paid-sources', {
          params: {
            ...periodParams,
            view: paidSourceView,
            p: paidSourcesPage.page,
            page_size: paidSourcesPage.pageSize,
          },
        }),
        API.get('/api/finance/revenue/gift-audit-summary', {
          params: {
            ...periodParams,
            p: giftAuditSummaryPage.page,
            page_size: giftAuditSummaryPage.pageSize,
          },
        }),
        API.get('/api/finance/revenue/gift-audit', {
          params: {
            ...periodParams,
            p: giftAuditPage.page,
            page_size: giftAuditPage.pageSize,
          },
        }),
      ]);
      setRevenueSummary(unwrapResponse(summary));
      setPaidSourcesPage((prev) => ({ ...prev, ...unwrapResponse(paidSources) }));
      setGiftAuditSummaryPage((prev) => ({
        ...prev,
        ...unwrapResponse(giftAuditSummary),
      }));
      setGiftAuditPage((prev) => ({ ...prev, ...unwrapResponse(giftAudit) }));
    } catch (error) {
      showError(error.message);
    } finally {
      setRevenueLoading(false);
    }
  };

  const loadChannelData = async () => {
    setChannelLoading(true);
    try {
      const [summary, channels, models] = await Promise.all([
        API.get('/api/finance/channel-cost/summary', { params: periodParams }),
        API.get('/api/finance/channel-cost/channels', {
          params: {
            ...periodParams,
            p: channelPage.page,
            page_size: channelPage.pageSize,
          },
        }),
        API.get('/api/finance/channel-cost/models', {
          params: {
            ...periodParams,
            p: modelPage.page,
            page_size: modelPage.pageSize,
          },
        }),
      ]);
      setChannelSummary(unwrapResponse(summary));
      setChannelPage((prev) => ({ ...prev, ...unwrapResponse(channels) }));
      setModelPage((prev) => ({ ...prev, ...unwrapResponse(models) }));
    } catch (error) {
      showError(error.message);
    } finally {
      setChannelLoading(false);
    }
  };

  const loadCustomerSummary = async () => {
    setCustomerLoading(true);
    try {
      const response = await API.get('/api/finance/customer-bills/summary', {
        params: {
          bill_month: billQuery.billMonth,
          user_keyword: billQuery.userKeyword,
          p: customerSummaryPage.page,
          page_size: customerSummaryPage.pageSize,
        },
      });
      const pageData = unwrapResponse(response);
      const nextItems = pageData?.items || [];
      setCustomerSummaryPage((prev) => ({ ...prev, ...pageData }));
      if (nextItems.length === 0) {
        setSelectedBillUserId(null);
        setSelectedBillUserName('');
        setCustomerBillDetails([]);
        return;
      }
      const matchedItem =
        nextItems.find((item) => item.user_id === selectedBillUserId) ||
        nextItems[0];
      setSelectedBillUserId(matchedItem.user_id);
      setSelectedBillUserName(matchedItem.username || '');
    } catch (error) {
      showError(error.message);
    } finally {
      setCustomerLoading(false);
    }
  };

  const loadCustomerDetails = async (userId) => {
    setCustomerDetailLoading(true);
    try {
      const response = await API.get('/api/finance/customer-bills/details', {
        params: {
          bill_month: billQuery.billMonth,
          user_id: userId,
        },
      });
      const data = unwrapResponse(response);
      setCustomerBillDetails(data?.items || []);
    } catch (error) {
      showError(error.message);
    } finally {
      setCustomerDetailLoading(false);
    }
  };

  const handleCustomerGenerate = async () => {
    const nextQuery = {
      billMonth: billMonthInput,
      userKeyword: billKeywordInput.trim(),
    };
    setCustomerGenerating(true);
    try {
      await API.post('/api/finance/customer-bills/generate', null, {
        params: {
          bill_month: nextQuery.billMonth,
          user_keyword: nextQuery.userKeyword,
        },
      });
      setCustomerSummaryPage((prev) => ({ ...prev, page: 1 }));
      setBillQuery(nextQuery);
      setCustomerRefreshVersion((prev) => prev + 1);
      showSuccess('客户账单已重新生成并刷新');
    } catch (error) {
      showError(error.message);
    } finally {
      setCustomerGenerating(false);
    }
  };

  const handleExportCustomerBill = async () => {
    if (!selectedBillUserId) {
      showError('请先选择客户');
      return;
    }
    try {
      const response = await API.get('/api/finance/customer-bills/export', {
        params: {
          bill_month: billQuery.billMonth,
          user_id: selectedBillUserId,
        },
        responseType: 'blob',
      });
      const contentType = response.headers?.['content-type'] || '';
      if (contentType.includes('application/json')) {
        const text = await response.data.text();
        const json = JSON.parse(text);
        throw new Error(json?.message || '导出失败');
      }
      const disposition = response.headers?.['content-disposition'] || '';
      const matched = disposition.match(/filename\*=UTF-8''([^;]+)/);
      const filename = matched
        ? decodeURIComponent(matched[1])
        : `customer-bill-${selectedBillUserId}-${billQuery.billMonth}.csv`;
      const blob = new Blob([response.data], {
        type: 'text/csv;charset=utf-8',
      });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      link.click();
      window.URL.revokeObjectURL(url);
      showSuccess('导出成功');
    } catch (error) {
      showError(error.message);
    }
  };

  const loadAuditData = async () => {
    setAuditLoading(true);
    try {
      const response = await API.get('/api/finance/audit/logs', {
        params: {
          ...periodParams,
          module: auditQuery.module,
          action: auditQuery.action,
          operator_keyword: auditQuery.operatorKeyword,
          target_keyword: auditQuery.targetKeyword,
          p: auditPage.page,
          page_size: auditPage.pageSize,
        },
      });
      const pageData = unwrapResponse(response);
      const nextItems = pageData?.items || [];
      setAuditPage((prev) => ({ ...prev, ...pageData }));
      if (nextItems.length === 0) {
        setSelectedAuditLog(null);
        return;
      }
      const matchedItem =
        nextItems.find((item) => item.id === selectedAuditLog?.id) || nextItems[0];
      setSelectedAuditLog(matchedItem);
    } catch (error) {
      showError(error.message);
    } finally {
      setAuditLoading(false);
    }
  };

  const handleAuditQuery = () => {
    setAuditPage((prev) => ({ ...prev, page: 1 }));
    setAuditQuery({
      module: auditModuleInput,
      action: auditActionInput,
      operatorKeyword: auditOperatorInput.trim(),
      targetKeyword: auditTargetInput.trim(),
    });
  };

  const handleAuditReset = () => {
    setAuditModuleInput('');
    setAuditActionInput('');
    setAuditOperatorInput('');
    setAuditTargetInput('');
    setAuditPage((prev) => ({ ...prev, page: 1 }));
    setAuditQuery({
      module: '',
      action: '',
      operatorKeyword: '',
      targetKeyword: '',
    });
  };

  const dashboardTodoColumns = [
    {
      title: '类型',
      dataIndex: 'type',
      render: (_, record) => (
        <Tag color='orange'>
          {dashboardTodoTypeLabelMap[record.type] || record.type}
        </Tag>
      ),
    },
    {
      title: '对象',
      dataIndex: 'target_name',
      render: (_, record) => `${record.target_name || '-'} (#${record.target_id})`,
    },
    {
      title: '异常值',
      dataIndex: 'abnormal_value',
    },
    {
      title: '建议动作',
      dataIndex: 'suggested_action',
    },
    {
      title: '最后发生时间',
      dataIndex: 'last_occurred_at',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
  ];

  const rankingColumns = [
    {
      title: '排名',
      dataIndex: 'rank',
      width: 72,
    },
    {
      title: '对象',
      dataIndex: 'target_name',
      render: (_, record) => (
        <div>
          <div className='font-medium'>{record.target_name || '-'}</div>
          {record.extra ? (
            <div className='text-xs text-gray-500'>{record.extra}</div>
          ) : null}
        </div>
      ),
    },
    {
      title: '数值',
      dataIndex: 'value_usd',
      render: (_, record) =>
        getRankingValueText(record, rankingView, dashboardUnit),
    },
  ];

  const paidSourceSummaryColumns = [
    {
      title: '来源',
      dataIndex: 'source_label',
    },
    {
      title: '充值次数',
      dataIndex: 'recharge_count',
    },
    {
      title: '充值用户数',
      dataIndex: 'user_count',
    },
    {
      title: '充值额',
      render: (_, record) =>
        getAmountText(
          revenueUnit,
          record.recharge_amount_usd,
          record.recharge_amount_cos,
        ),
    },
    {
      title: '剩余额',
      render: (_, record) =>
        getAmountText(
          revenueUnit,
          record.remaining_amount_usd,
          record.remaining_amount_cos,
        ),
    },
  ];

  const paidSourceDetailColumns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: '渠道',
      dataIndex: 'source_label',
    },
    {
      title: '用户',
      render: (_, record) => `${record.username || '-'} (#${record.user_id})`,
    },
    {
      title: '充值额',
      render: (_, record) =>
        getAmountText(
          revenueUnit,
          record.recharge_amount_usd,
          record.recharge_amount_cos,
        ),
    },
    {
      title: '剩余额',
      render: (_, record) =>
        getAmountText(
          revenueUnit,
          record.remaining_amount_usd,
          record.remaining_amount_cos,
        ),
    },
    {
      title: '备注',
      dataIndex: 'remark',
      render: (value) => value || '-',
    },
  ];

  const giftAuditColumns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: '用户',
      render: (_, record) => `${record.username || '-'} (#${record.user_id})`,
    },
    {
      title: '来源类别',
      dataIndex: 'source_label',
    },
    {
      title: '发放额',
      render: (_, record) =>
        getAmountText(
          revenueUnit,
          record.granted_amount_usd,
          record.granted_amount_cos,
        ),
    },
    {
      title: '操作人',
      render: (_, record) =>
        record.operator_username
          ? `${record.operator_username} (#${record.operator_user_id || '-'})`
          : '-',
    },
    {
      title: '兑换码',
      dataIndex: 'voucher_code',
      render: (value) => value || '-',
    },
    {
      title: '备注',
      dataIndex: 'remark',
      render: (value) => value || '-',
    },
  ];

  const giftAuditSummaryColumns = [
    {
      title: '来源类别',
      dataIndex: 'source_label',
      render: (value) => value || '-',
    },
    {
      title: '活动说明',
      dataIndex: 'activity_description',
      render: (value) => value || '-',
    },
    {
      title: '兑换次数',
      dataIndex: 'exchange_count',
      render: (value) => formatCount(value),
    },
    {
      title: '发放总额',
      render: (_, record) =>
        getAmountText(
          revenueUnit,
          record.granted_amount_usd,
          record.granted_amount_cos,
        ),
    },
  ];

  const channelColumns = [
    {
      title: '渠道',
      render: (_, record) => (
        <div>
          <div className='font-medium'>{record.channel_name || '-'}</div>
          <div className='text-xs text-gray-500'>
            {record.provider_snapshot || '-'}
          </div>
        </div>
      ),
    },
    {
      title: '总调用量',
      dataIndex: 'total_tokens',
      render: (value) => formatTokens(value),
    },
    {
      title: '收入 USD',
      dataIndex: 'revenue_usd',
      render: (value) => formatMoney(value),
    },
    {
      title: '成本 USD',
      dataIndex: 'cost_usd',
      render: (value) => formatMoney(value),
    },
  ];

  const modelColumns = [
    {
      title: '模型',
      render: (_, record) => (
        <div>
          <div className='font-medium'>{record.model_name || '-'}</div>
          <div className='text-xs text-gray-500'>
            {record.channel_name || '-'} / {record.provider_snapshot || '-'}
          </div>
        </div>
      ),
    },
    {
      title: '总调用量',
      dataIndex: 'total_tokens',
      render: (value) => formatTokens(value),
    },
    {
      title: '收入 USD',
      dataIndex: 'revenue_usd',
      render: (value) => formatMoney(value),
    },
    {
      title: '成本 USD',
      dataIndex: 'cost_usd',
      render: (value) => formatMoney(value),
    },
  ];

  const customerSummaryColumns = [
    {
      title: '客户',
      render: (_, record) => (
        <button
          className='text-left'
          onClick={() => {
            setSelectedBillUserId(record.user_id);
            setSelectedBillUserName(record.username || '');
          }}
        >
          <div className='font-medium text-blue-600'>{record.username || '-'}</div>
          <div className='text-xs text-gray-500'>#{record.user_id}</div>
        </button>
      ),
    },
    {
      title: '账户余额',
      render: (_, record) =>
        getAmountText(
          customerUnit,
          record.current_balance_usd,
          record.current_balance_cos,
        ),
    },
    {
      title: '总消耗额',
      render: (_, record) =>
        getAmountText(
          customerUnit,
          record.total_consume_usd,
          record.total_consume_cos,
        ),
    },
    {
      title: '付费消耗',
      render: (_, record) =>
        getAmountText(
          customerUnit,
          record.paid_consume_usd,
          record.paid_consume_cos,
        ),
    },
    {
      title: '赠送消耗',
      render: (_, record) =>
        getAmountText(
          customerUnit,
          record.gift_consume_usd,
          record.gift_consume_cos,
        ),
    },
  ];

  const customerDetailColumns = [
    {
      title: '时间',
      dataIndex: 'occurred_at',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: '令牌',
      dataIndex: 'token_display',
      render: (value) => value || '-',
    },
    {
      title: '消费类型',
      dataIndex: 'entry_type',
      render: (value) => (
        <Tag color={billEntryTypeColorMap[value] || 'blue'}>
          {billEntryTypeLabelMap[value] || value || '-'}
        </Tag>
      ),
    },
    {
      title: '模型',
      dataIndex: 'model_name',
      render: (value) => value || '-',
    },
    {
      title: 'COS币变动',
      dataIndex: 'amount_cos',
      render: (value, record) => formatCOS(getCosValue(value, record?.amount_usd)),
    },
    {
      title: '等价 USD',
      dataIndex: 'amount_usd',
      render: (value) => formatMoney(value),
    },
    {
      title: '渠道',
      dataIndex: 'channel_name',
      render: (value) => value || '-',
    },
    {
      title: '请求 ID',
      dataIndex: 'request_id',
      render: (value) => value || '-',
    },
    {
      title: '说明',
      dataIndex: 'remark',
      render: (value) => value || '-',
    },
  ];

  const auditColumns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: '模块',
      dataIndex: 'module',
      render: (value) => auditModuleLabelMap[value] || value || '-',
    },
    {
      title: '动作',
      dataIndex: 'action',
      render: (value) => (
        <Tag color={auditActionColorMap[value] || 'blue'}>
          {auditActionLabelMap[value] || value || '-'}
        </Tag>
      ),
    },
    {
      title: '操作人',
      render: (_, record) =>
        record.operator_user_id
          ? `${record.operator_username || record.operator_username_snapshot || '-'} (#${record.operator_user_id})`
          : record.operator_username || record.operator_username_snapshot || '-',
    },
    {
      title: '目标对象',
      render: (_, record) => (
        <div>
          <div className='font-medium'>
            {auditTargetLabelMap[record.target_type] || record.target_type || '-'}
          </div>
          <div className='text-xs text-slate-400'>
            {record.target_id ? `#${record.target_id}` : '-'}
          </div>
        </div>
      ),
    },
    {
      title: '目标用户',
      render: (_, record) =>
        record.target_user_id
          ? `${record.target_username || '-'} (#${record.target_user_id})`
          : '-',
    },
    {
      title: '备注',
      dataIndex: 'remark',
      render: (value) => value || '-',
    },
    {
      title: '详情',
      width: 96,
      render: (_, record) => (
        <Button
          theme={selectedAuditLog?.id === record.id ? 'solid' : 'outline'}
          type='primary'
          size='small'
          onClick={() => setSelectedAuditLog(record)}
        >
          查看
        </Button>
      ),
    },
  ];

  const channelChartData =
    channelMetric === 'usd'
      ? channelSummary?.charts?.by_channel_usd || []
      : channelSummary?.charts?.by_channel_tokens || [];
  const modelChartData =
    channelMetric === 'usd'
      ? channelSummary?.charts?.by_model_usd || []
      : channelSummary?.charts?.by_model_tokens || [];
  const dashboardContent = (
    <div className='space-y-4'>
      <div className='grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4'>
        <MetricCard
          icon={Receipt}
          title='销售收入'
          value={getAmountText(
            dashboardUnit,
            dashboardSummary?.kpis?.sales_income?.usd,
            dashboardSummary?.kpis?.sales_income?.cos,
          )}
          helper='当前周期内已确认的付费充值收入'
          tone='emerald'
        />
        <MetricCard
          icon={WalletCards}
          title='赠送额度'
          value={getAmountText(
            dashboardUnit,
            dashboardSummary?.kpis?.gift_granted?.usd,
            dashboardSummary?.kpis?.gift_granted?.cos,
          )}
          helper='当前周期内发放的免费额度总额'
          tone='amber'
        />
        <MetricCard
          icon={LayoutDashboard}
          title='渠道支出'
          value={getAmountText(
            dashboardUnit,
            dashboardSummary?.kpis?.channel_cost?.usd,
            dashboardSummary?.kpis?.channel_cost?.cos,
          )}
          helper='当前周期内按官网口径折算的渠道支出'
          tone='rose'
        />
        <MetricCard
          icon={Users}
          title='客户活跃'
          value={`${dashboardSummary?.kpis?.active_customer_count || 0} / ${dashboardSummary?.kpis?.paid_customer_count || 0}`}
          helper='活跃客户数 / 付费客户数'
          tone='sky'
        />
      </div>

      <div className='grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4'>
        <MetricCard
          icon={WalletCards}
          title='当前付费余额'
          value={getAmountText(
            dashboardUnit,
            dashboardSummary?.kpis?.current_paid_balance?.usd,
            dashboardSummary?.kpis?.current_paid_balance?.cos,
          )}
          tone='slate'
        />
        <MetricCard
          icon={WalletCards}
          title='当前赠送余额'
          value={getAmountText(
            dashboardUnit,
            dashboardSummary?.kpis?.current_gift_balance?.usd,
            dashboardSummary?.kpis?.current_gift_balance?.cos,
          )}
          tone='slate'
        />
      </div>

      <Card bordered className='!rounded-2xl' title='告警概览'>
        <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4'>
          {(dashboardSummary?.alerts || []).map((item) => (
            <div
              key={item.type}
              className='rounded-2xl border border-amber-200 bg-amber-50 p-4'
            >
              <div className='font-medium text-amber-900'>{item.title}</div>
              <div className='mt-2 text-xl font-semibold text-amber-950'>
                {item.type === 'zero_usage_users' ||
                item.type === 'balance_alert_users'
                  ? formatCount(item.abnormal_value)
                  : formatMoney(item.abnormal_value)}
              </div>
              <div className='mt-1 text-xs text-amber-800'>
                基线:{' '}
                {item.type === 'zero_usage_users' ||
                item.type === 'balance_alert_users'
                  ? formatCount(item.baseline_value)
                  : formatMoney(item.baseline_value)}
              </div>
              <div className='mt-2 text-xs text-amber-900'>
                {item.suggested_action}
              </div>
            </div>
          ))}
        </div>
      </Card>

      <TableCard
        title='待处理事项'
        loading={dashboardLoading}
        columns={dashboardTodoColumns}
        dataSource={dashboardTodos}
        rowKey={(record) => `${record.type}-${record.target_id}`}
      />

      <TableCard
        title='关键排行'
        extra={
          <Tabs
            type='button'
            activeKey={rankingView}
            onChange={setRankingView}
          >
            <TabPane tab='收入贡献' itemKey='income' />
            <TabPane tab='客户调用' itemKey='usage' />
            <TabPane tab='付费消耗' itemKey='paid_usage' />
            <TabPane tab='渠道支出' itemKey='channel_cost' />
          </Tabs>
        }
        loading={dashboardLoading}
        columns={rankingColumns}
        dataSource={dashboardRankings}
        rowKey={(record) => `${record.target_type}-${record.target_id}`}
      />
    </div>
  );

  const revenueContent = (
    <div className='space-y-4'>
      <div className='grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4'>
        <MetricCard
          icon={Receipt}
          title='付费充值总额'
          value={getAmountText(
            revenueUnit,
            revenueSummary?.paid_recharge_total?.usd,
            revenueSummary?.paid_recharge_total?.cos,
          )}
          tone='emerald'
        />
        <MetricCard
          icon={WalletCards}
          title='免费兑换总额'
          value={getAmountText(
            revenueUnit,
            revenueSummary?.gift_granted_total?.usd,
            revenueSummary?.gift_granted_total?.cos,
          )}
          tone='amber'
        />
        <MetricCard
          icon={WalletCards}
          title='未发放充值卡总额'
          value={getAmountText(
            revenueUnit,
            revenueSummary?.unissued_paid_voucher_total?.usd,
            revenueSummary?.unissued_paid_voucher_total?.cos,
          )}
          tone='sky'
        />
        <MetricCard
          icon={WalletCards}
          title='未兑换赠送卡总额'
          value={getAmountText(
            revenueUnit,
            revenueSummary?.unredeemed_gift_voucher_total?.usd,
            revenueSummary?.unredeemed_gift_voucher_total?.cos,
          )}
          tone='rose'
        />
      </div>

      <TableCard
        title='付费资金来源'
        extra={
          <Tabs
            type='button'
            activeKey={paidSourceView}
            onChange={(key) => {
              setPaidSourcesPage((prev) => ({ ...prev, page: 1 }));
              setPaidSourceView(key);
            }}
          >
            <TabPane tab='汇总' itemKey='summary' />
            <TabPane tab='明细' itemKey='detail' />
          </Tabs>
        }
        loading={revenueLoading}
        columns={
          paidSourceView === 'summary'
            ? paidSourceSummaryColumns
            : paidSourceDetailColumns
        }
        dataSource={paidSourcesPage.items || []}
        rowKey={(record, index) =>
          paidSourceView === 'summary'
            ? `${record.source_type}-${index}`
            : `${record.created_at}-${record.user_id}-${index}`
        }
        pagination={paidSourcesPage}
        onPageChange={(page) => setPaidSourcesPage((prev) => ({ ...prev, page }))}
        onPageSizeChange={(pageSize) =>
          setPaidSourcesPage((prev) => ({ ...prev, page: 1, pageSize }))
        }
      />

      <TableCard
        title='免费额度审计'
        extra={
          <Tabs
            type='button'
            activeKey={giftAuditView}
            onChange={setGiftAuditView}
          >
            <TabPane tab='汇总' itemKey='summary' />
            <TabPane tab='明细' itemKey='detail' />
          </Tabs>
        }
        loading={revenueLoading}
        columns={
          giftAuditView === 'summary'
            ? giftAuditSummaryColumns
            : giftAuditColumns
        }
        dataSource={
          giftAuditView === 'summary'
            ? giftAuditSummaryPage.items || []
            : giftAuditPage.items || []
        }
        rowKey={(record, index) =>
          giftAuditView === 'summary'
            ? `${record.source_type}-${record.activity_description}-${index}`
            : `${record.created_at}-${record.user_id}-${index}`
        }
        pagination={
          giftAuditView === 'summary' ? giftAuditSummaryPage : giftAuditPage
        }
        onPageChange={(page) => {
          if (giftAuditView === 'summary') {
            setGiftAuditSummaryPage((prev) => ({ ...prev, page }));
            return;
          }
          setGiftAuditPage((prev) => ({ ...prev, page }));
        }}
        onPageSizeChange={(pageSize) => {
          if (giftAuditView === 'summary') {
            setGiftAuditSummaryPage((prev) => ({ ...prev, page: 1, pageSize }));
            return;
          }
          setGiftAuditPage((prev) => ({ ...prev, page: 1, pageSize }));
        }}
      />
    </div>
  );

  const channelContent = (
    <div className='space-y-4'>
      <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
        <MetricCard
          icon={Receipt}
          title='本期渠道支出'
          value={formatMoney(channelSummary?.kpis?.total_cost_usd)}
          helper='按官网价格折算的渠道成本'
          tone='rose'
        />
        <MetricCard
          icon={LayoutDashboard}
          title='总调用量'
          value={formatTokens(channelSummary?.kpis?.total_tokens)}
          helper='输入、输出与缓存合计'
          tone='sky'
        />
      </div>

      <div className='grid grid-cols-1 gap-4 xl:grid-cols-2'>
        <ChartCard
          title='渠道成本占比'
          loading={channelLoading}
          hasData={channelChartData.length > 0}
          spec={buildPieSpec(
            channelChartData.map((item) => ({
              type: item.name,
              value: item.value,
            })),
            channelMetric === 'usd' ? '渠道 USD 支出' : '渠道总调用量',
          )}
        />
        <ChartCard
          title='模型成本占比'
          loading={channelLoading}
          hasData={modelChartData.length > 0}
          spec={buildPieSpec(
            modelChartData.map((item) => ({
              type: item.name,
              value: item.value,
            })),
            channelMetric === 'usd' ? '模型 USD 支出' : '模型总调用量',
          )}
        />
      </div>

      <TableCard
        title='渠道汇总'
        loading={channelLoading}
        columns={channelColumns}
        dataSource={channelPage.items || []}
        rowKey={(record) => record.channel_id}
        pagination={channelPage}
        onPageChange={(page) => setChannelPage((prev) => ({ ...prev, page }))}
        onPageSizeChange={(pageSize) =>
          setChannelPage((prev) => ({ ...prev, page: 1, pageSize }))
        }
      />

      <TableCard
        title='模型成本明细'
        loading={channelLoading}
        columns={modelColumns}
        dataSource={modelPage.items || []}
        rowKey={(record, index) =>
          `${record.channel_id}-${record.model_name}-${index}`
        }
        pagination={modelPage}
        onPageChange={(page) => setModelPage((prev) => ({ ...prev, page }))}
        onPageSizeChange={(pageSize) =>
          setModelPage((prev) => ({ ...prev, page: 1, pageSize }))
        }
      />
    </div>
  );

  const customerContent = (
    <div className='space-y-4'>
      <Card bordered className='!rounded-2xl' title='客户账单筛选'>
        <div className='flex flex-col gap-3 xl:flex-row xl:items-end'>
          <div className='min-w-[220px]'>
            <div className='mb-2 text-sm font-medium text-gray-600'>月份</div>
            <DatePicker
              type='month'
              value={getPickerValue('month', billMonthInput)}
              inputReadOnly
              onChange={(value) => setBillMonthInput(getPickerNextValue('month', value))}
            />
          </div>
          <div className='min-w-[260px] flex-1'>
            <div className='mb-2 text-sm font-medium text-gray-600'>
              用户 ID / 用户名
            </div>
            <Input
              value={billKeywordInput}
              onChange={setBillKeywordInput}
              placeholder='可选输入'
            />
          </div>
          <div className='flex gap-2'>
            <Button
              theme='solid'
              type='primary'
              loading={customerGenerating}
              disabled={!canWriteFinance}
              onClick={handleCustomerGenerate}
            >
              生成
            </Button>
            <Button
              icon={<Download size={14} />}
              onClick={handleExportCustomerBill}
              disabled={!selectedBillUserId}
            >
              导出 CSV
            </Button>
          </div>
        </div>
      </Card>

      <TableCard
        title='客户账单列表'
        loading={customerLoading}
        columns={customerSummaryColumns}
        dataSource={customerSummaryPage.items || []}
        rowKey={(record) => record.user_id}
        pagination={customerSummaryPage}
        onPageChange={(page) =>
          setCustomerSummaryPage((prev) => ({ ...prev, page }))
        }
        onPageSizeChange={(pageSize) =>
          setCustomerSummaryPage((prev) => ({ ...prev, page: 1, pageSize }))
        }
      />

      <TableCard
        title={`账单明细${selectedBillUserId ? ` - ${selectedBillUserName || `#${selectedBillUserId}`}` : ''}`}
        loading={customerDetailLoading}
        columns={customerDetailColumns}
        dataSource={customerBillDetails}
        rowKey={(record, index) =>
          `${record.occurred_at}-${record.request_id}-${index}`
        }
      />
    </div>
  );

  const auditContent = (
    <div className='space-y-4'>
      <Card bordered className='!rounded-2xl' title='财务审计筛选'>
        <div className='grid grid-cols-1 gap-3 xl:grid-cols-5'>
          <div>
            <div className='mb-2 text-sm font-medium text-gray-600'>模块</div>
            <Select
              value={auditModuleInput}
              onChange={(value) => setAuditModuleInput(`${value || ''}`)}
              optionList={[
                { label: '全部模块', value: '' },
                { label: '兑换码管理', value: 'redemption' },
                { label: '用户额度调整', value: 'user_quota' },
              ]}
            />
          </div>
          <div>
            <div className='mb-2 text-sm font-medium text-gray-600'>动作</div>
            <Select
              value={auditActionInput}
              onChange={(value) => setAuditActionInput(`${value || ''}`)}
              optionList={[
                { label: '全部动作', value: '' },
                { label: '创建', value: 'create' },
                { label: '修改', value: 'update' },
                { label: '删除', value: 'delete' },
                { label: '清理失效', value: 'cleanup_invalid' },
                { label: '调整', value: 'adjust' },
              ]}
            />
          </div>
          <div>
            <div className='mb-2 text-sm font-medium text-gray-600'>操作人</div>
            <Input
              value={auditOperatorInput}
              onChange={setAuditOperatorInput}
              placeholder='用户 ID / 用户名'
            />
          </div>
          <div>
            <div className='mb-2 text-sm font-medium text-gray-600'>目标用户</div>
            <Input
              value={auditTargetInput}
              onChange={setAuditTargetInput}
              placeholder='用户 ID / 用户名'
            />
          </div>
          <div className='flex items-end gap-2'>
            <Button theme='solid' type='primary' onClick={handleAuditQuery}>
              查询
            </Button>
            <Button onClick={handleAuditReset}>重置</Button>
          </div>
        </div>
      </Card>

      <TableCard
        title='审计流水'
        loading={auditLoading}
        columns={auditColumns}
        dataSource={auditPage.items || []}
        rowKey={(record) => record.id}
        pagination={auditPage}
        onPageChange={(page) => setAuditPage((prev) => ({ ...prev, page }))}
        onPageSizeChange={(pageSize) =>
          setAuditPage((prev) => ({ ...prev, page: 1, pageSize }))
        }
      />

      <Card
        bordered
        className='!rounded-[28px] border border-cyan-400/15 bg-[linear-gradient(180deg,rgba(17,26,72,0.94),rgba(8,13,36,0.98))] shadow-[0_26px_70px_rgba(2,6,23,0.48)] backdrop-blur-xl'
        title={<span className='text-base font-semibold tracking-[0.08em] text-slate-100'>审计详情</span>}
        bodyStyle={{ padding: 20 }}
      >
        {selectedAuditLog ? (
          <div className='space-y-4 text-slate-100'>
            <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4'>
              <div className='rounded-2xl border border-white/10 bg-white/5 px-4 py-3'>
                <div className='text-xs uppercase tracking-[0.2em] text-cyan-300'>模块</div>
                <div className='mt-2 font-medium'>
                  {auditModuleLabelMap[selectedAuditLog.module] || selectedAuditLog.module || '-'}
                </div>
              </div>
              <div className='rounded-2xl border border-white/10 bg-white/5 px-4 py-3'>
                <div className='text-xs uppercase tracking-[0.2em] text-fuchsia-300'>动作</div>
                <div className='mt-2 font-medium'>
                  {auditActionLabelMap[selectedAuditLog.action] || selectedAuditLog.action || '-'}
                </div>
              </div>
              <div className='rounded-2xl border border-white/10 bg-white/5 px-4 py-3'>
                <div className='text-xs uppercase tracking-[0.2em] text-sky-300'>操作人</div>
                <div className='mt-2 font-medium break-all'>
                  {selectedAuditLog.operator_user_id
                    ? `${selectedAuditLog.operator_username || selectedAuditLog.operator_username_snapshot || '-'} (#${selectedAuditLog.operator_user_id})`
                    : selectedAuditLog.operator_username || selectedAuditLog.operator_username_snapshot || '-'}
                </div>
              </div>
              <div className='rounded-2xl border border-white/10 bg-white/5 px-4 py-3'>
                <div className='text-xs uppercase tracking-[0.2em] text-violet-300'>目标</div>
                <div className='mt-2 font-medium break-all'>
                  {auditTargetLabelMap[selectedAuditLog.target_type] || selectedAuditLog.target_type || '-'}
                  {selectedAuditLog.target_id ? ` #${selectedAuditLog.target_id}` : ''}
                </div>
                <div className='mt-1 text-xs text-slate-400'>
                  {selectedAuditLog.target_user_id
                    ? `${selectedAuditLog.target_username || '-'} (#${selectedAuditLog.target_user_id})`
                    : '无目标用户'}
                </div>
              </div>
            </div>

            <div className='rounded-2xl border border-white/10 bg-white/5 px-4 py-3'>
              <div className='text-xs uppercase tracking-[0.2em] text-amber-300'>备注</div>
              <div className='mt-2 break-all text-sm text-slate-200'>
                {selectedAuditLog.remark || '-'}
              </div>
            </div>

            <div className='grid grid-cols-1 gap-4 xl:grid-cols-2'>
              <div className='rounded-2xl border border-cyan-400/15 bg-[#09112d]/80 p-4'>
                <div className='mb-3 text-sm font-semibold text-cyan-300'>Before</div>
                <pre className='max-h-[420px] overflow-auto whitespace-pre-wrap break-all text-xs leading-6 text-slate-200'>
                  {formatAuditJSON(selectedAuditLog.before_json)}
                </pre>
              </div>
              <div className='rounded-2xl border border-fuchsia-400/15 bg-[#120f34]/80 p-4'>
                <div className='mb-3 text-sm font-semibold text-fuchsia-300'>After</div>
                <pre className='max-h-[420px] overflow-auto whitespace-pre-wrap break-all text-xs leading-6 text-slate-200'>
                  {formatAuditJSON(selectedAuditLog.after_json)}
                </pre>
              </div>
            </div>
          </div>
        ) : (
          <Empty description='暂无审计详情' />
        )}
      </Card>
    </div>
  );

  return (
    <div className='billing-neon relative mt-[60px] overflow-hidden rounded-[36px] bg-[#08112d] px-3 pb-10 pt-3 shadow-[0_30px_120px_rgba(2,6,23,0.65)]'>
      <div className='pointer-events-none absolute left-0 top-0 h-full w-2 bg-gradient-to-b from-cyan-400 via-sky-400 to-fuchsia-500 opacity-90' />
      <div className='pointer-events-none absolute inset-0 opacity-90'>
        <div className='absolute inset-x-20 top-0 h-44 bg-[radial-gradient(circle_at_top,rgba(56,189,248,0.28),transparent_65%)]' />
        <div className='absolute -left-24 top-16 h-72 w-72 rounded-full bg-cyan-400/18 blur-3xl' />
        <div className='absolute right-8 top-14 h-80 w-80 rounded-full bg-fuchsia-500/14 blur-3xl' />
        <div className='absolute bottom-12 left-1/4 h-64 w-64 rounded-full bg-blue-500/14 blur-3xl' />
        <div className='absolute bottom-0 right-1/4 h-56 w-56 rounded-full bg-pink-500/12 blur-3xl' />
      </div>
      <div className='relative space-y-4'>
        <Card
          bordered
          className='!rounded-[32px] border border-cyan-400/15 bg-[linear-gradient(135deg,rgba(12,21,69,0.96),rgba(13,30,84,0.92),rgba(32,18,95,0.90))] shadow-[0_28px_90px_rgba(2,6,23,0.55)] backdrop-blur-xl'
          bodyStyle={{ padding: 24, position: 'relative', overflow: 'hidden' }}
        >
          <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(56,189,248,0.20),transparent_30%),radial-gradient(circle_at_bottom_left,rgba(236,72,153,0.16),transparent_34%)]' />
          <div className='pointer-events-none absolute inset-x-10 top-0 h-px bg-gradient-to-r from-transparent via-cyan-300/80 to-transparent' />
          <div className='flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between'>
            <div className='max-w-3xl'>
              <div className='mt-3 max-w-2xl text-sm leading-6 text-slate-300'>
                在统一指挥台里查看收入、赠送、渠道成本、客户账单与财务审计，快速判断经营状态与异常波动。
              </div>
            </div>
            <div className='grid grid-cols-2 gap-3 text-sm text-slate-300 sm:grid-cols-4'>
              <div className='rounded-2xl border border-cyan-300/15 bg-white/5 px-4 py-3 shadow-[0_12px_32px_rgba(8,47,73,0.25)] backdrop-blur-sm'>
                <div className='text-[11px] uppercase tracking-[0.24em] text-cyan-300'>维度</div>
                <div className='mt-2 font-semibold text-slate-50'>{periodType === 'year' ? '年度' : '月度'}</div>
              </div>
              <div className='rounded-2xl border border-fuchsia-300/15 bg-white/5 px-4 py-3 shadow-[0_12px_32px_rgba(88,28,135,0.26)] backdrop-blur-sm'>
                <div className='text-[11px] uppercase tracking-[0.24em] text-fuchsia-300'>窗口</div>
                <div className='mt-2 font-semibold text-slate-50'>{periodValue}</div>
              </div>
              <div className='rounded-2xl border border-violet-300/15 bg-white/5 px-4 py-3 shadow-[0_12px_32px_rgba(76,29,149,0.24)] backdrop-blur-sm'>
                <div className='text-[11px] uppercase tracking-[0.24em] text-violet-300'>面板</div>
                <div className='mt-2 font-semibold text-slate-50'>
                  {activeTabLabelMap[activeTab] || '财务中心'}
                </div>
              </div>
              <div className='rounded-2xl border border-sky-300/15 bg-white/5 px-4 py-3 shadow-[0_12px_32px_rgba(14,165,233,0.24)] backdrop-blur-sm'>
                <div className='text-[11px] uppercase tracking-[0.24em] text-sky-300'>状态</div>
                <div className='mt-2 font-semibold text-slate-50'>实时同步</div>
              </div>
            </div>
          </div>
        </Card>
        <Card
          bordered
          className='!rounded-[32px] border border-cyan-400/15 bg-[linear-gradient(180deg,rgba(12,18,52,0.96),rgba(7,11,31,0.98))] shadow-[0_24px_80px_rgba(2,6,23,0.55)] backdrop-blur-xl'
          headerExtraContent={
            <Tabs type='button' activeKey={activeTab} onChange={setActiveTab}>
              <TabPane tab='营运看板' itemKey='dashboard' />
              <TabPane tab='收入与赠送' itemKey='revenue' />
              <TabPane tab='渠道成本' itemKey='channel' />
              <TabPane tab='客户账单' itemKey='customer' />
              {canViewAudit ? <TabPane tab='财务审计' itemKey='audit' /> : null}
            </Tabs>
          }
        >
          <div className='flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between'>
            <div>
              <div className='text-sm font-medium text-slate-300'>统计维度</div>
              <Tabs
                type='button'
                activeKey={periodType}
                onChange={(key) => {
                  setPeriodType(key);
                  setPeriodValue(getCurrentPeriodValue(key));
                }}
              >
                <TabPane tab='月度' itemKey='month' />
                <TabPane tab='年度' itemKey='year' />
              </Tabs>
            </div>

            {activeTab !== 'customer' ? (
              <div className='flex flex-col gap-3 md:flex-row md:items-end'>
                <div>
                  <div className='mb-2 text-sm font-medium text-slate-300'>
                    时间
                  </div>
                  {periodType === 'year' ? (
                    <Select
                      value={periodValue}
                      optionList={getYearOptionList(periodValue)}
                      onChange={(value) => setPeriodValue(`${value}`)}
                      style={{ minWidth: 140 }}
                    />
                  ) : (
                    <DatePicker
                      type={getPickerType(periodType)}
                      value={getPickerValue(periodType, periodValue)}
                      inputReadOnly
                      onChange={(value) =>
                        setPeriodValue(getPickerNextValue(periodType, value))
                      }
                    />
                  )}
                </div>

                {activeTab === 'audit' ? null : (
                  <div>
                    <div className='text-sm font-medium text-slate-300'>单位切换</div>
                    {activeTab === 'channel' ? (
                      <Tabs
                        type='button'
                        activeKey={channelMetric}
                        onChange={setChannelMetric}
                      >
                        <TabPane tab='USD' itemKey='usd' />
                        <TabPane tab='Tokens' itemKey='tokens' />
                      </Tabs>
                    ) : (
                      <Tabs
                        type='button'
                        activeKey={
                          activeTab === 'dashboard' ? dashboardUnit : revenueUnit
                        }
                        onChange={(key) => {
                          if (activeTab === 'dashboard') {
                            setDashboardUnit(key);
                            return;
                          }
                          setRevenueUnit(key);
                        }}
                      >
                        <TabPane tab='USD' itemKey='usd' />
                        <TabPane tab='COS币' itemKey='cos' />
                      </Tabs>
                    )}
                  </div>
                )}
              </div>
            ) : (
              <div>
                <div className='text-sm font-medium text-slate-300'>单位切换</div>
                <Tabs
                  type='button'
                  activeKey={customerUnit}
                  onChange={setCustomerUnit}
                >
                  <TabPane tab='USD' itemKey='usd' />
                  <TabPane tab='COS币' itemKey='cos' />
                </Tabs>
              </div>
            )}
          </div>
        </Card>

        {activeTab === 'dashboard' ? dashboardContent : null}
        {activeTab === 'revenue' ? revenueContent : null}
        {activeTab === 'channel' ? channelContent : null}
        {activeTab === 'customer' ? customerContent : null}
        {activeTab === 'audit' ? auditContent : null}
      </div>
    </div>
  );
};

export default Billing;

