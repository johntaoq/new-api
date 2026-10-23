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
import {
  Check,
  Gift,
  ExternalLink,
  Loader2,
  Receipt,
  WalletCards,
} from 'lucide-react'
import { useState, useEffect, type ReactNode } from 'react'
import { Trans, useTranslation } from 'react-i18next'

import alipayPaymentImage from '@/assets/payments/alipay-payment.png'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { TitledCard } from '@/components/ui/titled-card'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  formatCurrency,
  getDiscountLabel,
  getPaymentIcon,
  getMinTopupAmount,
  calculatePresetPricing,
} from '../lib'
import type {
  PaymentMethod,
  PresetAmount,
  TopupInfo,
  CreemProduct,
  WaffoPayMethod,
} from '../types'
import { CreemProductsSection } from './creem-products-section'

const PRESET_TONES = [
  'border-cyan-300 border-l-cyan-500 bg-cyan-50 text-cyan-950 hover:border-cyan-500 hover:bg-cyan-100 hover:text-cyan-950 aria-pressed:border-cyan-600 aria-pressed:bg-cyan-100 dark:border-cyan-800 dark:border-l-cyan-400 dark:bg-cyan-950/60 dark:text-cyan-100 dark:hover:border-cyan-400 dark:hover:bg-cyan-900/70 dark:hover:text-cyan-100 dark:aria-pressed:border-cyan-400 dark:aria-pressed:bg-cyan-900/80',
  'border-emerald-300 border-l-emerald-500 bg-emerald-50 text-emerald-950 hover:border-emerald-500 hover:bg-emerald-100 hover:text-emerald-950 aria-pressed:border-emerald-600 aria-pressed:bg-emerald-100 dark:border-emerald-800 dark:border-l-emerald-400 dark:bg-emerald-950/60 dark:text-emerald-100 dark:hover:border-emerald-400 dark:hover:bg-emerald-900/70 dark:hover:text-emerald-100 dark:aria-pressed:border-emerald-400 dark:aria-pressed:bg-emerald-900/80',
  'border-blue-300 border-l-blue-500 bg-blue-50 text-blue-950 hover:border-blue-500 hover:bg-blue-100 hover:text-blue-950 aria-pressed:border-blue-600 aria-pressed:bg-blue-100 dark:border-blue-800 dark:border-l-blue-400 dark:bg-blue-950/60 dark:text-blue-100 dark:hover:border-blue-400 dark:hover:bg-blue-900/70 dark:hover:text-blue-100 dark:aria-pressed:border-blue-400 dark:aria-pressed:bg-blue-900/80',
  'border-amber-300 border-l-amber-500 bg-amber-50 text-amber-950 hover:border-amber-500 hover:bg-amber-100 hover:text-amber-950 aria-pressed:border-amber-600 aria-pressed:bg-amber-100 dark:border-amber-800 dark:border-l-amber-400 dark:bg-amber-950/60 dark:text-amber-100 dark:hover:border-amber-400 dark:hover:bg-amber-900/70 dark:hover:text-amber-100 dark:aria-pressed:border-amber-400 dark:aria-pressed:bg-amber-900/80',
  'border-rose-300 border-l-rose-500 bg-rose-50 text-rose-950 hover:border-rose-500 hover:bg-rose-100 hover:text-rose-950 aria-pressed:border-rose-600 aria-pressed:bg-rose-100 dark:border-rose-800 dark:border-l-rose-400 dark:bg-rose-950/60 dark:text-rose-100 dark:hover:border-rose-400 dark:hover:bg-rose-900/70 dark:hover:text-rose-100 dark:aria-pressed:border-rose-400 dark:aria-pressed:bg-rose-900/80',
  'border-violet-300 border-l-violet-500 bg-violet-50 text-violet-950 hover:border-violet-500 hover:bg-violet-100 hover:text-violet-950 aria-pressed:border-violet-600 aria-pressed:bg-violet-100 dark:border-violet-800 dark:border-l-violet-400 dark:bg-violet-950/60 dark:text-violet-100 dark:hover:border-violet-400 dark:hover:bg-violet-900/70 dark:hover:text-violet-100 dark:aria-pressed:border-violet-400 dark:aria-pressed:bg-violet-900/80',
  'border-orange-300 border-l-orange-500 bg-orange-50 text-orange-950 hover:border-orange-500 hover:bg-orange-100 hover:text-orange-950 aria-pressed:border-orange-600 aria-pressed:bg-orange-100 dark:border-orange-800 dark:border-l-orange-400 dark:bg-orange-950/60 dark:text-orange-100 dark:hover:border-orange-400 dark:hover:bg-orange-900/70 dark:hover:text-orange-100 dark:aria-pressed:border-orange-400 dark:aria-pressed:bg-orange-900/80',
] as const

