import type { WalletAdjustment } from '../api'
import type { User } from '../types'

export type WalletAdjustmentDraft = WalletAdjustment & {
  id: number
  currency: 'CNY' | 'USD'
  amount: string
}

export const walletAdjustmentReasons = {
  paid: [
    ['compensation', 'Compensation'],
    ['refund', 'Refund'],
    ['manual_topup', 'Manual top-up'],
    ['account_closure', 'Account closure'],
  ],
  gift: [
    ['admin_grant', 'Admin grant'],
    ['promo_campaign', 'Promotion'],
    ['compensation', 'Compensation'],
    ['gift_voucher', 'Gift voucher'],
    ['system_adjustment', 'System adjustment'],
  ],
} as const

export function previewWalletAdjustments(
  user: User,
  drafts: WalletAdjustmentDraft[]
) {
  const balances = {
    total: user.quota,
    paid: user.paid_quota ?? 0,
    gift: user.gift_quota ?? 0,
  }
  let valid = Object.values(balances).every(Number.isSafeInteger)
  const entries = drafts.map((draft) => {
    const before = balances[draft.funding_type]
    balances[draft.funding_type] += draft.delta_quota
    balances.total += draft.delta_quota
    const after = balances[draft.funding_type]
    valid &&=
      Number.isSafeInteger(draft.delta_quota) &&
      draft.delta_quota !== 0 &&
      Object.values(balances).every(Number.isSafeInteger) &&
      // Credits may partially repay existing debt. Deductions must leave every
      // wallet non-negative, matching the server's specific-wallet debit guard.
      (draft.delta_quota > 0 ||
        (balances.paid >= 0 && balances.gift >= 0 && balances.total >= 0))
    return { draft, before, after }
  })
  return { balances, entries, valid }
}
