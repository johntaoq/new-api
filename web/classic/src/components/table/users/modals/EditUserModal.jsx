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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  hasAnyPermission,
  hasPermission,
  isRoot,
  renderQuota,
  showError,
  showSuccess,
} from '../../../../helpers';
import { getQuotaPerUnit } from '../../../../helpers/quota';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Avatar,
  Button,
  Card,
  Col,
  Form,
  Input,
  Modal,
  Row,
  SideSheet,
  Space,
  Spin,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconClose,
  IconEdit,
  IconLink,
  IconMinusCircle,
  IconPlus,
  IconSave,
  IconUser,
  IconUserGroup,
} from '@douyinfe/semi-icons';
import UserBindingManagementModal from './UserBindingManagementModal';

const { Text, Title } = Typography;

class AdjustmentPanelBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { errorMessage: '' };
  }

  static getDerivedStateFromError(error) {
    return {
      errorMessage:
        error?.message || 'Adjustment panel failed to render unexpectedly.',
    };
  }

  componentDidCatch(error, info) {
    // Keep the error visible in the browser console for local debugging.
    console.error('Adjustment panel render failure:', error, info);
  }

  render() {
    if (this.state.errorMessage) {
      return (
        <Card className='!rounded-2xl border border-red-200 bg-red-50 mt-3'>
          <Space vertical align='start' spacing='tight'>
            <Text strong>{this.props.title}</Text>
            <Text type='danger'>{this.state.errorMessage}</Text>
            <Button htmlType='button' theme='light' onClick={this.props.onReset}>
              {this.props.resetLabel}
            </Button>
          </Space>
        </Card>
      );
    }

    return this.props.children;
  }
}

const getInitValues = () => ({
  username: '',
  display_name: '',
  password: '',
  github_id: '',
  oidc_id: '',
  discord_id: '',
  wechat_id: '',
  telegram_id: '',
  linux_do_id: '',
  email: '',
  quota: 0,
  paid_quota: 0,
  gift_quota: 0,
  group: 'default',
  remark: '',
  staff_role: '',
});

