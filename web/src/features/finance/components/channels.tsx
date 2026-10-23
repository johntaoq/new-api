import { useMutation } from '@tanstack/react-query'
import { Download } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Pie, PieChart, Tooltip } from 'recharts'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { ChartContainer } from '@/components/ui/chart'

import { exportCostReport, useFinanceQuery } from '../api'
import type { ChannelSummary, FinanceColumn, FinanceParams } from '../types'
import { useFinanceFormat } from '../use-finance-format'
import { FinanceMetrics, FinanceRemoteTable, FinanceSegments } from './shared'

const costColumns: FinanceColumn[] = [
  { key: 'prompt_tokens', label: 'Input tokens', kind: 'number' },
  { key: 'completion_tokens', label: 'Output tokens', kind: 'number' },
  { key: 'cache_tokens', label: 'Cache tokens', kind: 'number' },
  { key: 'total_tokens', label: 'Total tokens', kind: 'number' },
  { key: 'revenue_usd', label: 'Recognized revenue', kind: 'money' },
  { key: 'cost_usd', label: 'Estimated cost', kind: 'money' },
]
const colors = [
  '#0d9488',
  '#2563eb',
  '#e11d48',
  '#ca8a04',
  '#64748b',
  '#7c3aed',
]

function CostChart(props: {
  title: string
  data: { name: string; value: number }[]
  metric: string
}) {
  const { t } = useTranslation()
  const format = useFinanceFormat()
  const data = props.data.map((row, index) => ({
    ...row,
    fill: colors[index % colors.length],
  }))
  return (
    <section className='min-w-0 space-y-3'>
      <h3 className='text-sm font-semibold'>{t(props.title)}</h3>
      {data.length === 0 ? (
        <EmptyState title={t('No data')} />
      ) : (
        <>
          <ChartContainer
            className='h-60 w-full'
            config={{ value: { label: t(props.title) } }}
          >
            <PieChart accessibilityLayer>
              <Pie
                data={data}
                dataKey='value'
                nameKey='name'
                innerRadius={55}
                outerRadius={85}
                isAnimationActive={false}
              />
              <Tooltip
                formatter={(value) =>
                  props.metric === 'usd'
                    ? format.amount(value)
                    : format.number(value)
                }
              />
            </PieChart>
          </ChartContainer>
          <ul className='flex flex-wrap gap-x-4 gap-y-2 text-xs'>
            {data.map((row) => (
              <li key={row.name} className='flex min-w-0 items-center gap-1.5'>
                <span
                  className='size-2 shrink-0 rounded-sm'
                  style={{ backgroundColor: row.fill }}
                  aria-hidden
                />
                <span className='break-all'>{row.name}</span>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  )
}

function CostDownload(props: {
  view: 'channels' | 'models'
  params: FinanceParams
}) {
  const { t } = useTranslation()
  const download = useMutation({
    mutationFn: () => exportCostReport(props.view, props.params),
  })
  return (
    <Button
      variant='outline'
      size='sm'
      disabled={download.isPending}
      onClick={() => download.mutate()}
    >
      <Download />
      {t('Export CSV')}
    </Button>
  )
}

export function FinanceChannels(props: { params: FinanceParams }) {
  const [metric, setMetric] = useState('usd')
  const summary = useFinanceQuery<ChannelSummary>(
    'channel-cost/summary',
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
            label: 'Estimated channel cost',
            value: format.amount(summary.data.kpis.total_cost_usd),
          },
          {
            label: 'Total tokens',
            value: format.number(summary.data.kpis.total_tokens),
          },
        ]}
      />
      <div className='flex justify-end'>
        <FinanceSegments
          label='Chart metric'
          value={metric}
          onChange={setMetric}
          items={[
            { value: 'usd', label: 'USD' },
            { value: 'tokens', label: 'Tokens' },
          ]}
        />
      </div>
      <div className='grid min-w-0 gap-6 lg:grid-cols-2'>
        <CostChart
          title='Cost by channel'
          metric={metric}
          data={
            summary.data.charts[
              metric === 'usd' ? 'by_channel_usd' : 'by_channel_tokens'
            ] ?? []
          }
        />
        <CostChart
          title='Cost by model'
          metric={metric}
          data={
            summary.data.charts[
              metric === 'usd' ? 'by_model_usd' : 'by_model_tokens'
            ] ?? []
          }
        />
      </div>
      <FinanceRemoteTable
        title='Channel costs'
        path='channel-cost/channels'
        params={props.params}
        extra={<CostDownload view='channels' params={props.params} />}
        columns={[
          { key: 'channel_name', label: 'Channel' },
          { key: 'provider_snapshot', label: 'Provider' },
          ...costColumns,
        ]}
      />
      <FinanceRemoteTable
        title='Model costs'
        path='channel-cost/models'
        params={props.params}
        extra={<CostDownload view='models' params={props.params} />}
        columns={[{ key: 'model_name', label: 'Model' }, ...costColumns]}
      />
    </div>
  )
}
