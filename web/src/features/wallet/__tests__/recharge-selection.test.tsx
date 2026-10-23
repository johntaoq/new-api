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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { useState, type ComponentProps } from 'react'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { beforeEach, expect, it, vi } from 'vitest'

import en from '@/i18n/locales/en.json'

import { RechargeFormCard } from '../components/recharge-form-card'

const i18n = createInstance()
const presets = [10, 20, 50, 100, 200, 300, 500].map((value) => ({ value }))

function RechargeFixture(
  props: Partial<ComponentProps<typeof RechargeFormCard>>
) {
  const [currency, setCurrency] = useState<'CNY' | 'USD'>('CNY')
  const [selected, setSelected] = useState<number | null>(null)
  const [amount, setAmount] = useState(10)

  return (
    <I18nextProvider i18n={i18n}>
      <RechargeFormCard
        topupInfo={{
          enable_online_topup: true,
          enable_stripe_topup: false,
          enable_redemption: true,
          pay_methods: [{ name: 'Alipay', type: 'alipay', min_topup: 10 }],
          min_topup: 10,
          stripe_min_topup: 1,
          amount_options: [10, 20, 50, 100, 200, 300, 500],
          discount: {},
        }}
        topupCurrency={currency}
        currencySelectable
        onCurrencyChange={setCurrency}
        presetAmounts={presets}
        selectedPreset={selected}
        onSelectPreset={(preset) => {
          setSelected(preset.value)
          setAmount(preset.value)
        }}
        topupAmount={amount}
        onTopupAmountChange={(value) => {
          setAmount(value)
          setSelected(null)
        }}
        paymentAmount={10}
        calculating={false}
        paymentLoading={null}
        onPaymentMethodSelect={vi.fn()}
        redemptionCode=''
        onRedemptionCodeChange={vi.fn()}
        onRedeem={vi.fn()}
        redeeming={false}
        {...props}
      />
    </I18nextProvider>
  )
}

beforeEach(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    nsSeparator: false,
    resources: { en },
    interpolation: { escapeValue: false },
  })
})

it('switches currency directly using tabs without changing the entered amount', async () => {
  const user = userEvent.setup()
  render(<RechargeFixture />)

  expect(screen.getByRole('tablist', { name: 'Currency' })).toBeVisible()
  expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
  expect(screen.getByRole('tab', { name: 'CNY' })).toHaveAttribute(
    'aria-selected',
    'true'
  )

  await user.click(screen.getByRole('tab', { name: 'USD' }))

  expect(screen.getByRole('tab', { name: 'USD' })).toHaveAttribute(
    'aria-selected',
    'true'
  )
  expect(screen.getByRole('tab', { name: 'CNY' })).toHaveAttribute(
    'aria-selected',
    'false'
  )
  expect(
    screen.getByRole('spinbutton', { name: 'Custom Amount (USD)' })
  ).toHaveValue(10)
  expect(screen.getByRole('button', { name: /^10 USD/ })).toHaveTextContent(
    'Pay 10'
  )
})

it('supports arrow-key currency selection with matching focus and active state', async () => {
  const user = userEvent.setup()
  render(<RechargeFixture />)

  await user.click(screen.getByRole('tab', { name: 'CNY' }))
  await user.keyboard('{ArrowRight}')

  expect(screen.getByRole('tab', { name: 'USD' })).toHaveFocus()
  expect(screen.getByRole('tab', { name: 'USD' })).toHaveAttribute(
    'aria-selected',
    'true'
  )
})

it('keeps the currency fixed when currency selection is not available', () => {
  render(<RechargeFixture currencySelectable={false} />)

  expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: /^10 CNY/ })).toBeVisible()
})

it('selects one denomination by click or keyboard and updates the amount', async () => {
  const user = userEvent.setup()
  render(<RechargeFixture />)

  await user.click(screen.getByRole('button', { name: /^50 CNY/ }))
  expect(screen.getByRole('button', { name: /^50 CNY/ })).toHaveAttribute(
    'aria-pressed',
    'true'
  )
  expect(screen.getByRole('spinbutton')).toHaveValue(50)

  screen.getByRole('button', { name: /^20 CNY/ }).focus()
  await user.keyboard('{Enter}')

  expect(screen.getByRole('button', { name: /^20 CNY/ })).toHaveAttribute(
    'aria-pressed',
    'true'
  )
  expect(screen.getByRole('button', { name: /^50 CNY/ })).toHaveAttribute(
    'aria-pressed',
    'false'
  )
  expect(screen.getByRole('spinbutton')).toHaveValue(20)
})

it('clears the preset selection when a custom amount is entered', async () => {
  const user = userEvent.setup()
  render(<RechargeFixture />)

  await user.click(screen.getByRole('button', { name: /^50 CNY/ }))
  await user.clear(screen.getByRole('spinbutton'))
  await user.type(screen.getByRole('spinbutton'), '35')

  expect(screen.getByRole('spinbutton')).toHaveValue(35)
  expect(
    screen.queryByRole('button', { pressed: true })
  ).not.toBeInTheDocument()
})

it('gives each displayed denomination a different color and allows long amounts to wrap', () => {
  render(
    <RechargeFixture presetAmounts={[...presets, { value: 1234567890 }]} />
  )

  const tiles = presets.map((preset) =>
    screen.getByRole('button', { name: new RegExp(`^${preset.value} CNY`) })
  )
  const colors = tiles.map((tile) =>
    [...tile.classList].find((className) =>
      /^bg-(cyan|emerald|blue|amber|rose|violet|orange)-/.test(className)
    )
  )
  expect(new Set(colors).size).toBe(7)
  expect(colors.every(Boolean)).toBe(true)
  expect(
    screen.getByRole('button', { name: /^1,234,567,890 CNY/ })
  ).toHaveClass('h-auto', 'min-w-0')
  expect(screen.getByText('1,234,567,890')).toHaveClass('break-all')
})

it('preserves the disabled payment action while a payment is loading', () => {
  render(<RechargeFixture paymentLoading='alipay' />)

  expect(screen.getByRole('button', { name: 'Alipay' })).toBeDisabled()
})

it('shows the supplied Alipay banner on an enabled, colored payment button', async () => {
  const onPaymentMethodSelect = vi.fn()
  const user = userEvent.setup()
  render(<RechargeFixture onPaymentMethodSelect={onPaymentMethodSelect} />)

  const button = screen.getByRole('button', { name: 'Alipay' })
  expect(button).toBeEnabled()
  expect(screen.getByRole('img', { name: 'Alipay' })).toHaveAttribute(
    'src',
    expect.stringContaining('alipay-payment.png')
  )
  expect(button).toHaveClass('bg-[#487aef]')
  await user.click(button)
  expect(onPaymentMethodSelect).toHaveBeenCalledWith({
    name: 'Alipay',
    type: 'alipay',
    min_topup: 10,
  })
})

it('keeps a below-minimum payment disabled with a visible minimum amount', () => {
  render(<RechargeFixture topupAmount={5} />)

  const button = screen.getByRole('button', {
    name: 'Alipay. Minimum topup amount: 10',
  })
  expect(button).toBeDisabled()
  expect(button).toHaveTextContent('Minimum: 10')
  expect(button).toHaveClass('disabled:bg-muted')
})
