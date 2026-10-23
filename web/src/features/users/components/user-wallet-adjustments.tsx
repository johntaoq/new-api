import { Pencil, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import {
  previewWalletAdjustments,
  walletAdjustmentReasons,
  type WalletAdjustmentDraft,
} from '../lib/wallet-adjustments'
import type { User } from '../types'
import { UserQuotaDialog } from './user-quota-dialog'

export function UserWalletAdjustments(props: {
  user: User
  pending: WalletAdjustmentDraft[]
  onChange?: (drafts: WalletAdjustmentDraft[]) => void
  disabled?: boolean
}) {
  const { t, i18n } = useTranslation()
  const root = useAuthStore((s) => s.auth.user?.role === ROLE.SUPER_ADMIN)
  const config = useSystemConfigStore((s) => s.config.currency)
  const [editor, setEditor] = useState<{
    fundingType: 'gift' | 'paid'
    draft?: WalletAdjustmentDraft
  } | null>(null)
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const money = (quota: number) =>
    formatQuotaWithCurrency(quota, {
      digitsLarge: 6,
      digitsSmall: 6,
      abbreviate: false,
      locale,
    })
  const usd = (value: number) =>
    new Intl.NumberFormat(locale, {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 6,
    }).format(value)
  const preview = previewWalletAdjustments(props.user, props.pending)
  const summaries = [
    {
      label: t('Total balance'),
      current: props.user.quota,
      after: preview.balances.total,
    },
    {
      label: t('Paid wallet'),
      current: props.user.paid_quota ?? 0,
      after: preview.balances.paid,
    },
    {
      label: t('Gift wallet'),
      current: props.user.gift_quota ?? 0,
      after: preview.balances.gift,
    },
  ]
  const updateDrafts = (draft: WalletAdjustmentDraft) => {
    if (editor?.draft) {
      return props.pending.map((item) =>
        item.id === editor.draft?.id ? draft : item
      )
    }
    return [...props.pending, draft]
  }
  const otherDrafts = props.pending.filter(
    (item) => item.id !== editor?.draft?.id
  )
  const editorBalances = previewWalletAdjustments(
    props.user,
    otherDrafts
  ).balances
  return (
    <div className='space-y-4'>
      <dl
        className='grid gap-4 border-y py-4 sm:grid-cols-3'
        aria-live='polite'
      >
        {summaries.map((item) => (
          <div key={item.label} className='min-w-0'>
            <dt className='text-muted-foreground text-xs'>{item.label}</dt>
            <dd
              aria-label={item.label}
              className='mt-1 text-lg font-semibold break-words'
            >
              {money(item.after)}
            </dd>
            <dd className='text-muted-foreground mt-1 text-xs break-words'>
              {t('Current balance')}: {money(item.current)}
            </dd>
            <dd className='text-muted-foreground mt-1 text-xs break-words'>
              {t('USD equivalent')}: {usd(item.after / config.quotaPerUnit)}
            </dd>
          </div>
        ))}
      </dl>
      {props.onChange && (
        <div className='flex flex-wrap gap-2'>
          <Button
            type='button'
            variant='outline'
            disabled={props.disabled}
            onClick={() => setEditor({ fundingType: 'gift' })}
          >
            <Pencil />
            {t('Adjust gift balance')}
          </Button>
          {root && (
            <Button
              type='button'
              variant='outline'
              disabled={props.disabled}
              onClick={() => setEditor({ fundingType: 'paid' })}
            >
              <Pencil />
              {t('Adjust paid balance')}
            </Button>
          )}
        </div>
      )}
      {props.pending.length > 0 && (
        <section aria-label={t('Pending adjustments')} className='space-y-2'>
          <h3 className='text-sm font-medium'>
            <span>{t('Pending adjustments')}</span> ({props.pending.length})
          </h3>
          {preview.entries.map(({ draft, before, after }) => (
            <div
              key={draft.id}
              className='flex items-start gap-2 border-b py-3 text-sm'
            >
              <div className='min-w-0 flex-1 space-y-1 break-words'>
                <div className='font-medium'>
                  {t(
                    draft.funding_type === 'paid'
                      ? 'Paid wallet'
                      : 'Gift wallet'
                  )}
                  {' · '}
                  <span>
                    {t(
                      walletAdjustmentReasons[draft.funding_type].find(
                        ([value]) => value === draft.source_type
                      )?.[1] ?? draft.source_type
                    )}
                  </span>
                </div>
                <p>
                  {t('Adjustment amount')}: {draft.amount} {draft.currency} (
                  {usd(draft.delta_quota / config.quotaPerUnit)})
                </p>
                <p>
                  {t('Balance before adjustment')}: {money(before)}
                </p>
                <p>
                  {t('Balance after adjustment')}: {money(after)}
                </p>
                {draft.funding_type === 'paid' && (
                  <p>
                    {t('Recognized revenue')}: {usd(draft.revenue_usd)}
                  </p>
                )}
                {draft.remark && (
                  <p>
                    {t('Remark')}: {draft.remark}
                  </p>
                )}
              </div>
              {props.onChange && (
                <div className='flex shrink-0'>
                  <Button
                    type='button'
                    size='icon-sm'
                    variant='ghost'
                    aria-label={t('Edit')}
                    title={t('Edit')}
                    disabled={props.disabled}
                    onClick={() =>
                      setEditor({ fundingType: draft.funding_type, draft })
                    }
                  >
                    <Pencil />
                  </Button>
                  <Button
                    type='button'
                    size='icon-sm'
                    variant='ghost'
                    aria-label={t('Delete')}
                    title={t('Delete')}
                    disabled={props.disabled}
                    onClick={() =>
                      props.onChange?.(
                        props.pending.filter((item) => item.id !== draft.id)
                      )
                    }
                  >
                    <Trash2 />
                  </Button>
                </div>
              )}
            </div>
          ))}
          {!preview.valid && (
            <p role='alert' className='text-destructive text-sm'>
              {t('Check the amount and pending wallet balances')}
            </p>
          )}
        </section>
      )}
      {editor && (
        <UserQuotaDialog
          key={editor.draft?.id ?? editor.fundingType}
          fundingType={editor.fundingType}
          draft={editor.draft}
          username={props.user.username}
          currentQuota={editorBalances[editor.fundingType]}
          onClose={() => setEditor(null)}
          canSave={(draft) =>
            previewWalletAdjustments(props.user, updateDrafts(draft)).valid
          }
          onSave={(draft) => {
            props.onChange?.(
              updateDrafts({
                ...draft,
                id:
                  editor.draft?.id ??
                  Math.max(0, ...props.pending.map((item) => item.id)) + 1,
              })
            )
            setEditor(null)
          }}
        />
      )}
    </div>
  )
}
