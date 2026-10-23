import { useTranslation } from 'react-i18next'

import { toIntlLocale } from '@/i18n/languages'
import dayjs from '@/lib/dayjs'

import type { StatementUnit } from './types'

const labels: Record<string, string> = {
  consume: 'Consumption',
  refund: 'Refund',
  topup: 'Top-up',
  gift: 'Gift',
  adjustment: 'Adjustment',
  redemption: 'Redemption Codes',
  user_quota: 'User quota',
  create: 'Create',
  update: 'Update',
  delete: 'Delete',
  cleanup_invalid: 'Delete invalid codes',
  adjust: 'Adjustment',
  balance_alert: 'Low balance',
  inactive_balance: 'Inactive balance',
  redemption_batch: 'Redemption batch',
  user_wallet: 'User wallet',
}

export function useFinanceFormat() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  // Fixed statements never consult mutable wallet display settings.
  const usd = new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  const cos = new Intl.NumberFormat(locale, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  const cny = new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  const billingDate = new Intl.DateTimeFormat('sv-SE', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
  return {
    number: (value: unknown) => {
      const numeric = Number(value ?? 0)
      return Number.isFinite(numeric) ? cos.format(numeric) : '-'
    },
    amount: (value: unknown, unit: StatementUnit = 'usd') => {
      if (value == null) return '-'
      if (unit === 'cny') return cny.format(Number(value))
      if (unit === 'cos') return `${cos.format(Number(value))} COS`
      return usd.format(Number(value))
    },
    label: (value: unknown) => t(labels[String(value)] ?? String(value ?? '-')),
    date: (value: unknown) =>
      Number(value) > 0
        ? dayjs.unix(Number(value)).format('YYYY-MM-DD HH:mm:ss')
        : '-',
    billingDate: (value: unknown) =>
      Number(value) > 0
        ? billingDate.format(new Date(Number(value) * 1000))
        : '-',
  }
}
