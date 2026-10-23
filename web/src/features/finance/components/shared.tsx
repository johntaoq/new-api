import {
  getCoreRowModel,
  useReactTable,
  type PaginationState,
} from '@tanstack/react-table'
import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTablePagination, StaticDataTable } from '@/components/data-table'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { useAuthStore } from '@/stores/auth-store'

import { useFinanceQuery } from '../api'
import type {
  FinanceColumn,
  FinancePage,
  FinanceParams,
  FinanceRow,
  StatementUnit,
} from '../types'
import { useFinanceFormat } from '../use-finance-format'

export function FinanceSelect(props: {
  label: string
  value: string
  onChange: (value: string) => void
  items: { value: string; label: string }[]
}) {
  const { t } = useTranslation()
  const items = props.items.map((item) => ({ ...item, label: t(item.label) }))
  return (
    <Select
      value={props.value}
      onValueChange={(value) => value != null && props.onChange(value)}
      items={items}
    >
      <SelectTrigger
        aria-label={t(props.label)}
        className='w-full min-w-0 sm:w-auto sm:min-w-36'
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {items.map((item) => (
          <SelectItem value={item.value} key={item.value}>
            {item.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export function FinanceSegments(props: {
  label: string
  value: string
  onChange: (value: string) => void
  items: { value: string; label: string }[]
}) {
  const { t } = useTranslation()
  return (
    <ToggleGroup
      aria-label={t(props.label)}
      value={[props.value]}
      onValueChange={(values) => values[0] && props.onChange(String(values[0]))}
      variant='outline'
      size='sm'
      className='max-w-full shrink-0'
    >
      {props.items.map((item) => (
        <ToggleGroupItem key={item.value} value={item.value}>
          {t(item.label)}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  )
}

export function FinanceMetrics(props: {
  items: { label: string; value: string }[]
}) {
  const { t } = useTranslation()
  return (
    <dl className='grid grid-cols-2 gap-x-5 gap-y-5 border-y py-5 lg:grid-cols-4'>
      {props.items.map((item) => (
        <div key={item.label} className='min-w-0'>
          <dt className='text-muted-foreground text-xs'>{t(item.label)}</dt>
          <dd className='mt-1.5 text-lg font-semibold [overflow-wrap:anywhere] tabular-nums'>
            {item.value}
          </dd>
        </div>
      ))}
    </dl>
  )
}

export function FinanceRows(props: {
  rows: FinanceRow[]
  columns: FinanceColumn[]
  unit?: StatementUnit
  action?: { label: string; onClick: (row: FinanceRow) => void }
}) {
  const { t } = useTranslation()
  const format = useFinanceFormat()
  const columns = props.columns.map((column) => ({
    id: column.key,
    header: t(column.label),
    className:
      column.kind === 'money' || column.kind === 'number' ? 'text-right' : '',
    cellClassName: 'max-w-64 align-top whitespace-normal break-words',
    cell: (row: FinanceRow) => {
      const value = row[column.key]
      if (column.kind === 'money') {
        let displayed = value
        if (props.unit === 'cos' && column.cosKey) {
          displayed = row[column.cosKey]
        }
        if (props.unit === 'cny') {
          displayed = column.cnyKey ? row[column.cnyKey] : null
        }
        return (
          <div className='text-right tabular-nums'>
            {format.amount(displayed, props.unit)}
          </div>
        )
      }
      if (column.kind === 'number') {
        return (
          <div className='text-right tabular-nums'>{format.number(value)}</div>
        )
      }
      if (column.kind === 'date') return format.date(value)
      if (column.kind === 'label') return format.label(value)
      return String(value ?? '-')
    },
  }))
  if (props.action) {
    const action = props.action
    columns.push({
      id: 'actions',
      header: t('Actions'),
      className: '',
      cellClassName: '',
      cell: (row) => (
        <Button variant='ghost' size='sm' onClick={() => action.onClick(row)}>
          {t(action.label)}
        </Button>
      ),
    })
  }
  return (
    <StaticDataTable
      columns={columns}
      data={props.rows}
      tableClassName='min-w-[560px]'
      headerRowClassName='bg-blue-50 dark:bg-blue-950/50 [&>th]:font-semibold [&>th]:text-blue-950 dark:[&>th]:text-blue-100'
      emptyContent={<EmptyState title={t('No data')} className='min-h-32' />}
    />
  )
}

export function FinanceRemoteTable(props: {
  title: string
  path: string
  params: FinanceParams
  columns: FinanceColumn[]
  unit?: StatementUnit
  action?: { label: string; onClick: (row: FinanceRow) => void }
  extra?: ReactNode
  summary?: (page: FinancePage) => ReactNode
}) {
  const { t } = useTranslation()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const query = useFinanceQuery<FinancePage>(
    props.path,
    {
      ...props.params,
      p: pagination.pageIndex + 1,
      page_size: pagination.pageSize,
    },
    true,
    true
  )
  const userID = useAuthStore((state) => state.auth.user?.id)
  const reportKey = JSON.stringify([userID, props.path, props.params])
  const [lastPage, setLastPage] = useState<{
    reportKey: string
    data: FinancePage
  }>()
  useEffect(() => {
    if (query.data && !query.isPlaceholderData) {
      setLastPage({ reportKey, data: query.data })
    }
  }, [query.data, query.isPlaceholderData, reportKey])
  const page =
    query.data ??
    (lastPage?.reportKey === reportKey ? lastPage.data : undefined)
  const table = useReactTable({
    data: page?.items ?? [],
    columns: [],
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    rowCount: page?.total ?? 0,
    state: { pagination },
    onPaginationChange: setPagination,
  })
  return (
    <section
      className='min-w-0 space-y-3'
      aria-label={t(props.title)}
      aria-busy={query.isFetching}
    >
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <h3 className='text-sm font-semibold'>{t(props.title)}</h3>
        {props.extra}
      </div>
      {query.isPending && !page && <LoadingState />}
      {query.isError && <ErrorState onRetry={() => void query.refetch()} />}
      {page && (
        <>
          {props.summary?.(page)}
          <div
            style={{
              minHeight: Math.min(pagination.pageSize, page.total) * 60 + 41,
            }}
          >
            <FinanceRows
              rows={page.items ?? []}
              columns={props.columns}
              unit={props.unit}
              action={props.action}
            />
          </div>
          <DataTablePagination table={table} />
        </>
      )}
    </section>
  )
}
