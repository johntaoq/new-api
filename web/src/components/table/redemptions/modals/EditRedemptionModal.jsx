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
  downloadTextAsFile,
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
  Modal,
  Row,
  SideSheet,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconClose,
  IconCreditCard,
  IconGift,
  IconSave,
} from '@douyinfe/semi-icons';

const { Text, Title } = Typography;

const QUICK_AMOUNT_OPTIONS = [1, 10, 50, 100, 500, 1000];

const formatUSD = (value, withPrefix = true) => {
  const amount = Number(value || 0);
  if (!Number.isFinite(amount)) {
    return withPrefix ? '$0.000000' : '0.000000';
  }
  const formatted = amount.toFixed(6);
  return withPrefix ? `$${formatted}` : formatted;
};

const buildDefaultName = (fundingType, amountUSD) => {
  const label = fundingType === 'paid' ? 'Paid' : 'Free';
  return `${label} $${Number(amountUSD || 0).toFixed(2)}`;
};

const EditRedemptionModal = (props) => {
  const { t } = useTranslation();
  const isEdit = props.editingRedemption.id !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);

  const quotaPerUnit = useMemo(() => getQuotaPerUnit(), []);

  const usdToQuota = (usdAmount) => {
    const amount = Number(usdAmount || 0);
    if (!Number.isFinite(amount) || amount <= 0 || quotaPerUnit <= 0) {
      return 0;
    }
    return Math.round(amount * quotaPerUnit);
  };

  const getInitValues = () => ({
    name: '',
    funding_type: 'paid',
    amount_usd: 1,
    count: 1,
    expired_time: null,
    remark: '',
  });

  const handleCancel = () => {
    props.handleClose();
  };

  const loadRedemption = async () => {
    setLoading(true);
    try {
      const res = await API.get(`/api/redemption/${props.editingRedemption.id}`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      const nextValues = {
        ...getInitValues(),
        ...data,
        amount_usd: Number(data.amount_usd || 0),
        funding_type: data.funding_type || 'gift',
      };
      if (data.expired_time === 0) {
        nextValues.expired_time = null;
      } else {
        nextValues.expired_time = new Date(data.expired_time * 1000);
      }
      formApiRef.current?.setValues(nextValues);
    } catch (error) {
      showError(error.message || error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!formApiRef.current) {
      return;
    }
    if (isEdit) {
      loadRedemption();
    } else {
      formApiRef.current.setValues(getInitValues());
      setLoading(false);
    }
  }, [props.editingRedemption.id]);

  const submit = async (values) => {
    const fundingType = values.funding_type === 'paid' ? 'paid' : 'gift';
    const amountUSD = Number(values.amount_usd || 0);
    const quota = usdToQuota(amountUSD);
    if (!Number.isFinite(amountUSD) || amountUSD <= 0) {
      showError(t('请输入大于 0 的 USD 金额'));
      return;
    }
    if (quota <= 0) {
      showError(t('该金额过小，无法换算出有效的平台额度'));
      return;
    }
    if (fundingType === 'gift' && !(values.remark || '').trim()) {
      showError(t('免费兑换码必须填写备注，便于审计'));
      return;
    }

    setLoading(true);
    const payload = {
      ...values,
      funding_type: fundingType,
      amount_usd: amountUSD,
      quota,
      recognized_revenue_usd: fundingType === 'paid' ? amountUSD : 0,
      remark: (values.remark || '').trim(),
      name: (values.name || '').trim(),
      count: parseInt(values.count, 10) || 0,
    };
    if (!payload.name) {
      payload.name = buildDefaultName(fundingType, amountUSD);
    }
    if (!payload.expired_time) {
      payload.expired_time = 0;
    } else {
      payload.expired_time = Math.floor(payload.expired_time.getTime() / 1000);
    }

    let res;
    if (isEdit) {
      res = await API.put('/api/redemption/', {
        ...payload,
        id: parseInt(props.editingRedemption.id, 10),
      });
    } else {
      res = await API.post('/api/redemption/', payload);
    }
    const { success, message, data } = res.data;
    if (success) {
      showSuccess(isEdit ? t('兑换码更新成功') : t('兑换码创建成功'));
      props.refresh();
      formApiRef.current?.setValues(getInitValues());
      props.handleClose();
    } else {
      showError(message);
    }

    if (!isEdit && data) {
      let text = '';
      for (let i = 0; i < data.length; i++) {
        text += data[i] + '\n';
      }
      Modal.confirm({
        title: t('兑换码创建成功'),
        content: (
          <div>
            <p>{t('是否下载本次生成的兑换码列表？')}</p>
            <p>{t('系统会将兑换码保存为文本文件，文件名默认使用兑换码名称。')}</p>
          </div>
        ),
        onOk: () => {
          downloadTextAsFile(text, `${payload.name}.txt`);
        },
      });
    }
    setLoading(false);
  };

  return (
    <SideSheet
      placement={isEdit ? 'right' : 'left'}
      title={
        <Space>
          {isEdit ? (
            <Tag color='blue' shape='circle'>
              {t('编辑')}
            </Tag>
          ) : (
            <Tag color='green' shape='circle'>
              {t('新建')}
            </Tag>
          )}
          <Title heading={4} className='m-0'>
            {isEdit ? t('编辑兑换码') : t('创建兑换码')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: 0 }}
      visible={props.visiable}
      width={isMobile ? '100%' : 640}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button
              theme='solid'
              onClick={() => formApiRef.current?.submitForm()}
              icon={<IconSave />}
              loading={loading}
            >
              {t('提交')}
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
        >
          {({ values }) => {
            const fundingType = values.funding_type === 'paid' ? 'paid' : 'gift';
            const previewQuota = usdToQuota(values.amount_usd);
            const isFree = fundingType === 'gift';

            return (
              <div className='p-2'>
                <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='blue'
                      className='mr-2 shadow-md'
                    >
                      <IconGift size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>{t('基本信息')}</Text>
                      <div className='text-xs text-gray-600'>
                        {t('为兑换码设置类型、名称和有效期')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.Select
                        field='funding_type'
                        label={t('兑换码类型')}
                        style={{ width: '100%' }}
                        optionList={[
                          { label: t('付费兑换码'), value: 'paid' },
                          { label: t('免费兑换码'), value: 'gift' },
                        ]}
                        rules={[{ required: true, message: t('请选择兑换码类型') }]}
                      />
                    </Col>
                    <Col span={24}>
                      <Form.Input
                        field='name'
                        label={t('名称')}
                        placeholder={t('可留空，系统将按类型和 USD 金额自动命名')}
                        style={{ width: '100%' }}
                        showClear
                      />
                    </Col>
                    <Col span={24}>
                      <Form.DatePicker
                        field='expired_time'
                        label={t('过期时间')}
                        type='dateTime'
                        placeholder={t('可留空，表示不过期')}
                        style={{ width: '100%' }}
                        showClear
                      />
                    </Col>
                    <Col span={24}>
                      <Form.TextArea
                        field='remark'
                        label={
                          isFree
                            ? t('审计备注')
                            : t('备注 / 外部流水号（可选）')
                        }
                        placeholder={
                          isFree
                            ? t('例如：新用户赠送、活动补贴、售后补偿')
                            : t('可选填写：渠道-流水号、外部订单号、支付回执号等')
                        }
                        autosize={{ minRows: 3, maxRows: 5 }}
                        rules={
                          isFree
                            ? [
                                {
                                  required: true,
                                  message: t('免费兑换码必须填写备注'),
                                },
                              ]
                            : []
                        }
                        maxCount={255}
                        showClear
                      />
                    </Col>
                  </Row>
                </Card>

                <Card className='!rounded-2xl shadow-sm border-0'>
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='green'
                      className='mr-2 shadow-md'
                    >
                      <IconCreditCard size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>{t('金额设置')}</Text>
                      <div className='text-xs text-gray-600'>
                        {t('仅输入 USD，系统自动换算平台显示值，内部 Quota 不再展示')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={isEdit ? 24 : 12}>
                      <Form.InputNumber
                        field='amount_usd'
                        label={t('面值（USD）')}
                        prefix='$'
                        min={0.000001}
                        precision={6}
                        style={{ width: '100%' }}
                        placeholder={t('请输入兑换金额（USD）')}
                        rules={[
                          { required: true, message: t('请输入 USD 金额') },
                          {
                            validator: (rule, value) => {
                              return Number(value || 0) > 0
                                ? Promise.resolve()
                                : Promise.reject(t('USD 金额必须大于 0'));
                            },
                          },
                        ]}
                      />
                    </Col>
                    {!isEdit && (
                      <Col span={12}>
                        <Form.InputNumber
                          field='count'
                          label={t('生成数量')}
                          min={1}
                          style={{ width: '100%' }}
                          rules={[
                            { required: true, message: t('请输入生成数量') },
                            {
                              validator: (rule, value) => {
                                const count = parseInt(value, 10);
                                return count > 0
                                  ? Promise.resolve()
                                  : Promise.reject(t('生成数量必须大于 0'));
                              },
                            },
                          ]}
                        />
                      </Col>
                    )}
                  </Row>

                  <div className='mt-3 flex flex-wrap gap-2'>
                    {QUICK_AMOUNT_OPTIONS.map((amount) => (
                      <Button
                        key={amount}
                        size='small'
                        theme='outline'
                        onClick={() => formApiRef.current?.setValue('amount_usd', amount)}
                      >
                        {`$${amount}`}
                      </Button>
                    ))}
                  </div>

                  <Card className='!rounded-2xl border-0 bg-[var(--semi-color-fill-0)] mt-3'>
                    <Space vertical align='start' spacing='tight'>
                      <Space wrap>
                        <Tag color={fundingType === 'paid' ? 'blue' : 'green'}>
                          {fundingType === 'paid'
                            ? t('付费兑换码')
                            : t('免费兑换码')}
                        </Tag>
                        <Tag>{`${t('USD 输入')} ${formatUSD(values.amount_usd)}`}</Tag>
                      </Space>
                      <Title heading={5} className='!mb-0'>
                        {previewQuota > 0
                          ? renderQuota(previewQuota)
                          : t('请输入有效金额')}
                      </Title>
                      <Text type='secondary'>
                        {t('上方显示的是平台展示值，不再暴露内部 Quota')}
                      </Text>
                      {values.remark && (
                        <Text type='secondary'>
                          {`${isFree ? t('审计备注') : t('备注 / 外部流水号')}：${values.remark}`}
                        </Text>
                      )}
                    </Space>
                  </Card>
                </Card>
              </div>
            );
          }}
        </Form>
      </Spin>
    </SideSheet>
  );
};

export default EditRedemptionModal;
