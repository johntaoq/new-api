import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Save } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { hasPermission } from '@/lib/admin-permissions'
import { useAuthStore } from '@/stores/auth-store'

import { setMonthlyRate, useFinanceQuery } from '../api'
import type { FinanceMonthlyPolicy } from '../types'
import { useFinanceFormat } from '../use-finance-format'

export function FinanceMonthlyPolicySettings() {
  const { t } = useTranslation()
  const format = useFinanceFormat()
  const user = useAuthStore((state) => state.auth.user)
  const canEdit = hasPermission(user, 'finance', 'write')
  const query = useFinanceQuery<FinanceMonthlyPolicy>('monthly-billing', {})
  const client = useQueryClient()
  const [draft, setDraft] = useState<string | null>(null)
  const [confirm, setConfirm] = useState(false)
  const current = query.data?.months?.find(
    (month) => month.bill_month === query.data?.current_month
  )
  const next = query.data?.months?.find(
    (month) => month.bill_month === query.data?.next_month
  )
  const value = draft ?? String(next?.cny_rate ?? current?.cny_rate ?? '')
  const rate = Number(value)
  const valid =
    Number.isFinite(rate) &&
    rate > 0 &&
    rate <= 1000000 &&
    /^\d+(\.\d{1,8})?$/.test(value)
  const save = useMutation({
    mutationFn: () => {
      if (!query.data) throw new Error('Monthly policy is unavailable')
      return setMonthlyRate(query.data.next_month, rate)
    },
    onSuccess: () => {
      setConfirm(false)
      setDraft(null)
      toast.success(t('Next month exchange rate saved'))
      void client.invalidateQueries({ queryKey: ['finance'] })
    },
  })
  if (query.isPending) return <LoadingState />
  if (query.isError) return <ErrorState onRetry={() => void query.refetch()} />
  return (
    <section
      className='space-y-3 border-b pb-5'
      aria-label={t('Monthly exchange rate')}
    >
      <div className='flex flex-wrap gap-x-6 gap-y-2 text-sm'>
        <span>
          {t('Current month')}: {query.data.current_month}
        </span>
        <span>
          {t('Fixed exchange rate')}: 1 USD = {format.number(current?.cny_rate)}{' '}
          CNY
        </span>
        <span>
          {t('Next statement')}:{' '}
          {current ? format.billingDate(current.closes_at) : '-'} UTC+08:00
        </span>
      </div>
      {query.data.months?.some((month) => month.last_error) && (
        <p role='alert' className='text-destructive text-sm'>
          {t('Monthly billing failed; automatic retry pending')}
        </p>
      )}
      {canEdit && (
        <form
          className='flex flex-wrap items-end gap-2'
          onSubmit={(event) => {
            event.preventDefault()
            if (valid) setConfirm(true)
          }}
        >
          <div className='space-y-1.5'>
            <Label htmlFor='finance-next-month-rate'>
              {t('Next month exchange rate')} · {query.data.next_month}
            </Label>
            <Input
              id='finance-next-month-rate'
              type='number'
              min='0.00000001'
              max='1000000'
              step='0.00000001'
              value={value}
              onChange={(event) => setDraft(event.target.value)}
              className='w-52'
              required
            />
          </div>
          <Button
            type='submit'
            variant='outline'
            disabled={!valid || save.isPending}
          >
            <Save />
            {t('Save')}
          </Button>
        </form>
      )}
      <ConfirmDialog
        open={confirm}
        onOpenChange={setConfirm}
        title={t('Next month exchange rate')}
        desc={`${query.data.next_month} · 1 USD = ${value} CNY`}
        confirmText={t('Confirm')}
        isLoading={save.isPending}
        handleConfirm={() => save.mutate()}
      />
    </section>
  )
}