interface RechargeFormCardProps {
  topupCurrency?: 'CNY' | 'USD'
  onCurrencyChange?: (currency: 'CNY' | 'USD') => void
  currencySelectable?: boolean
  minimumAmount?: number
  topupInfo: TopupInfo | null
  presetAmounts: PresetAmount[]
  selectedPreset: number | null
  onSelectPreset: (preset: PresetAmount) => void
  topupAmount: number
  onTopupAmountChange: (amount: number) => void
  paymentAmount: number
  calculating: boolean
  onPaymentMethodSelect: (method: PaymentMethod) => void
  paymentLoading: string | null
  redemptionCode: string
  onRedemptionCodeChange: (code: string) => void
  onRedeem: () => void
  redeeming: boolean
  topupLink?: string
  loading?: boolean
  priceRatio?: number
  usdExchangeRate?: number
  onOpenBilling?: () => void
  creemProducts?: CreemProduct[]
  enableCreemTopup?: boolean
  onCreemProductSelect?: (product: CreemProduct) => void
  enableWaffoTopup?: boolean
  waffoPayMethods?: WaffoPayMethod[]
  waffoMinTopup?: number
  onWaffoMethodSelect?: (method: WaffoPayMethod, index: number) => void
  enableWaffoPancakeTopup?: boolean
}

