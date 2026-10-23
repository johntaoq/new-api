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
import type { TFunction } from 'i18next'
import { z } from 'zod'

import { useSystemConfigStore } from '@/stores/system-config-store'

import {
  REDEMPTION_VALIDATION,
  getRedemptionFormErrorMessages,
} from '../constants'
import type { RedemptionFormData, Redemption } from '../types'

// ============================================================================
// Form Schema (use getRedemptionFormSchema(t) in components for i18n messages)
// ============================================================================

export function getRedemptionFormSchema(t: TFunction) {
  const msg = getRedemptionFormErrorMessages(t)
  return z
    .object({
      name: z
        .string()
        .min(REDEMPTION_VALIDATION.NAME_MIN_LENGTH, msg.NAME_LENGTH_INVALID)
        .max(REDEMPTION_VALIDATION.NAME_MAX_LENGTH, msg.NAME_LENGTH_INVALID),
      quota_dollars: z.number().positive(t('Quota must be a positive number')),
      currency: z.enum(['USD', 'CNY']),
      funding_type: z.enum(['paid', 'gift']),
      recognized_revenue_usd: z.number().min(0, t('Invalid financial amount')),
      remark: z.string().max(255),
      expired_time: z.date().optional(),
      count: z
        .number()
        .min(REDEMPTION_VALIDATION.COUNT_MIN, msg.COUNT_INVALID)
        .max(REDEMPTION_VALIDATION.COUNT_MAX, msg.COUNT_INVALID)
        .optional(),
    })
    .refine(
      (value) =>
        value.funding_type !== 'gift' || value.remark.trim().length > 0,
      {
        path: ['remark'],
        message: t('Remark is required for gift vouchers'),
      }
    )
}

export type RedemptionFormValues = {
  name: string
  quota_dollars: number
  currency: 'USD' | 'CNY'
  funding_type: 'paid' | 'gift'
  recognized_revenue_usd: number
  remark: string
  expired_time?: Date
  count?: number
}

// ============================================================================
// Form Defaults
// ============================================================================

export const REDEMPTION_FORM_DEFAULT_VALUES: RedemptionFormValues = {
  name: '',
  quota_dollars: 10,
  currency: 'USD',
  funding_type: 'gift',
  recognized_revenue_usd: 0,
  remark: '',
  expired_time: undefined,
  count: 1,
}

// ============================================================================
// Form Data Transformation
// ============================================================================

/**
 * Transform form data to API payload
 */
export function transformFormDataToPayload(
  data: RedemptionFormValues
): RedemptionFormData {
  const config = useSystemConfigStore.getState().config.currency
  const usd =
    data.currency === 'CNY'
      ? data.quota_dollars / config.usdExchangeRate
      : data.quota_dollars
  return {
    name: data.name,
    quota: Math.round(usd * config.quotaPerUnit),
    currency: data.currency,
    input_amount: data.quota_dollars,
    funding_type: data.funding_type,
    recognized_revenue_usd:
      data.funding_type === 'paid' ? data.recognized_revenue_usd : 0,
    remark: data.remark,
    expired_time: data.expired_time
      ? Math.floor(data.expired_time.getTime() / 1000)
      : 0,
    count: data.count || 1,
  }
}

/**
 * Transform redemption data to form defaults
 */
export function transformRedemptionToFormDefaults(
  redemption: Redemption
): RedemptionFormValues {
  const config = useSystemConfigStore.getState().config.currency
  const currency =
    redemption.currency || (config.quotaDisplayType === 'CNY' ? 'CNY' : 'USD')
  const rate = currency === 'CNY' ? config.usdExchangeRate : 1
  return {
    name: redemption.name,
    quota_dollars:
      redemption.input_amount ||
      Number(((redemption.quota / config.quotaPerUnit) * rate).toFixed(4)),
    currency,
    funding_type: redemption.funding_type ?? 'gift',
    recognized_revenue_usd: redemption.recognized_revenue_usd ?? 0,
    remark: redemption.remark ?? '',
    expired_time:
      redemption.expired_time > 0
        ? new Date(redemption.expired_time * 1000)
        : undefined,
    count: 1,
  }
}
