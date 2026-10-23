import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'

import { useFinanceQuery } from '../api'
import type {
  DashboardSummary,
  FinanceParams,
  FinanceRow,
  FinanceUnit,
} from '../types'
import { useFinanceFormat } from '../use-finance-format'
import { FinanceMetrics, FinanceRows, FinanceSelect } from './shared'

export function FinanceOverview(props: {
  params: FinanceParams
  unit: FinanceUnit
}) {
  const { t } = useTranslation()
  const [ranking, setRanking] = useState('income')
  const format = useFinanceFormat()
  const summary = useFinanceQuery<DashboardSummary>(
    'dashboard/summary',
    props.params
  )
  const todos = useFinanceQuery<{ items: FinanceRow[] }>('dashboard/todos', {
    ...props.params,
    limit: 20,
  })
  const rankings = useFinanceQuery<{ items: FinanceRow[] }>(
    'dashboard/rankings',
    { ...props.params, view: ranking, limit: 20 }
  )
  if (summary.isPending) return <LoadingState />
  if (summary.isError) {
    return <ErrorState onRetry={() => void summary.refetch()} />
  }
  const kpis = summary.data.kpis
  return (
    <div className='space-y-6'>
      <FinanceMetrics
        items={[
          {
            label: 'Sales income',
            value: format.amount(kpis.sales_income[props.unit], props.unit),
          },
          {
            label: 'Gifts granted',
            value: format.amount(kpis.gift_granted[props.unit], props.unit),
          },
          {
            label: 'Estimated channel cost',
            value: format.amount(kpis.channel_cost[props.unit], props.unit),
          },
          {
            label: 'Paid balance',
            value: format.amount(
              kpis.current_paid_balance[props.unit],
              props.unit
            ),
          },
          {
            label: 'Gift balance',
            value: format.amount(
              kpis.current_gift_balance[props.unit],
              props.unit
            ),
          },
          {
            label: 'Active customers',
            value: format.number(kpis.active_customer_count),
          },
          {
            label: 'Paying customers',
            value: format.number(kpis.paid_customer_count),
          },
        ]}
      />
      {summary.data.alerts?.length > 0 && (
        <section className='space-y-3'>
          <h3 className='text-sm font-semibold'>{t('Finance alerts')}</h3>
          <FinanceRows
            rows={summary.data.alerts}
            columns={[
              { key: 'title', label: 'Alert' },
              { key: 'abnormal_value', label: 'Current value', kind: 'number' },
              { key: 'baseline_value', label: 'Baseline', kind: 'number' },
              { key: 'suggested_action', label: 'Suggested action' },
            ]}
          />
        </section>
      )}
      <section className='space-y-3'>
        <h3 className='text-sm font-semibold'>{t('Follow-up items')}</h3>
        {todos.isPending && <LoadingState />}
        {todos.isError && <ErrorState onRetry={() => void todos.refetch()} />}
        {todos.isSuccess && (
          <FinanceRows
            rows={todos.data.items ?? []}
            columns={[
              { key: 'type', label: 'Type', kind: 'label' },
              { key: 'target_name', label: 'Customer' },
              { key: 'abnormal_value', label: 'Current value' },
              { key: 'suggested_action', label: 'Suggested action' },
              { key: 'last_occurred_at', label: 'Last activity', kind: 'date' },
            ]}
          />
        )}
      </section>
      <section className='space-y-3'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <h3 className='text-sm font-semibold'>{t('Rankings')}</h3>
          <FinanceSelect
            label='Ranking metric'
            value={ranking}
            onChange={setRanking}
            items={[
              { value: 'income', label: 'Sales income' },
              { value: 'usage', label: 'Total consumption' },
              { value: 'paid_usage', label: 'Paid consumption' },
              { value: 'channel_cost', label: 'Estimated channel cost' },
            ]}
          />
        </div>
        {rankings.isPending && <LoadingState />}
        {rankings.isError && (
          <ErrorState onRetry={() => void rankings.refetch()} />
        )}
        {rankings.isSuccess && (
          <FinanceRows
            rows={rankings.data.items ?? []}
            unit={props.unit}
            columns={[
              { key: 'rank', label: 'Rank', kind: 'number' },
              { key: 'target_name', label: 'Name' },
              {
                key: 'value_usd',
                cosKey: 'value_cos',
                label: 'Amount',
                kind: 'money',
              },
              { key: 'extra', label: 'Details' },
            ]}
          />
        )}
      </section>
    </div>
  )
}
