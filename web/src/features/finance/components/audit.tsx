import { Search } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import type { FinanceParams, FinanceRow } from '../types'
import { FinanceRemoteTable, FinanceSelect } from './shared'

function snapshotText(value: unknown): string {
  if (!value) return '-'
  try {
    return JSON.stringify(JSON.parse(String(value)), null, 2)
  } catch {
    return String(value)
  }
}

export function FinanceAudit(props: { params: FinanceParams }) {
  const { t } = useTranslation()
  const [form, setForm] = useState({
    module: 'all',
    action: 'all',
    operator_keyword: '',
    target_keyword: '',
  })
  const [filters, setFilters] = useState<FinanceParams>({})
  const [selected, setSelected] = useState<FinanceRow | null>(null)
  return (
    <div className='space-y-5'>
      <form
        className='flex flex-wrap gap-2'
        onSubmit={(event) => {
          event.preventDefault()
          setSelected(null)
          setFilters({
            ...form,
            module: form.module === 'all' ? '' : form.module,
            action: form.action === 'all' ? '' : form.action,
          })
        }}
      >
        <FinanceSelect
          label='Audit module'
          value={form.module}
          onChange={(module) => setForm({ ...form, module })}
          items={[
            { value: 'all', label: 'All modules' },
            { value: 'redemption', label: 'Redemption Codes' },
            { value: 'user_quota', label: 'User quota' },
          ]}
        />
        <FinanceSelect
          label='Audit action'
          value={form.action}
          onChange={(action) => setForm({ ...form, action })}
          items={[
            { value: 'all', label: 'All actions' },
            { value: 'create', label: 'Create' },
            { value: 'update', label: 'Update' },
            { value: 'delete', label: 'Delete' },
            { value: 'cleanup_invalid', label: 'Delete invalid codes' },
            { value: 'adjust', label: 'Adjustment' },
          ]}
        />
        <Input
          aria-label={t('Operator')}
          placeholder={t('Operator')}
          className='w-full sm:w-44'
          value={form.operator_keyword}
          onChange={(event) =>
            setForm({ ...form, operator_keyword: event.target.value })
          }
        />
        <Input
          aria-label={t('Target customer')}
          placeholder={t('Target customer')}
          className='w-full sm:w-44'
          value={form.target_keyword}
          onChange={(event) =>
            setForm({ ...form, target_keyword: event.target.value })
          }
        />
        <Button type='submit' variant='outline'>
          <Search />
          {t('Search')}
        </Button>
      </form>
      <FinanceRemoteTable
        key={JSON.stringify(filters)}
        title='Financial audit'
        path='audit/logs'
        params={{ ...props.params, ...filters }}
        action={{ label: 'View changes', onClick: setSelected }}
        columns={[
          { key: 'created_at', label: 'Time', kind: 'date' },
          { key: 'module', label: 'Module', kind: 'label' },
          { key: 'action', label: 'Action', kind: 'label' },
          { key: 'operator_username', label: 'Operator' },
          { key: 'target_type', label: 'Target', kind: 'label' },
          { key: 'target_id', label: 'Target ID' },
          { key: 'target_username', label: 'Target customer' },
          { key: 'remark', label: 'Remark' },
        ]}
      />
      <Dialog
        title={t('Financial changes')}
        open={selected !== null}
        onOpenChange={(open) => {
          if (!open) setSelected(null)
        }}
        contentClassName='sm:max-w-4xl'
      >
        <div className='grid min-w-0 gap-4 md:grid-cols-2'>
          {['before_json', 'after_json'].map((key) => (
            <section key={key} className='min-w-0 space-y-2'>
              <h3 className='text-sm font-semibold'>
                {key === 'before_json' ? t('Before') : t('After')}
              </h3>
              <pre className='bg-muted max-h-96 overflow-auto rounded-md p-3 text-xs break-all whitespace-pre-wrap'>
                {snapshotText(selected?.[key])}
              </pre>
            </section>
          ))}
        </div>
      </Dialog>
    </div>
  )
}
