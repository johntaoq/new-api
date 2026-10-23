/*
Copyright (C) 2023-2026 QuantumNous

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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { toIntlLocale } from '@/i18n/languages'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import {
  walletAdjustmentReasons,
  type WalletAdjustmentDraft,
} from '../lib/wallet-adjustments'

export function UserQuotaDialog(props: {
  fundingType: 'paid' | 'gift'
  draft?: WalletAdjustmentDraft
  currentQuota: number
  username: string
  onClose: () => void
  onSave: (draft: WalletAdjustmentDraft) => void
  canSave: (draft: WalletAdjustmentDraft) => boolean
}) {
  const { t, i18n } = useTranslation()
  const config = useSystemConfigStore((s) => s.config.currency)
  const root = useAuthStore((s) => s.auth.user?.role === ROLE.SUPER_ADMIN)
  const paid = props.fundingType === 'paid'
  const [currency, setCurrency] = useState<'CNY' | 'USD'>(
    props.draft?.currency ?? (paid ? 'CNY' : 'USD')
  )
  const [amount, setAmount] = useState(props.draft?.amount ?? '')
  const [source, setSource] = useState(
    props.draft?.source_type ?? (paid ? 'manual_topup' : 'admin_grant')
  )
  const [remark, setRemark] = useState(props.draft?.remark ?? '')
  const [revenue, setRevenue] = useState(
    props.draft ? String(props.draft.revenue_usd) : ''
  )
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const usd = (value: number) =>
    new Intl.NumberFormat(locale, {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 6,
    }).format(value)
  const money = (quota: number) =>
    formatQuotaWithCurrency(quota, {
      digitsLarge: 6,
      digitsSmall: 6,
      abbreviate: false,
      locale,
    })
  const deltaUSD =
    currency === 'CNY'
      ? Number(amount) / config.usdExchangeRate
      : Number(amount)
  const deltaQuota =
    Math.sign(deltaUSD) * Math.round(Math.abs(deltaUSD) * config.quotaPerUnit)
  const recognized = paid
    ? Number(revenue === '' ? deltaUSD.toFixed(6) : revenue)
    : 0
  const draft: WalletAdjustmentDraft = {
    id: props.draft?.id ?? 0,
    funding_type: props.fundingType,
    delta_quota: deltaQuota,
    source_type: source,
    remark,
    revenue_usd: recognized,
    currency,
    amount,
  }
  const reasons = walletAdjustmentReasons[props.fundingType].filter(
    ([value]) => value !== 'system_adjustment' || root
  )
  const valid =
    amount.trim() !== '' &&
    Number.isFinite(config.quotaPerUnit) &&
    config.quotaPerUnit > 0 &&
    Number.isSafeInteger(deltaQuota) &&
    deltaQuota !== 0 &&
    Number.isFinite(recognized) &&
    (currency !== 'CNY' ||
      (Number.isFinite(config.usdExchangeRate) &&
        config.usdExchangeRate > 0)) &&
    (!paid || root) &&
    reasons.some(([value]) => value === source) &&
    props.canSave(draft)
  const after =
    props.currentQuota + (Number.isSafeInteger(deltaQuota) ? deltaQuota : 0)

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
      title={t(paid ? 'Adjust paid balance' : 'Adjust gift balance')}
      description={props.username}
      bodyClassName='space-y-4'
      contentClassName='sm:max-w-[760px]'
      footer={
        <>
          <Button variant='outline' onClick={props.onClose}>
            {t('Cancel')}
          </Button>
          <Button disabled={!valid} onClick={() => props.onSave(draft)}>
            {t(props.draft ? 'Update pending' : 'Add to pending')}
          </Button>
        </>
      }
    >
      <dl
        className='grid gap-3 border-y py-4 text-sm sm:grid-cols-2'
        aria-live='polite'
      >
        <div className='min-w-0'>
          <dt className='text-muted-foreground'>{t('Current balance')}</dt>
          <dd className='mt-1 text-lg font-semibold break-words'>
            {money(props.currentQuota)}
          </dd>
          <dd className='text-muted-foreground'>
            {t('USD equivalent')}:{' '}
            {usd(props.currentQuota / config.quotaPerUnit)}
          </dd>
        </div>
        <div className='min-w-0'>
          <dt className='text-muted-foreground'>
            {t('Balance after adjustment')}
          </dt>
          <dd
            aria-label={t('Balance after adjustment')}
            className='mt-1 text-lg font-semibold break-words'
          >
            {money(after)}
          </dd>
          <dd className='text-muted-foreground'>
            {t('USD equivalent')}: {usd(after / config.quotaPerUnit)}
          </dd>
        </div>
      </dl>
      <div className='space-y-2'>
        <Label htmlFor='wallet-reason'>
          {t(paid ? 'Adjustment reason' : 'Source category')}
        </Label>
        <Select
          value={source}
          onValueChange={(value) => setSource(value ?? '')}
          items={reasons.map(([value, label]) => ({ value, label: t(label) }))}
        >
          <SelectTrigger id='wallet-reason' className='w-full'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {reasons.map(([value, label]) => (
              <SelectItem key={value} value={value}>
                {t(label)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className='space-y-2'>
        <Label htmlFor='wallet-currency'>{t('Input currency')}</Label>
        <Select
          value={currency}
          onValueChange={(value) => {
            if (value === 'CNY' || value === 'USD') setCurrency(value)
          }}
          items={[
            { value: 'CNY', label: 'CNY' },
            { value: 'USD', label: 'USD' },
          ]}
        >
          <SelectTrigger id='wallet-currency' className='w-full'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='CNY'>CNY</SelectItem>
            <SelectItem value='USD'>USD</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className='space-y-2'>
        <Label htmlFor='wallet-amount'>
          {t('Input amount')} ({currency})
        </Label>
        <Input
          id='wallet-amount'
          type='number'
          step='any'
          value={amount}
          onChange={(event) => {
            setAmount(event.target.value)
            if (paid && (source === 'manual_topup' || source === 'refund')) {
              setSource(
                Number(event.target.value) < 0 ? 'refund' : 'manual_topup'
              )
            }
          }}
          placeholder={t('Positive to add, negative to deduct')}
        />
      </div>
      <div className='space-y-2'>
        <Label htmlFor='wallet-usd'>{t('Adjustment amount')} (USD)</Label>
        <Input
          id='wallet-usd'
          type='number'
          readOnly
          value={
            amount !== '' && Number.isFinite(deltaUSD)
              ? deltaUSD.toFixed(6)
              : ''
          }
        />
        <p className='text-muted-foreground text-xs'>
          {currency === 'CNY'
            ? `1 USD = ${config.usdExchangeRate} CNY`
            : t('USD equivalent')}
          {' · '}
          {t('Display amount')}:{' '}
          {money(Number.isSafeInteger(deltaQuota) ? deltaQuota : 0)}
        </p>
      </div>
      {paid && (
        <div className='space-y-2'>
          <Label htmlFor='wallet-revenue'>
            {t('Recognized revenue')} (USD)
          </Label>
          <Input
            id='wallet-revenue'
            type='number'
            step='any'
            value={
              revenue === '' && amount !== '' && Number.isFinite(recognized)
                ? String(recognized)
                : revenue
            }
            onChange={(event) => setRevenue(event.target.value)}
          />
        </div>
      )}
      <div className='space-y-2'>
        <Label htmlFor='wallet-remark'>{t('Remark')}</Label>
        <Textarea
          id='wallet-remark'
          rows={3}
          value={remark}
          maxLength={255}
          onChange={(event) => setRemark(event.target.value)}
        />
      </div>
      {amount !== '' && !valid && (
        <p role='alert' className='text-destructive text-sm'>
          {t('Check the amount and pending wallet balances')}
        </p>
      )}
    </Dialog>
  )
}
