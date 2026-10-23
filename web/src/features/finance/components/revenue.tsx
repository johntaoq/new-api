import { useState } from 'react'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'

import { useFinanceQuery } from '../api'
import type {
  FinanceColumn,
  FinanceParams,
  FinanceUnit,
  RevenueSummary,
} from '../types'
import { useFinanceFormat } from '../use-finance-format'
import { FinanceMetrics, FinanceRemoteTable, FinanceSegments } from './shared'

const amountColumns: FinanceColumn[] = [
  {
    key: 'recharge_amount_usd',
    cosKey: 'recharge_amount_cos',
    label: 'Recharge amount',
    kind: 'money',
  },
  {
    key: 'remaining_amount_usd',
    cosKey: 'remaining_amount_cos',
    label: 'Remaining balance',
    kind: 'money',
  },
]
const views = [
  { value: 'summary', label: 'Summary' },
  { value: 'detail', label: 'Details' },
]

export function FinanceRevenue(props: {
  params: FinanceParams
  unit: FinanceUnit
}) {
  const [paidView, setPaidView] = useState('summary')
  const [giftView, setGiftView] = useState('summary')
  const summary = useFinanceQuery<RevenueSummary>(
    'revenue/summary',
    props.params
  )
  const format = useFinanceFormat()
  if (summary.isPending) return <LoadingState />
  if (summary.isError) {
    return <ErrorState onRetry={() => void summary.refetch()} />
  }
  return (
    <div className='space-y-6'>
      <FinanceMetrics
        items={[
          {
            label: 'Paid recharge',
            value: format.amount(
              summary.data.paid_recharge_total[props.unit],
              props.unit
            ),
          },
          {
            label: 'Gifts granted',
            value: format.amount(
              summary.data.gift_granted_total[props.unit],
              props.unit
            ),
          },
          {
            label: 'Unredeemed paid vouchers',
            value: format.amount(
              summary.data.unissued_paid_voucher_total[props.unit],
              props.unit
            ),
          },
          {
            label: 'Unredeemed gift vouchers',
            value: format.amount(
              summary.data.unredeemed_gift_voucher_total[props.unit],
              props.unit
            ),
          },
        ]}
      />
      <FinanceRemoteTable
        key={`paid-${paidView}`}
        title='Paid recharge sources'
        path='revenue/paid-sources'
        params={{ ...props.params, view: paidView }}
        unit={props.unit}
        extra={
          <FinanceSegments
            label='Paid source view'
            value={paidView}
            onChange={setPaidView}
            items={views}
          />
        }
        columns={
          paidView === 'summary'
            ? [
                { key: 'source_label', label: 'Source' },
                { key: 'recharge_count', label: 'Recharges', kind: 'number' },
                { key: 'user_count', label: 'Customers', kind: 'number' },
                ...amountColumns,
              ]
            : [
                { key: 'created_at', label: 'Time', kind: 'date' },
                { key: 'username', label: 'Customer' },
                { key: 'source_label', label: 'Source' },
                ...amountColumns,
                { key: 'remark', label: 'Remark' },
              ]
        }
      />
      <FinanceRemoteTable
        key={`gift-${giftView}`}
        title='Gift records'
        path={
          giftView === 'summary'
            ? 'revenue/gift-audit-summary'
            : 'revenue/gift-audit'
        }
        params={props.params}
        unit={props.unit}
        extra={
          <FinanceSegments
            label='Gift view'
            value={giftView}
            onChange={setGiftView}
            items={views}
          />
        }
        columns={
          giftView === 'summary'
            ? [
                { key: 'source_label', label: 'Source' },
                { key: 'activity_description', label: 'Activity' },
                { key: 'exchange_count', label: 'Count', kind: 'number' },
                {
                  key: 'granted_amount_usd',
                  cosKey: 'granted_amount_cos',
                  label: 'Gift amount',
                  kind: 'money',
                },
              ]
            : [
                { key: 'created_at', label: 'Time', kind: 'date' },
                { key: 'username', label: 'Customer' },
                { key: 'source_label', label: 'Source' },
                {
                  key: 'granted_amount_usd',
                  cosKey: 'granted_amount_cos',
                  label: 'Gift amount',
                  kind: 'money',
                },
                { key: 'operator_username', label: 'Operator' },
                { key: 'voucher_code', label: 'Voucher' },
                { key: 'remark', label: 'Remark' },
              ]
        }
      />
    </div>
  )
}