const EditUserModal = (props) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);
  const pendingAdjustmentIdRef = useRef(0);

  const userId = props.editingUser?.id;
  const isEdit = Boolean(userId);

  const canManageOps = hasAnyPermission('ops.manage', 'system.manage');
  const canWriteFinance = hasAnyPermission('finance.write', 'system.manage');
  const canManageSystem = hasPermission('system.manage');
  const financeOnlyMode = canWriteFinance && !canManageOps;

  const [loading, setLoading] = useState(true);
  const [groupOptions, setGroupOptions] = useState([]);
  const [bindingModalVisible, setBindingModalVisible] = useState(false);
  const [adjustmentVisible, setAdjustmentVisible] = useState(false);
  const [pendingAdjustments, setPendingAdjustments] = useState([]);
  const [adjustmentEditingId, setAdjustmentEditingId] = useState(null);
  const [adjustmentFundingType, setAdjustmentFundingType] = useState('gift');
  const [adjustmentAmountUSD, setAdjustmentAmountUSD] = useState('');
  const [adjustmentSourceType, setAdjustmentSourceType] = useState('admin_grant');
  const [adjustmentRemark, setAdjustmentRemark] = useState('');
  const [adjustmentRevenueUSD, setAdjustmentRevenueUSD] = useState('');
  const [balanceSnapshot, setBalanceSnapshot] = useState({
    quota: 0,
    paid_quota: 0,
    gift_quota: 0,
  });

  const giftSourceOptions = useMemo(() => {
    const options = [
      { label: t('管理员赠送'), value: 'admin_grant' },
      { label: t('活动赠送'), value: 'promo_campaign' },
      { label: t('补偿'), value: 'compensation' },
      { label: t('赠送券/兑换码'), value: 'gift_voucher' },
    ];
    if (isRoot()) {
      options.push({ label: t('系统修正'), value: 'system_adjustment' });
    }
    return options;
  }, [t]);

  const quotaToUSD = (quota) => {
    const quotaPerUnit = getQuotaPerUnit();
    if (!quotaPerUnit) return 0;
    return Number((Number(quota || 0) / quotaPerUnit).toFixed(6));
  };

  const usdToQuota = (usdAmount) => {
    const amount = Number(usdAmount || 0);
    if (!Number.isFinite(amount) || amount === 0) {
      return 0;
    }
    return Math.round(Math.abs(amount) * getQuotaPerUnit()) * Math.sign(amount);
  };

  const formatUSD = (amount, withSign = false) => {
    const numeric = Number(amount || 0);
    const sign = withSign && numeric > 0 ? '+' : '';
    return `${sign}$${numeric.toFixed(6)}`;
  };

  const loadUser = async () => {
    setLoading(true);
    try {
      const url = userId ? `/api/user/${userId}` : '/api/user/self';
      const res = await API.get(url);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('加载用户失败'));
        return;
      }

      formApiRef.current?.setValues({
        ...getInitValues(),
        ...data,
        password: '',
      });
      setBalanceSnapshot({
        quota: Number(data?.quota || 0),
        paid_quota: Number(data?.paid_quota || 0),
        gift_quota: Number(data?.gift_quota || 0),
      });
    } catch (error) {
      if (financeOnlyMode && error?.response?.status === 403) {
        showError(t('财务角色不能查看或调账后台账号'));
        props.handleClose();
        return;
      }
      showError(error.message || error);
    } finally {
      setLoading(false);
    }
  };

  const fetchGroups = async () => {
    if (!canManageOps) return;
    try {
      const res = await API.get('/api/group/');
      setGroupOptions(
        (res.data.data || []).map((group) => ({ label: group, value: group })),
      );
    } catch (error) {
      showError(error.message || error);
    }
  };

  useEffect(() => {
    if (!props.visible) return;
    loadUser();
    fetchGroups();
    setBindingModalVisible(false);
    setPendingAdjustments([]);
    setAdjustmentVisible(false);
    setAdjustmentEditingId(null);
  }, [props.visible, userId, canManageOps]);

  const handleCancel = () => props.handleClose();

  const currentTotalQuota = Number(balanceSnapshot.quota || 0);
  const currentPaidQuota = Number(balanceSnapshot.paid_quota || 0);
  const currentGiftQuota = Number(balanceSnapshot.gift_quota || 0);

  const getPendingQuotaDelta = (fundingType, excludedId = null) =>
    pendingAdjustments.reduce((sum, item) => {
      if (item.fundingType !== fundingType) return sum;
      if (excludedId != null && item.id === excludedId) return sum;
      return sum + Number(item.deltaQuota || 0);
    }, 0);

  const pendingPaidQuotaDelta = getPendingQuotaDelta('paid');
  const pendingGiftQuotaDelta = getPendingQuotaDelta('gift');
  const pendingTotalQuotaDelta = pendingPaidQuotaDelta + pendingGiftQuotaDelta;

  const getBaseQuotaForAdjustment = (fundingType, excludedId = null) => {
    const currentQuota =
      fundingType === 'paid' ? currentPaidQuota : currentGiftQuota;
    return currentQuota + getPendingQuotaDelta(fundingType, excludedId);
  };

  const resetAdjustmentState = (fundingType = 'gift') => {
    setAdjustmentEditingId(null);
    setAdjustmentFundingType(fundingType);
    setAdjustmentAmountUSD('');
    setAdjustmentRemark('');
    setAdjustmentRevenueUSD('');
    setAdjustmentSourceType(
      fundingType === 'paid' ? 'system_adjustment' : 'admin_grant',
    );
  };

  const openAdjustmentModal = (fundingType) => {
    resetAdjustmentState(fundingType);
    setAdjustmentVisible(true);
  };

  const closeAdjustmentModal = () => {
    setAdjustmentVisible(false);
    resetAdjustmentState(adjustmentFundingType);
  };

  const stopEvent = (event) => {
    event?.preventDefault?.();
    event?.stopPropagation?.();
  };

  const handleAdjustmentButtonClick = (event, fundingType) => {
    stopEvent(event);
    openAdjustmentModal(fundingType);
  };

  const handleCloseAdjustmentClick = (event) => {
    stopEvent(event);
    closeAdjustmentModal();
  };

  const handleSubmitAdjustmentClick = (event) => {
    stopEvent(event);
    submitAdjustment();
  };

  const submitAdjustment = () => {
    const deltaUSD = Number(adjustmentAmountUSD || 0);
    if (!Number.isFinite(deltaUSD) || deltaUSD === 0) {
      showError(t('请输入非 0 的 USD 金额'));
      return;
    }

    const deltaQuota = usdToQuota(deltaUSD);
    if (deltaQuota === 0) {
      showError(t('该金额过小，换算后无法形成有效调账'));
      return;
    }

    const nextAdjustment = {
      id:
        adjustmentEditingId ||
        `pending-${Date.now()}-${pendingAdjustmentIdRef.current++}`,
      fundingType: adjustmentFundingType,
      sourceType:
        adjustmentFundingType === 'paid'
          ? 'system_adjustment'
          : adjustmentSourceType,
      remark: adjustmentRemark,
      deltaUSD,
      deltaQuota,
      revenueUSD:
        adjustmentFundingType === 'paid'
          ? Number(
              adjustmentRevenueUSD === '' || adjustmentRevenueUSD == null
                ? deltaUSD
                : adjustmentRevenueUSD,
            )
          : 0,
    };

    setPendingAdjustments((prev) => {
      if (adjustmentEditingId) {
        return prev.map((item) =>
          item.id === adjustmentEditingId ? nextAdjustment : item,
        );
      }
      return [...prev, nextAdjustment];
    });

    showSuccess(
      adjustmentEditingId ? t('待提交调账已更新') : t('调账已加入待提交队列'),
    );
    closeAdjustmentModal();
  };

  const editPendingAdjustment = (adjustment) => {
    setAdjustmentEditingId(adjustment.id);
    setAdjustmentFundingType(adjustment.fundingType);
    setAdjustmentAmountUSD(`${adjustment.deltaUSD ?? ''}`);
    setAdjustmentSourceType(adjustment.sourceType || 'admin_grant');
    setAdjustmentRemark(adjustment.remark || '');
    setAdjustmentRevenueUSD(
      adjustment.fundingType === 'paid' ? `${adjustment.revenueUSD ?? ''}` : '',
    );
    setAdjustmentVisible(true);
  };

  const removePendingAdjustment = (adjustmentId) => {
    setPendingAdjustments((prev) =>
      prev.filter((item) => item.id !== adjustmentId),
    );
  };

  const submit = async (values) => {
    setLoading(true);
    try {
      if (!canManageOps && pendingAdjustments.length === 0) {
        showError(t('当前没有可提交的财务调整'));
        return;
      }

      if (canManageOps) {
        const payload = { ...values };
        delete payload.quota;
        delete payload.paid_quota;
        delete payload.gift_quota;
        if (userId) {
          payload.id = Number.parseInt(`${userId}`, 10);
        }

        const url = userId ? '/api/user/' : '/api/user/self';
        const res = await API.put(url, payload);
        const { success, message } = res.data;
        if (!success) {
          showError(message || t('保存用户失败'));
          return;
        }
      } else if (!userId) {
        showError(t('财务调账只能对已有用户执行'));
        return;
      }

      let appliedAdjustmentCount = 0;
      try {
        for (const adjustment of pendingAdjustments) {
          const adjustmentPayload = {
            funding_type: adjustment.fundingType,
            delta_quota: adjustment.deltaQuota,
            source_type: adjustment.sourceType,
            remark: adjustment.remark,
          };
          if (adjustment.fundingType === 'paid') {
            adjustmentPayload.revenue_usd = Number(adjustment.revenueUSD || 0);
          }

          const adjustmentRes = await API.post(
            `/api/user/${userId}/quota_adjust`,
            adjustmentPayload,
          );
          if (!adjustmentRes.data?.success) {
            throw new Error(adjustmentRes.data?.message || t('调账提交失败'));
          }
          appliedAdjustmentCount += 1;
        }
      } catch (error) {
        const remainingAdjustments = pendingAdjustments.slice(
          appliedAdjustmentCount,
        );
        setPendingAdjustments(remainingAdjustments);
        await loadUser();
        props.refresh();
        showError(
          appliedAdjustmentCount > 0
            ? `${t('部分调账已提交，剩余待提交')} ${remainingAdjustments.length} ${t('条')}：${
                error.message || error
              }`
            : error.message || error,
        );
        return;
      }

      setPendingAdjustments([]);
      showSuccess(
        pendingAdjustments.length > 0
          ? t('用户信息与调账修改已提交')
          : t('用户信息更新成功'),
      );
      props.refresh();
      props.handleClose();
    } catch (error) {
      showError(error.message || error);
    } finally {
      setLoading(false);
    }
  };

  const currentAdjustmentQuota = getBaseQuotaForAdjustment(
    adjustmentFundingType,
    adjustmentEditingId,
  );
  const currentAdjustmentDeltaQuota = usdToQuota(adjustmentAmountUSD);
  const currentAdjustmentRevenueUSD =
    adjustmentFundingType === 'paid'
      ? Number(
          adjustmentRevenueUSD === '' || adjustmentRevenueUSD == null
            ? adjustmentAmountUSD || 0
            : adjustmentRevenueUSD,
        )
      : 0;

  const buildPendingAdjustmentPreview = (adjustment) => {
    let baseQuota =
      adjustment.fundingType === 'paid' ? currentPaidQuota : currentGiftQuota;
    for (const item of pendingAdjustments) {
      if (item.id === adjustment.id) {
        break;
      }
      if (item.fundingType === adjustment.fundingType) {
        baseQuota += item.deltaQuota;
      }
    }
    return {
      baseQuota,
      afterQuota: baseQuota + adjustment.deltaQuota,
    };
  };

  const quotaCards = [
    {
      key: 'quota',
      color: 'blue',
      title: t('总余额'),
      currentValue: currentTotalQuota,
      previewValue: currentTotalQuota + pendingTotalQuotaDelta,
      helper:
        pendingTotalQuotaDelta !== 0
          ? t('已包含待提交调账预览')
          : t('当前已生效余额'),
    },
    {
      key: 'paid_quota',
      color: 'red',
      title: t('付费余额'),
      currentValue: currentPaidQuota,
      previewValue: currentPaidQuota + pendingPaidQuotaDelta,
      helper: t('仅 root 可调账'),
    },
    {
      key: 'gift_quota',
      color: 'green',
      title: t('赠送余额'),
      currentValue: currentGiftQuota,
      previewValue: currentGiftQuota + pendingGiftQuotaDelta,
      helper: t('财务角色可调赠送余额'),
    },
  ];

  return (
    <>
      <SideSheet
        placement='right'
        title={
          <Space>
            <Tag color='blue' shape='circle'>
              {t(isEdit ? '编辑' : '新建')}
            </Tag>
            <Title heading={4} className='m-0'>
              {financeOnlyMode ? t('查看 / 调账用户') : t('编辑用户')}
            </Title>
          </Space>
        }
        bodyStyle={{ padding: 0 }}
        visible={props.visible}
        width={isMobile ? '100%' : 680}
        footer={
          <div className='flex justify-end bg-white'>
            <Space>
              <Button
                theme='solid'
                onClick={() => formApiRef.current?.submitForm()}
                icon={<IconSave />}
                loading={loading}
              >
                {pendingAdjustments.length > 0 ? t('提交全部修改') : t('提交')}
              </Button>
              <Button
                theme='light'
                type='primary'
                onClick={handleCancel}
                icon={<IconClose />}
              >
                {t('取消')}
              </Button>
            </Space>
          </div>
        }
        closeIcon={null}
        onCancel={handleCancel}
      >
        <Spin spinning={loading}>
          <Form
            initValues={getInitValues()}
            getFormApi={(api) => {
              formApiRef.current = api;
            }}
            onSubmit={submit}
            onSubmitFail={(errs) => {
              const first = Object.values(errs || {})[0];
              if (first) {
                showError(Array.isArray(first) ? first[0] : first);
              }
              formApiRef.current?.scrollToError?.();
            }}
          >
            <div className='p-2 space-y-3'>
              <Card className='!rounded-2xl shadow-sm border-0'>
                <div className='flex items-center mb-2'>
                  <Avatar size='small' color='blue' className='mr-2 shadow-md'>
                    <IconUser size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>{t('基本信息')}</Text>
                    <div className='text-xs text-gray-600'>
                      {financeOnlyMode
                        ? t('财务角色仅查看基础资料，不可编辑运营字段')
                        : t('用户基础账号信息')}
                    </div>
                  </div>
                </div>

                <Row gutter={12}>
                  <Col span={24}>
                    <Form.Input
                      field='username'
                      label={t('用户名')}
                      placeholder={t('请输入用户名')}
                      rules={[{ required: true, message: t('请输入用户名') }]}
                      showClear
                      disabled={financeOnlyMode}
                    />
                  </Col>

                  {canManageOps ? (
                    <Col span={24}>
                      <Form.Input
                        field='password'
                        label={t('密码')}
                        placeholder={t('如需重置密码请在此输入新密码')}
                        mode='password'
                        showClear
                      />
                    </Col>
                  ) : null}

                  <Col span={24}>
                    <Form.Input
                      field='display_name'
                      label={t('显示名称')}
                      placeholder={t('请输入显示名称')}
                      showClear
                      disabled={financeOnlyMode}
                    />
                  </Col>

                  {canManageOps ? (
                    <Col span={24}>
                      <Form.Input
                        field='remark'
                        label={t('备注')}
                        placeholder={t('仅管理员可见')}
                        showClear
                      />
                    </Col>
                  ) : null}

                  {canManageSystem ? (
                    <Col span={24}>
                      <Form.Select
                        field='staff_role'
                        label={t('后台职责')}
                        optionList={[
                          { label: t('普通用户'), value: '' },
                          { label: t('运营管理员'), value: 'admin' },
                          { label: t('财务'), value: 'finance' },
                          { label: t('审计'), value: 'audit' },
                        ]}
                      />
                    </Col>
                  ) : null}
                </Row>
              </Card>

              {userId ? (
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <div className='flex items-center mb-2'>
                    <Avatar size='small' color='green' className='mr-2 shadow-md'>
                      <IconUserGroup size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>{t('权限与账务')}</Text>
                      <div className='text-xs text-gray-600'>
                        {t('这里只展示用户可理解的余额，不直接暴露内部记账单位')}
                      </div>
                    </div>
                  </div>

                  {canManageOps ? (
                    <Row gutter={12}>
                      <Col span={24}>
                        <Form.Select
                          field='group'
                          label={t('分组')}
                          placeholder={t('请选择分组')}
                          optionList={groupOptions}
                          allowAdditions
                          search
                          rules={[{ required: true, message: t('请选择分组') }]}
                        />
                      </Col>
                    </Row>
                  ) : null}

                  <Row gutter={[12, 12]} className='mt-1'>
                    {quotaCards.map((card) => (
                      <Col xs={24} md={8} key={card.key}>
                        <Card className='!rounded-2xl border-0 bg-[var(--semi-color-fill-0)] h-full'>
                          <Space vertical align='start' spacing='tight'>
                            <Tag color={card.color}>{card.title}</Tag>
                            <Title heading={5} className='!mb-0'>
                              {renderQuota(card.previewValue)}
                            </Title>
                            <Text type='secondary'>
                              {`${t('当前')} ${renderQuota(card.currentValue)}`}
                            </Text>
                            <Text type='secondary'>
                              {`${t('等价 USD')} ${formatUSD(
                                quotaToUSD(card.previewValue),
                              )}`}
                            </Text>
                            <Text type='secondary'>{card.helper}</Text>
                          </Space>
                        </Card>
                      </Col>
                    ))}
                  </Row>

                  {canWriteFinance ? (
                    <>
                      <div className='mt-3 flex flex-wrap gap-2 items-center'>
                        <Button
                          htmlType='button'
                          icon={<IconPlus />}
                          onClick={(event) =>
                            handleAdjustmentButtonClick(event, 'gift')
                          }
                        >
                          {t('调整赠送余额')}
                        </Button>
                        {canManageSystem ? (
                          <Button
                            htmlType='button'
                            type='primary'
                            theme='outline'
                            icon={<IconPlus />}
                            onClick={(event) =>
                              handleAdjustmentButtonClick(event, 'paid')
                            }
                          >
                            {t('调整付费余额')}
                          </Button>
                        ) : null}
                        <Text type='secondary'>
                          {canManageSystem
                            ? t('管理员输入 USD，系统展示对应余额值，提交前可先 review')
                            : t('财务角色仅可调整赠送余额，输入口径统一为 USD')}
                        </Text>
                      </div>

                      {pendingAdjustments.length > 0 ? (
                        <Card className='!rounded-2xl border-0 bg-[var(--semi-color-fill-0)] mt-3'>
                          <Space
                            vertical
                            align='start'
                            style={{ width: '100%' }}
                            spacing='medium'
                          >
                            <div className='flex items-center justify-between w-full'>
                              <div>
                                <Text className='text-base font-medium'>
                                  {t('待提交账务修改')}
                                </Text>
                                <div className='text-xs text-gray-500'>
                                  {t('管理员可以先 review 调账内容，再统一提交')}
                                </div>
                              </div>
                              <Tag color='orange'>{pendingAdjustments.length}</Tag>
                            </div>

                            {pendingAdjustments.map((adjustment) => {
                              const preview =
                                buildPendingAdjustmentPreview(adjustment);
                              return (
                                <Card
                                  key={adjustment.id}
                                  className='!rounded-2xl border border-[var(--semi-color-border)] w-full'
                                  bodyStyle={{ padding: 16 }}
                                >
                                  <div className='flex items-start justify-between gap-3'>
                                    <Space vertical align='start' spacing='tight'>
                                      <Space wrap>
                                        <Tag
                                          color={
                                            adjustment.fundingType === 'paid'
                                              ? 'red'
                                              : 'green'
                                          }
                                        >
                                          {adjustment.fundingType === 'paid'
                                            ? t('付费余额')
                                            : t('赠送余额')}
                                        </Tag>
                                        <Tag>{adjustment.sourceType}</Tag>
                                      </Space>
                                      <Text>
                                        {`${t('本次调整')} ${formatUSD(
                                          adjustment.deltaUSD,
                                          true,
                                        )}`}
                                      </Text>
                                      <Text>
                                        {`${t('平台展示值')} ${renderQuota(
                                          adjustment.deltaQuota,
                                        )}`}
                                      </Text>
                                      <Text>
                                        {`${t('调整前余额')} ${renderQuota(
                                          preview.baseQuota,
                                        )}`}
                                      </Text>
                                      <Text>
                                        {`${t('调整后余额')} ${renderQuota(
                                          preview.afterQuota,
                                        )}`}
                                      </Text>
                                      {adjustment.fundingType === 'paid' ? (
                                        <Text>
                                          {`${t('确认收入')} ${formatUSD(
                                            adjustment.revenueUSD,
                                          )}`}
                                        </Text>
                                      ) : null}
                                      {adjustment.remark ? (
                                        <Text type='secondary'>
                                          {`${t('备注')}：${adjustment.remark}`}
                                        </Text>
                                      ) : null}
                                    </Space>
                                    <Space>
                                      <Button
                                        htmlType='button'
                                        theme='outline'
                                        icon={<IconEdit />}
                                        onClick={() =>
                                          editPendingAdjustment(adjustment)
                                        }
                                      >
                                        {t('编辑')}
                                      </Button>
                                      <Button
                                        htmlType='button'
                                        theme='borderless'
                                        type='danger'
                                        icon={<IconMinusCircle />}
                                        onClick={() =>
                                          removePendingAdjustment(adjustment.id)
                                        }
                                      >
                                        {t('删除')}
                                      </Button>
                                    </Space>
                                  </div>
                                </Card>
                              );
                            })}
                          </Space>
                        </Card>
                      ) : null}
                    </>
                  ) : null}
                </Card>
              ) : null}

              {userId && canManageOps ? (
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <div className='flex items-center justify-between gap-3'>
                    <div className='flex items-center min-w-0'>
                      <Avatar size='small' color='purple' className='mr-2 shadow-md'>
                        <IconLink size={16} />
                      </Avatar>
                      <div className='min-w-0'>
                        <Text className='text-lg font-medium'>{t('绑定信息')}</Text>
                        <div className='text-xs text-gray-600'>
                          {t('管理用户已绑定的第三方账号，支持查看与解绑')}
                        </div>
                      </div>
                    </div>
                    <Button
                      htmlType='button'
                      type='primary'
                      theme='outline'
                      onClick={() => setBindingModalVisible(true)}
                    >
                      {t('管理绑定')}
                    </Button>
                  </div>
                </Card>
              ) : null}
            </div>
          </Form>
        </Spin>
      </SideSheet>

      <UserBindingManagementModal
        visible={bindingModalVisible}
        onCancel={() => setBindingModalVisible(false)}
        userId={userId}
        isMobile={isMobile}
        formApiRef={formApiRef}
      />

      <Modal
        centered
        visible={adjustmentVisible}
        title={t(
          adjustmentFundingType === 'paid' ? '调整付费余额' : '调整赠送余额',
        )}
        footer={null}
        onCancel={closeAdjustmentModal}
        style={{ maxWidth: isMobile ? 'calc(100vw - 16px)' : 760 }}
        bodyStyle={{ paddingTop: 8 }}
        closeOnEsc
        keepDOM={false}
      >
        <AdjustmentPanelBoundary
          title={t('调账面板渲染失败')}
          resetLabel={t('关闭调账面板')}
          onReset={closeAdjustmentModal}
        >
          <Space
            vertical
            align='start'
            style={{ width: '100%' }}
            spacing='medium'
          >
            <Card className='!rounded-2xl border-0 bg-[var(--semi-color-fill-0)] w-full'>
              <Space vertical align='start' spacing='tight'>
                <Text className='text-base font-medium'>
                  {t('输入 USD，系统自动换算展示值；确认后先加入待提交列表')}
                </Text>
                <Text type='secondary'>{t('当前余额')}</Text>
                <Title heading={5} className='!mb-0'>
                  {renderQuota(currentAdjustmentQuota)}
                </Title>
                <Text type='secondary'>
                  {`${t('等价 USD')} ${formatUSD(
                    quotaToUSD(currentAdjustmentQuota),
                  )}`}
                </Text>
                <Text type='secondary'>
                  {`${t('调整后余额')} ${renderQuota(
                    currentAdjustmentQuota + currentAdjustmentDeltaQuota,
                  )}`}
                </Text>
              </Space>
            </Card>

            {adjustmentFundingType === 'gift' ? (
              <div style={{ width: '100%' }}>
                <Text size='small'>{t('来源类别')}</Text>
                <select
                  value={adjustmentSourceType}
                  onChange={(event) =>
                    setAdjustmentSourceType(event.target.value || 'admin_grant')
                  }
                  style={{
                    width: '100%',
                    marginTop: 8,
                    minHeight: 36,
                    borderRadius: 8,
                    border: '1px solid var(--semi-color-border)',
                    padding: '0 12px',
                    background: 'var(--semi-color-bg-2)',
                  }}
                >
                  {giftSourceOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
            ) : null}

            <div style={{ width: '100%' }}>
              <Text size='small'>{t('输入 USD 调整金额')}</Text>
              <Input
                type='number'
                style={{ width: '100%', marginTop: 8 }}
                value={adjustmentAmountUSD}
                onChange={(value) => setAdjustmentAmountUSD(value || '')}
                placeholder={t('正数为增加，负数为扣减')}
              />
              <Text type='secondary' size='small'>
                {`${t('对应平台展示值')} ${renderQuota(
                  currentAdjustmentDeltaQuota,
                )}`}
              </Text>
            </div>

            {adjustmentFundingType === 'paid' ? (
              <div style={{ width: '100%' }}>
                <Text size='small'>{t('确认收入 USD')}</Text>
                <Input
                  type='number'
                  style={{ width: '100%', marginTop: 8 }}
                  value={adjustmentRevenueUSD}
                  onChange={(value) => setAdjustmentRevenueUSD(value || '')}
                  placeholder={t('默认等于本次调整的 USD 值')}
                />
                <Text type='secondary' size='small'>
                  {`${t('当前确认收入预览')} ${formatUSD(
                    currentAdjustmentRevenueUSD,
                  )}`}
                </Text>
              </div>
            ) : null}

            <div style={{ width: '100%' }}>
              <Text size='small'>{t('备注')}</Text>
              <TextArea
                value={adjustmentRemark}
                onChange={setAdjustmentRemark}
                rows={4}
                style={{ marginTop: 8 }}
                placeholder={t(
                  '用于财务审计说明，例如活动名、补偿原因',
                )}
              />
            </div>

            <div className='flex justify-end w-full gap-2'>
              <Button
                htmlType='button'
                theme='light'
                onClick={handleCloseAdjustmentClick}
              >
                {t('取消')}
              </Button>
              <Button
                htmlType='button'
                type='primary'
                icon={<IconPlus />}
                onClick={handleSubmitAdjustmentClick}
              >
                {t(adjustmentEditingId ? '更新待提交' : '加入待提交')}
              </Button>
            </div>
          </Space>
        </AdjustmentPanelBoundary>
      </Modal>
    </>
  );
};

export default EditUserModal;