export function RechargeFormCard({
  topupCurrency = 'USD',
  onCurrencyChange,
  currencySelectable,
  minimumAmount,
  topupInfo,
  presetAmounts,
  selectedPreset,
  onSelectPreset,
  topupAmount,
  onTopupAmountChange,
  paymentAmount,
  calculating,
  onPaymentMethodSelect,
  paymentLoading,
  redemptionCode,
  onRedemptionCodeChange,
  onRedeem,
  redeeming,
  topupLink,
  loading,
  priceRatio = 1,
  usdExchangeRate = 1,
  onOpenBilling,
  creemProducts,
  enableCreemTopup,
  onCreemProductSelect,
  enableWaffoTopup,
  waffoPayMethods,
  waffoMinTopup,
  onWaffoMethodSelect,
  enableWaffoPancakeTopup,
}: RechargeFormCardProps) {
  const { t } = useTranslation()
  const [localAmount, setLocalAmount] = useState(topupAmount.toString())

  useEffect(() => {
    // Empty string must survive, otherwise the field can never be cleared
    setLocalAmount((prev) =>
      prev === '' && topupAmount === 0 ? prev : topupAmount.toString()
    )
  }, [topupAmount])

  const handleAmountChange = (value: string) => {
    setLocalAmount(value)
    const numValue = Number.parseInt(value) || 0
    if (numValue >= 0) {
      onTopupAmountChange(numValue)
    }
  }

  const hasConfigurableTopup =
    topupInfo?.enable_online_topup ||
    topupInfo?.enable_stripe_topup ||
    enableWaffoTopup ||
    enableWaffoPancakeTopup
  const hasAnyTopup = hasConfigurableTopup || enableCreemTopup
  const hasStandardPaymentMethods =
    Array.isArray(topupInfo?.pay_methods) && topupInfo.pay_methods.length > 0
  const hasWaffoPaymentMethods =
    Array.isArray(waffoPayMethods) && waffoPayMethods.length > 0
  const minTopup = minimumAmount ?? getMinTopupAmount(topupInfo)
  const redemptionEnabled = topupInfo?.enable_redemption !== false

  if (loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
          <Skeleton className='h-6 w-32' />
          <Skeleton className='mt-2 h-4 w-48' />
        </CardHeader>
        <CardContent className='space-y-4 p-3 sm:space-y-6 sm:p-5'>
          <div className='space-y-4 sm:space-y-6'>
            {/* Preset Amounts Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-16' />
              <div className='grid grid-cols-2 gap-3 sm:grid-cols-4'>
                {Array.from({ length: 8 }, (_, index) => `preset-${index}`).map(
                  (key) => (
                    <Skeleton key={key} className='h-[72px] rounded-lg' />
                  )
                )}
              </div>
            </div>

            {/* Custom Amount Input Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-28' />
              <Skeleton className='h-[42px] w-full' />
            </div>

            {/* Payment Methods Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-32' />
              <div className='flex flex-wrap gap-3'>
                {['primary', 'secondary', 'tertiary'].map((key) => (
                  <Skeleton key={key} className='h-10 w-24 rounded-lg' />
                ))}
              </div>
            </div>
          </div>

          {/* Redemption Code Section Skeleton */}
          <div className='space-y-3 border-t pt-8'>
            <Skeleton className='h-3 w-24' />
            <div className='flex gap-2'>
              <Skeleton className='h-10 flex-1' />
              <Skeleton className='h-10 w-20' />
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <TitledCard
      title={t('Add Funds')}
      description={t('Choose an amount and payment method')}
      icon={<WalletCards className='h-4 w-4' />}
      iconTone='success'
      disableHoverEffect
      action={
        onOpenBilling ? (
          <Button
            variant='outline'
            size='sm'
            onClick={onOpenBilling}
            className='w-full gap-2 sm:w-auto'
          >
            <Receipt className='h-4 w-4' />
            {t('Order History')}
          </Button>
        ) : null
      }
      contentClassName='space-y-4 sm:space-y-6'
    >
      {/* Online Topup Section */}
      {hasAnyTopup ? (
        <div className='space-y-4 sm:space-y-6'>
          {hasConfigurableTopup && (
            <>
              {currencySelectable && (
                <div className='flex items-center gap-3'>
                  <Label id='topup-currency-label'>{t('Currency')}</Label>
                  <Tabs
                    value={topupCurrency}
                    onValueChange={(value) => {
                      if (value === 'CNY' || value === 'USD') {
                        onCurrencyChange?.(value)
                      }
                    }}
                  >
                    <TabsList
                      activateOnFocus
                      aria-labelledby='topup-currency-label'
                      className='bg-muted/50 w-52 max-w-full border p-1 group-data-horizontal/tabs:h-11'
                    >
                      <TabsTrigger
                        value='CNY'
                        className='data-active:bg-foreground data-active:text-background dark:data-active:bg-foreground dark:data-active:text-background px-5 font-semibold dark:data-active:border-transparent'
                      >
                        CNY
                      </TabsTrigger>
                      <TabsTrigger
                        value='USD'
                        className='data-active:bg-foreground data-active:text-background dark:data-active:bg-foreground dark:data-active:text-background px-5 font-semibold dark:data-active:border-transparent'
                      >
                        USD
                      </TabsTrigger>
                    </TabsList>
                  </Tabs>
                </div>
              )}
              {presetAmounts.length > 0 && (
                <div className='space-y-2.5 sm:space-y-3'>
                  <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                    {t('Amount')}
                  </Label>
                  <div className='grid auto-rows-fr grid-cols-2 gap-2 sm:gap-3 md:grid-cols-4'>
                    {presetAmounts.map((preset, index) => {
                      const discount =
                        preset.discount ||
                        topupInfo?.discount?.[preset.value] ||
                        1.0
                      const {
                        displayValue,
                        actualPrice,
                        savedAmount,
                        hasDiscount,
                      } = calculatePresetPricing(
                        preset.value,
                        priceRatio,
                        discount,
                        usdExchangeRate
                      )
                      return (
                        <Button
                          key={preset.value}
                          variant='outline'
                          aria-pressed={selectedPreset === preset.value}
                          className={cn(
                            'relative flex h-auto min-h-28 min-w-0 flex-col items-start justify-between rounded-lg border-l-4 px-3 py-3 text-left whitespace-normal aria-pressed:shadow-sm aria-pressed:ring-1 aria-pressed:ring-current sm:p-4',
                            PRESET_TONES[index % PRESET_TONES.length]
                          )}
                          onClick={() => onSelectPreset(preset)}
                        >
                          <Check
                            aria-hidden='true'
                            className={cn(
                              'absolute top-3 right-3 size-4',
                              selectedPreset !== preset.value && 'opacity-0'
                            )}
                          />
                          <div className='flex w-full flex-wrap items-baseline gap-x-2 gap-y-1 pr-5'>
                            <div className='min-w-0 text-2xl leading-8 font-bold break-all tabular-nums sm:text-3xl sm:leading-9'>
                              {formatNumber(displayValue)}{' '}
                              <span className='text-xs font-semibold sm:text-sm'>
                                {topupCurrency}
                              </span>
                            </div>
                            {hasDiscount && (
                              <div className='text-xs font-semibold text-emerald-800 dark:text-emerald-200'>
                                {getDiscountLabel(discount, t)}
                              </div>
                            )}
                          </div>
                          <div className='mt-2 w-full text-xs leading-5 font-medium sm:text-sm'>
                            {hasDiscount && savedAmount > 0 ? (
                              <Trans
                                t={t}
                                i18nKey='Pay {{amount}} <savings>• Save {{saved}}</savings>'
                                values={{
                                  amount: formatCurrency(actualPrice),
                                  saved: formatCurrency(savedAmount),
                                }}
                                components={{
                                  savings: (
                                    <span className='text-emerald-800 dark:text-emerald-200' />
                                  ),
                                }}
                              />
                            ) : (
                              t('Pay {{amount}}', {
                                amount: formatCurrency(actualPrice),
                              })
                            )}
                          </div>
                        </Button>
                      )
                    })}
                  </div>
                </div>
              )}

              <div className='space-y-2.5 sm:space-y-3'>
                <Label
                  htmlFor='topup-amount'
                  className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
                >
                  {t('Custom Amount')} ({topupCurrency})
                </Label>
                <div className='grid grid-cols-[minmax(0,1fr)_minmax(110px,0.55fr)] gap-2 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center'>
                  <Input
                    id='topup-amount'
                    type='number'
                    value={localAmount}
                    onChange={(e) => handleAmountChange(e.target.value)}
                    min={minTopup}
                    placeholder={t('Minimum {{amount}}', { amount: minTopup })}
                    className='h-9 text-base sm:h-10 sm:text-lg'
                  />
                  <div className='bg-muted/30 flex min-h-9 items-center justify-between gap-2 rounded-md border px-3 lg:min-w-52'>
                    <span className='text-muted-foreground truncate text-xs'>
                      {t('Amount to pay:')}
                    </span>
                    {calculating ? (
                      <Skeleton className='h-5 w-16' />
                    ) : (
                      <span className='text-sm font-semibold'>
                        {formatCurrency(paymentAmount)}
                      </span>
                    )}
                  </div>
                </div>
              </div>

              <div className='space-y-2.5 sm:space-y-3'>
                <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                  {t('Payment Method')}
                </Label>
                {hasStandardPaymentMethods ? (
                  <div className='grid grid-cols-2 gap-1.5 sm:gap-3 lg:grid-cols-3'>
                    {topupInfo?.pay_methods?.map((method) => {
                      const minTopup = Math.max(
                        method.min_topup || 0,
                        minimumAmount ?? getMinTopupAmount(topupInfo)
                      )
                      const disabled = minTopup > topupAmount
                      const disabledReason = disabled
                        ? t('Minimum topup amount: {{amount}}', {
                            amount: minTopup,
                          })
                        : undefined
                      const disabledLabel = disabled
                        ? `${t('Minimum:')} ${minTopup}`
                        : undefined
                      const isAlipay = method.type === 'alipay'
                      let paymentIcon: ReactNode
                      if (paymentLoading === method.type) {
                        paymentIcon = (
                          <Loader2
                            aria-hidden='true'
                            className='size-5 animate-spin'
                          />
                        )
                      } else if (isAlipay) {
                        paymentIcon = (
                          <img
                            src={alipayPaymentImage}
                            alt={method.name}
                            width={826}
                            height={368}
                            className={cn(
                              'h-12 w-auto max-w-full shrink-0 object-contain',
                              disabled && 'mb-4 grayscale'
                            )}
                          />
                        )
                      } else {
                        paymentIcon = getPaymentIcon(
                          method.type,
                          'h-5 w-5',
                          method.icon,
                          method.name
                        )
                      }

                      const button = (
                        <Button
                          key={method.type}
                          variant='outline'
                          onClick={() => onPaymentMethodSelect(method)}
                          disabled={disabled || !!paymentLoading}
                          title={disabledReason}
                          aria-label={
                            disabledReason
                              ? `${method.name}. ${disabledReason}`
                              : method.name
                          }
                          className={cn(
                            'relative h-20 min-w-0 justify-center gap-3 rounded-lg border-blue-600 bg-blue-600 px-3 py-2 text-left font-semibold text-white shadow-sm hover:border-blue-700 hover:bg-blue-700 hover:text-white focus-visible:ring-blue-400/50 disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-70 disabled:shadow-none dark:border-blue-500 dark:bg-blue-600 dark:hover:border-blue-400 dark:hover:bg-blue-500 dark:disabled:border-border dark:disabled:bg-muted',
                            isAlipay &&
                              'border-[#487aef] bg-[#487aef] hover:border-[#3565d8] hover:bg-[#3565d8] dark:border-[#487aef] dark:bg-[#487aef] dark:hover:border-[#3565d8] dark:hover:bg-[#3565d8]'
                          )}
                        >
                          {paymentIcon}
                          <span
                            className={cn(
                              'flex min-w-0 flex-col items-start gap-0.5',
                              isAlipay &&
                                paymentLoading !== method.type &&
                                'absolute inset-x-2 bottom-1 items-center'
                            )}
                          >
                            <span
                              className={cn(
                                'max-w-full truncate',
                                isAlipay &&
                                  paymentLoading !== method.type &&
                                  'sr-only'
                              )}
                            >
                              {method.name}
                            </span>
                            {disabledLabel && (
                              <span className='text-muted-foreground max-w-full truncate text-[11px] leading-4 font-normal'>
                                {disabledLabel}
                              </span>
                            )}
                          </span>
                        </Button>
                      )

                      return disabled ? (
                        <TooltipProvider key={method.type}>
                          <Tooltip>
                            <TooltipTrigger render={button} />
                            <TooltipContent>{disabledReason}</TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      ) : (
                        button
                      )
                    })}
                  </div>
                ) : null}
                {!hasStandardPaymentMethods && !hasWaffoPaymentMethods && (
                  <Alert>
                    <AlertDescription>
                      {t(
                        'No payment methods available. Please contact administrator.'
                      )}
                    </AlertDescription>
                  </Alert>
                )}
              </div>

              {enableWaffoTopup &&
                hasWaffoPaymentMethods &&
                onWaffoMethodSelect && (
                  <div className='space-y-2.5 sm:space-y-3'>
                    <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                      {t('Waffo Payment')}
                    </Label>
                    <div className='grid grid-cols-2 gap-1.5 sm:gap-3 lg:grid-cols-3'>
                      {waffoPayMethods?.map((method, index) => {
                        const loadingKey = `waffo-${index}`
                        const methodKey = `${method.payMethodType ?? 'unknown'}-${method.payMethodName ?? method.name}`
                        const waffoMin = waffoMinTopup || 0
                        const belowMin = waffoMin > topupAmount
                        const disabledReason = belowMin
                          ? t('Minimum topup amount: {{amount}}', {
                              amount: waffoMin,
                            })
                          : undefined
                        const disabledLabel = belowMin
                          ? `${t('Minimum:')} ${waffoMin}`
                          : undefined

                        let methodIcon = getPaymentIcon('waffo')
                        if (paymentLoading === loadingKey) {
                          methodIcon = (
                            <Loader2 className='h-4 w-4 animate-spin' />
                          )
                        } else if (method.icon) {
                          methodIcon = (
                            <img
                              src={method.icon}
                              alt={method.name}
                              className='h-4 w-4 object-contain'
                            />
                          )
                        }

                        const button = (
                          <Button
                            key={methodKey}
                            variant='outline'
                            onClick={() => onWaffoMethodSelect(method, index)}
                            disabled={belowMin || !!paymentLoading}
                            title={disabledReason}
                            aria-label={
                              disabledReason
                                ? `${method.name}. ${disabledReason}`
                                : method.name
                            }
                            className='disabled:border-border disabled:bg-muted disabled:text-muted-foreground dark:disabled:border-border dark:disabled:bg-muted h-20 min-w-0 justify-center gap-3 rounded-lg border-blue-600 bg-blue-600 px-3 py-2 text-left font-semibold text-white shadow-sm hover:border-blue-700 hover:bg-blue-700 hover:text-white focus-visible:ring-blue-400/50 disabled:opacity-70 disabled:shadow-none dark:border-blue-500 dark:bg-blue-600 dark:hover:border-blue-400 dark:hover:bg-blue-500'
                          >
                            {methodIcon}
                            <span className='flex min-w-0 flex-col items-start gap-0.5'>
                              <span className='max-w-full truncate'>
                                {method.name}
                              </span>
                              {disabledLabel && (
                                <span className='text-muted-foreground max-w-full truncate text-[11px] leading-4 font-normal'>
                                  {disabledLabel}
                                </span>
                              )}
                            </span>
                          </Button>
                        )

                        return belowMin ? (
                          <TooltipProvider key={methodKey}>
                            <Tooltip>
                              <TooltipTrigger render={button} />
                              <TooltipContent>{disabledReason}</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        ) : (
                          button
                        )
                      })}
                    </div>
                  </div>
                )}
            </>
          )}
        </div>
      ) : (
        <Alert>
          <AlertDescription>
            {t(
              'Online topup is not enabled. Please use redemption code or contact administrator.'
            )}
          </AlertDescription>
        </Alert>
      )}

      {/* Creem Products Section */}
      {enableCreemTopup &&
        Array.isArray(creemProducts) &&
        creemProducts.length > 0 &&
        onCreemProductSelect && (
          <div className='space-y-2.5 border-t pt-4 sm:space-y-3 sm:pt-6'>
            <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
              {t('Creem Payment')}
            </Label>
            <CreemProductsSection
              products={creemProducts}
              onProductSelect={onCreemProductSelect}
            />
          </div>
        )}

      {/* Redemption Code Section */}
      {redemptionEnabled ? (
        <div className='space-y-2.5 border-t pt-4 sm:space-y-3 sm:pt-6'>
          <div className='flex items-center gap-2'>
            <IconBadge tone='warning' size='xs'>
              <Gift />
            </IconBadge>
            <Label
              htmlFor='redemption-code'
              className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
            >
              {t('Have a Code?')}
            </Label>
          </div>
          <div className='grid grid-cols-[minmax(0,1fr)_auto] gap-2'>
            <Input
              id='redemption-code'
              value={redemptionCode}
              onChange={(e) => onRedemptionCodeChange(e.target.value)}
              placeholder={t('Enter your redemption code')}
              className='h-9 min-w-0'
            />
            <Button
              onClick={onRedeem}
              disabled={redeeming}
              variant='outline'
              className='h-9 px-4'
            >
              {redeeming && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
              {t('Redeem')}
            </Button>
          </div>
          {topupLink && (
            <p className='text-muted-foreground text-xs'>
              {t('Need a redemption code?')}{' '}
              <a
                href={topupLink}
                target='_blank'
                rel='noopener noreferrer'
                className='inline-flex items-center gap-1 underline-offset-4 hover:underline'
              >
                {t('Get one here')}
                <ExternalLink className='h-3 w-3' />
              </a>
            </p>
          )}
        </div>
      ) : (
        <Alert className='border-t'>
          <AlertDescription>
            {t(
              'Redemption codes are disabled until the administrator confirms compliance terms.'
            )}
          </AlertDescription>
        </Alert>
      )}
    </TitledCard>
  )
}
