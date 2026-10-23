import { useMutation } from '@tanstack/react-query'
import { Download, Search } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import dayjs from '@/lib/dayjs'

import { exportStatement } from '../api'
import type { FinanceRow, StatementUnit } from '../types'
import { FinanceMonthlyPolicySettings } from './monthly-policy'
import { FinanceRemoteTable, FinanceSegments } from './shared'

export function FinanceCustomers(props: {
  unit?: StatementUnit
  self?: boolean
}) {
  const { t } = useTranslation()
  const [unit, setUnit] = useState<StatementUnit>(props.unit ?? 'cny')
  const [month, setMonth] = useState(
    dayjs().subtract(1, 'month').format('YYYY-MM')
  )
  const [keyword, setKeyword] = useState('')
  const [query, setQuery] = useState({ bill_month: month, user_keyword: '' })
  const [selected, setSelected] = useState<FinanceRow | null>(null)
  const path = props.self ? 'self/statements' : 'customer-bills/summary'
  const download = useMutation({
    mutationFn: () =>
      exportStatement(
        {
          bill_month: String(selected?.bill_month ?? query.bill_month),
          user_id: Number(selected?.user_id),
        },
        props.self
      ),
  })
  return (
    <div className='min-w-0 space-y-5'>
      {!props.self && <FinanceMonthlyPolicySettings />}
      <form
        className='flex flex-wrap items-center gap-2'
        onSubmit={(event) => {
          event.preventDefault()
          setSelected(null)
          setQuery({ bill_month: month, user_keyword: keyword.trim() })
        }}
      >
        <Input
          type='month'
          required
          aria-label={t('Bill month')}
          value={month}
          onChange={(event) => setMonth(event.target.value)}
          className='w-full sm:w-44'
        />
        {!props.self && (
          <Input
            aria-label={t('Customer search')}
            placeholder={t('Username or user ID')}
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            className='w-full sm:w-60'
          />
        )}
        <Button type='submit' variant='outline'>
          <Search />
          {t('Search')}
        </Button>
        {!props.self && (
          <FinanceSegments
            label='Statement currency'
            value={unit}
            onChange={(value) => setUnit(value as StatementUnit)}
            items={[
              { value: 'cny', label: 'CNY' },
              { value: 'usd', label: 'USD' },
              { value: 'cos', label: 'COS' },
            ]}
          />
        )}
      </form>
      <FinanceRemoteTable
        key={JSON.stringify(query)}
        title='Finalized monthly statements'
        path={path}
        params={props.self ? { bill_month: query.bill_month } : query}
        unit={unit}
        action={{ label: 'View details', onClick: setSelected }}
        columns={[
          { key: 'bill_month', label: 'Bill month' },
          ...(!props.self ? [{ key: 'username', label: 'Customer' }] : []),
          { key: 'cny_rate', label: 'USD/CNY', kind: 'number' },
          {
            key: 'total_consume_usd',
            cosKey: 'total_consume_cos',
            cnyKey: 'total_consume_cny',
            label: 'Total consumption',
            kind: 'money',
          },
          {
            key: 'paid_consume_usd',
            cosKey: 'paid_consume_cos',
            cnyKey: 'paid_consume_cny',
            label: 'Paid consumption',
            kind: 'money',
          },
          {
            key: 'gift_consume_usd',
            cosKey: 'gift_consume_cos',
            cnyKey: 'gift_consume_cny',
            label: 'Gift consumption',
            kind: 'money',
          },
          { key: 'generated_at', label: 'Generated at', kind: 'date' },
        ]}
      />
      <Dialog
        open={selected !== null}
        onOpenChange={(open) => {
          if (!open) setSelected(null)
        }}
        title={`${t('Customer statement')} · ${String(selected?.username ?? '')} · ${String(selected?.bill_month ?? query.bill_month)}`}
        contentClassName='sm:max-w-6xl'
        footer={
          <Button
            disabled={download.isPending || !selected}
            onClick={() => download.mutate()}
          >
            <Download />
            {t('Export CSV')}
          </Button>
        }
      >
        {selected && (
          <FinanceRemoteTable
            key={`${query.bill_month}-${selected.user_id}`}
            title='Statement details'
            path={
              props.self ? 'self/statements/details' : 'customer-bills/details'
            }
            params={
              props.self
                ? { bill_month: String(selected.bill_month) }
                : {
                    bill_month: String(selected.bill_month),
                    user_id: Number(selected.user_id),
                  }
            }
            unit={unit}
            columns={[
              { key: 'occurred_at', label: 'Time', kind: 'date' },
              { key: 'entry_type', label: 'Type', kind: 'label' },
              { key: 'token_display', label: 'Token' },
              { key: 'model_name', label: 'Model' },
              {
                key: 'amount_usd',
                cosKey: 'amount_cos',
                cnyKey: 'amount_cny',
                label: 'Balance change',
                kind: 'money',
              },
              { key: 'request_id', label: 'Request ID' },
            ]}
          />
        )}
      </Dialog>
    </div>
  )
}
