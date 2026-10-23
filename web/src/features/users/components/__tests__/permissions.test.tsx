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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { toast } from 'sonner'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import {
  previewWalletAdjustments,
  type WalletAdjustmentDraft,
} from '../../lib/wallet-adjustments'
import type { User } from '../../types'
import { UserPasswordDialog } from '../user-password-dialog'
import { UsersMutateDrawer } from '../users-mutate-drawer'
import { UsersProvider } from '../users-provider'

const target: User = {
  id: 2,
  username: 'managed-admin',
  display_name: 'Managed admin',
  role: 10,
  status: 1,
  quota: 0,
  used_quota: 0,
  request_count: 0,
  group: 'default',
}
const label = "View other accounts' audit logs"
const description =
  'View audit records from user and admin roles. Root records are always excluded.'

function renderPermissions(
  viewerRole: number,
  allowed?: boolean,
  wallet: Partial<User> = {
    quota: 1500000,
    paid_quota: 1000000,
    gift_quota: 500000,
  }
) {
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...useSystemConfigStore.getState().config.currency,
      quotaDisplayType: 'USD',
      usdExchangeRate: 7,
      quotaPerUnit: 500000,
    },
  })
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'operator', role: viewerRole })
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/authz/catalog') {
      return {
        data: {
          success: true,
          data: {
            resources: [
              {
                resource: 'audit',
                label_key: 'Audit Logs',
                actions: [
                  {
                    action: 'read',
                    label_key: label,
                    description_key: description,
                  },
                ],
              },
            ],
            roles: [{ key: 'admin', grants: { audit: { read: false } } }],
          },
        },
      }
    }
    if (url === '/api/group/') {
      return { data: { success: true, data: ['default'] } }
    }
    if (url === '/api/verify/methods') {
      return {
        data: {
          success: true,
          data: {
            scope: 'admin.user.update',
            methods: [{ method: '2fa', available: true }],
            oauth_providers: [],
            password_encryption_enabled: false,
          },
        },
      }
    }
    return {
      data: {
        success: true,
        data: {
          ...target,
          ...wallet,
          admin_permissions:
            allowed === undefined ? {} : { audit: { read: allowed } },
        },
      },
    }
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <UsersProvider>
        <UsersMutateDrawer
          open
          onOpenChange={() => undefined}
          currentRow={target}
        />
      </UsersProvider>
    </QueryClientProvider>
  )
  return get
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  useAuthStore.getState().auth.reset()
})

it.each([undefined, true])(
  'root can save an audit grant or revocation after step-up verification (previous=%s)',
  async (allowed) => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    vi.spyOn(api, 'post').mockResolvedValue({
      data: {
        success: true,
        data: {
          proof_token: 'update-proof',
          method: '2fa',
          scope: 'admin.user.update',
          expires_at: Math.floor(Date.now() / 1000) + 60,
        },
      },
    })
    renderPermissions(100, allowed)
    await screen.findByDisplayValue('Managed admin')
    const toggle = await screen.findByRole('button', { name: label })
    await waitFor(() =>
      expect(toggle).toHaveAttribute('aria-pressed', String(!!allowed))
    )
    expect(toggle).toHaveAccessibleDescription(description)
    await userEvent.click(toggle)
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    // The permission matrix changed, so the save waits for verification.
    expect(put).not.toHaveBeenCalled()
    await userEvent.type(
      await screen.findByLabelText('Authenticator code or backup code'),
      '123456'
    )
    await userEvent.click(screen.getByRole('button', { name: 'Verify' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(
        '/api/user/',
        expect.objectContaining({
          id: 2,
          admin_permissions: { audit: { read: !allowed } },
        }),
        expect.objectContaining({
          headers: { 'X-Security-Proof': 'update-proof' },
          singleUseAuthorization: true,
        })
      )
    )
  }
)

it('root saving an administrator without changing permissions or password does not verify', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  const get = renderPermissions(100, true)
  const displayName = await screen.findByDisplayValue('Managed admin')
  await userEvent.clear(displayName)
  await userEvent.type(displayName, 'Renamed admin')
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith(
      '/api/user/',
      expect.objectContaining({ id: 2, display_name: 'Renamed admin' }),
      {}
    )
  )
  expect(put.mock.calls[0][1]).not.toHaveProperty('admin_permissions')
  expect(get).not.toHaveBeenCalledWith('/api/verify/methods', expect.anything())
})

it('admin cannot edit the audit permission even when the catalog is available', async () => {
  renderPermissions(10)
  await screen.findByDisplayValue('Managed admin')
  expect(screen.queryByRole('button', { name: label })).not.toBeInTheDocument()
})

it('editing a profile never exposes or submits a password', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  renderPermissions(100, true)
  await screen.findByDisplayValue('Managed admin')
  expect(screen.queryByLabelText('Password')).not.toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(put).toHaveBeenCalled())
  expect(put.mock.calls[0][1]).not.toHaveProperty('password')
})

it('password reset is separate and needs confirmation, not the old password or step-up', async () => {
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={new QueryClient()}>
      <UserPasswordDialog user={target} onClose={onClose} onSuccess={vi.fn()} />
    </QueryClientProvider>
  )
  expect(screen.queryByLabelText(/old password/i)).not.toBeInTheDocument()
  await userEvent.type(
    screen.getByLabelText('New Password'),
    'new-password-123'
  )
  await userEvent.type(
    screen.getByLabelText('Confirm Password'),
    'different-password'
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Review password reset' })
  )
  expect(
    await screen.findByText('Password confirmation does not match')
  ).toBeVisible()
  expect(post).not.toHaveBeenCalled()
  await userEvent.clear(screen.getByLabelText('Confirm Password'))
  await userEvent.type(
    screen.getByLabelText('Confirm Password'),
    'new-password-123'
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Review password reset' })
  )
  await screen.findByRole('alertdialog')
  expect(post).not.toHaveBeenCalled()
  await userEvent.click(screen.getByRole('button', { name: 'Reset password' }))
  await waitFor(() => expect(onClose).toHaveBeenCalledOnce())
  expect(post).toHaveBeenCalledExactlyOnceWith(
    '/api/user/2/password',
    {
      password: 'new-password-123',
      confirm_password: 'new-password-123',
      confirmed: true,
    },
    { singleUseAuthorization: true }
  )
})

it('gift adjustment previews and queues without submitting until final confirmation', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  renderPermissions(100)
  const displayName = await screen.findByDisplayValue('Managed admin')
  fireEvent.change(displayName, { target: { value: 'Unsaved profile change' } })
  await userEvent.click(
    screen.getByRole('button', { name: 'Adjust gift balance' })
  )
  const dialog = screen.getByRole('dialog', { name: 'Adjust gift balance' })
  expect(within(dialog).queryByLabelText('Wallet')).not.toBeInTheDocument()
  expect(within(dialog).getByLabelText('Input currency')).toBeEnabled()
  await userEvent.type(screen.getByLabelText('Input amount (USD)'), '2')
  expect(
    within(dialog).getByLabelText('Balance after adjustment')
  ).toHaveTextContent('$3')
  await userEvent.click(screen.getByRole('button', { name: 'Add to pending' }))
  expect(post).not.toHaveBeenCalled()
  expect(put).not.toHaveBeenCalled()
  await waitFor(() =>
    expect(
      screen.queryByRole('dialog', { name: 'Adjust gift balance' })
    ).not.toBeInTheDocument()
  )
  expect(screen.getByDisplayValue('Unsaved profile change')).toBeVisible()
  expect(await screen.findByText('Pending adjustments')).toBeVisible()
  expect(screen.getByText('Admin grant')).toBeVisible()
  expect(screen.getByLabelText('Total balance')).toHaveTextContent('$5')
  await userEvent.click(
    screen.getByRole('button', { name: 'Submit all changes' })
  )
  await screen.findByRole('alertdialog')
  expect(post).not.toHaveBeenCalled()
  expect(put).not.toHaveBeenCalled()
  await userEvent.click(
    within(screen.getByRole('alertdialog')).getByRole('button', {
      name: 'Cancel',
    })
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Submit all changes' })
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Confirm adjustments' })
  )
  await waitFor(() => expect(post).toHaveBeenCalled())
  expect(put).toHaveBeenCalledWith(
    '/api/user/',
    expect.objectContaining({ display_name: 'Unsaved profile change' }),
    {}
  )
  expect(put.mock.calls[0][1]).not.toHaveProperty('quota')
  expect(put.mock.invocationCallOrder[0]).toBeLessThan(
    post.mock.invocationCallOrder[0]
  )
  expect(post).toHaveBeenCalledExactlyOnceWith(
    '/api/user/2/quota_adjust',
    {
      funding_type: 'gift',
      delta_quota: 1000000,
      source_type: 'admin_grant',
      remark: '',
      revenue_usd: 0,
    },
    { singleUseAuthorization: true }
  )
})

it('paid refunds prefill negative revenue and allow overriding it before confirmation', async () => {
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  renderPermissions(100)
  await screen.findByDisplayValue('Managed admin')
  await userEvent.click(
    screen.getByRole('button', { name: 'Adjust paid balance' })
  )
  fireEvent.change(screen.getByLabelText('Input amount (CNY)'), {
    target: { value: '-14' },
  })
  expect(screen.getByLabelText('Adjustment amount (USD)')).toHaveValue(-2)
  expect(screen.getByLabelText('Recognized revenue (USD)')).toHaveValue(-2)
  fireEvent.change(screen.getByLabelText('Recognized revenue (USD)'), {
    target: { value: '-1.8' },
  })
  await userEvent.click(screen.getByRole('button', { name: 'Add to pending' }))
  expect(post).not.toHaveBeenCalled()
  await userEvent.click(
    screen.getByRole('button', { name: 'Submit all changes' })
  )
  await screen.findByRole('alertdialog')
  await userEvent.click(
    screen.getByRole('button', { name: 'Confirm adjustments' })
  )
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith(
      '/api/user/2/quota_adjust',
      expect.objectContaining({
        funding_type: 'paid',
        delta_quota: -1000000,
        source_type: 'refund',
        revenue_usd: -1.8,
      }),
      { singleUseAuthorization: true }
    )
  )
})

it('editing and removing drafts updates the parent preview without writing balances', async () => {
  const post = vi.spyOn(api, 'post')
  renderPermissions(100)
  await screen.findByDisplayValue('Managed admin')
  await userEvent.click(
    screen.getByRole('button', { name: 'Adjust gift balance' })
  )
  fireEvent.change(screen.getByLabelText('Input amount (USD)'), {
    target: { value: '2' },
  })
  await userEvent.click(screen.getByRole('button', { name: 'Add to pending' }))
  const pending = screen.getByRole('region', { name: 'Pending adjustments' })
  await userEvent.click(within(pending).getByRole('button', { name: 'Edit' }))
  expect(screen.getByLabelText('Input amount (USD)')).toHaveValue(2)
  fireEvent.change(screen.getByLabelText('Input amount (USD)'), {
    target: { value: '3' },
  })
  await userEvent.click(screen.getByRole('button', { name: 'Update pending' }))
  expect(screen.getByLabelText('Total balance')).toHaveTextContent('$6')
  await userEvent.click(within(pending).getByRole('button', { name: 'Delete' }))
  expect(
    screen.queryByRole('region', { name: 'Pending adjustments' })
  ).not.toBeInTheDocument()
  expect(screen.getByLabelText('Total balance')).toHaveTextContent('$3')
  expect(post).not.toHaveBeenCalled()
})

it('a partial failure keeps only unsubmitted drafts and retries without duplicating successful adjustments', async () => {
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValueOnce({ data: { success: true } })
    .mockResolvedValueOnce({
      data: { success: false, message: 'Adjustment rejected' },
    })
    .mockResolvedValue({ data: { success: true } })
  const get = renderPermissions(100)
  await screen.findByDisplayValue('Managed admin')
  for (const amount of ['2', '3']) {
    await userEvent.click(
      screen.getByRole('button', { name: 'Adjust gift balance' })
    )
    fireEvent.change(screen.getByLabelText('Input amount (USD)'), {
      target: { value: amount },
    })
    await userEvent.click(
      screen.getByRole('button', { name: 'Add to pending' })
    )
  }
  get.mockResolvedValue({
    data: {
      success: true,
      data: {
        ...target,
        quota: 2500000,
        paid_quota: 1000000,
        gift_quota: 1500000,
      },
    },
  })
  await userEvent.click(
    screen.getByRole('button', { name: 'Submit all changes' })
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Confirm adjustments' })
  )
  await waitFor(() =>
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  )
  const pending = screen.getByRole('region', { name: 'Pending adjustments' })
  expect(within(pending).getAllByRole('button', { name: 'Edit' })).toHaveLength(
    1
  )
  expect(
    screen.getByText('Some adjustments were submitted. Remaining: 1')
  ).toBeVisible()
  expect(screen.getByLabelText('Total balance')).toHaveTextContent('$8')
  await userEvent.click(
    screen.getByRole('button', { name: 'Submit all changes' })
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Confirm adjustments' })
  )
  await waitFor(() => expect(post).toHaveBeenCalledTimes(3))
  expect(
    post.mock.calls.map(
      (call) => (call[1] as { delta_quota: number }).delta_quota
    )
  ).toEqual([1000000, 1500000, 1500000])
})

it('profile save failure leaves the pending wallet adjustments untouched', async () => {
  vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: false, message: 'Profile rejected' },
  })
  const post = vi.spyOn(api, 'post')
  renderPermissions(100)
  await screen.findByDisplayValue('Managed admin')
  await userEvent.click(
    screen.getByRole('button', { name: 'Adjust gift balance' })
  )
  fireEvent.change(screen.getByLabelText('Input amount (USD)'), {
    target: { value: '2' },
  })
  await userEvent.click(screen.getByRole('button', { name: 'Add to pending' }))
  await userEvent.click(
    screen.getByRole('button', { name: 'Submit all changes' })
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Confirm adjustments' })
  )
  await waitFor(() =>
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  )
  expect(
    screen.getByRole('region', { name: 'Pending adjustments' })
  ).toBeVisible()
  expect(post).not.toHaveBeenCalled()
})

it('removing an earlier credit cannot leave a later deduction ready to overdraw the wallet', async () => {
  renderPermissions(100)
  await screen.findByDisplayValue('Managed admin')
  for (const amount of ['2', '-3']) {
    await userEvent.click(
      screen.getByRole('button', { name: 'Adjust gift balance' })
    )
    fireEvent.change(screen.getByLabelText('Input amount (USD)'), {
      target: { value: amount },
    })
    await userEvent.click(
      screen.getByRole('button', { name: 'Add to pending' })
    )
  }
  const pending = screen.getByRole('region', { name: 'Pending adjustments' })
  await userEvent.click(
    within(pending).getAllByRole('button', { name: 'Delete' })[0]
  )
  expect(
    screen.getByRole('button', { name: 'Submit all changes' })
  ).toBeDisabled()
  expect(within(pending).getByRole('alert')).toHaveTextContent(
    'Check the amount and pending wallet balances'
  )
})

it('an administrator gets the production gift entry without the root-only paid entry or system reason', async () => {
  renderPermissions(10)
  await screen.findByDisplayValue('Managed admin')
  expect(
    screen.queryByRole('button', { name: 'Adjust paid balance' })
  ).not.toBeInTheDocument()
  await userEvent.click(
    screen.getByRole('button', { name: 'Adjust gift balance' })
  )
  await userEvent.click(screen.getByLabelText('Source category'))
  expect(screen.getByRole('option', { name: 'Promotion' })).toBeVisible()
  expect(
    screen.queryByRole('option', { name: 'System adjustment' })
  ).not.toBeInTheDocument()
})

it('a CNY gift converts to USD, preserves its currency when edited and submits zero revenue', async () => {
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  renderPermissions(100)
  await screen.findByDisplayValue('Managed admin')
  await userEvent.click(
    screen.getByRole('button', { name: 'Adjust gift balance' })
  )
  await userEvent.click(screen.getByLabelText('Input currency'))
  await userEvent.click(screen.getByRole('option', { name: 'CNY' }))
  fireEvent.change(screen.getByLabelText('Input amount (CNY)'), {
    target: { value: '14' },
  })
  expect(screen.getByLabelText('Adjustment amount (USD)')).toHaveValue(2)
  expect(screen.getByLabelText('Balance after adjustment')).toHaveTextContent(
    '$3'
  )
  expect(screen.getByText('1 USD = 7 CNY', { exact: false })).toBeVisible()
  expect(
    screen.queryByLabelText('Recognized revenue (USD)')
  ).not.toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Add to pending' }))
  let pending = screen.getByRole('region', { name: 'Pending adjustments' })
  expect(pending).toHaveTextContent('14 CNY')
  await userEvent.click(within(pending).getByRole('button', { name: 'Edit' }))
  expect(screen.getByLabelText('Input amount (CNY)')).toHaveValue(14)
  fireEvent.change(screen.getByLabelText('Input amount (CNY)'), {
    target: { value: '-7' },
  })
  expect(screen.getByLabelText('Adjustment amount (USD)')).toHaveValue(-1)
  expect(screen.getByLabelText('Balance after adjustment')).toHaveTextContent(
    '$0'
  )
  await userEvent.click(screen.getByRole('button', { name: 'Update pending' }))
  pending = screen.getByRole('region', { name: 'Pending adjustments' })
  expect(pending).toHaveTextContent('-7 CNY')
  expect(post).not.toHaveBeenCalled()
  await userEvent.click(
    screen.getByRole('button', { name: 'Submit all changes' })
  )
  await userEvent.click(
    screen.getByRole('button', { name: 'Confirm adjustments' })
  )
  await waitFor(() =>
    expect(post).toHaveBeenCalledExactlyOnceWith(
      '/api/user/2/quota_adjust',
      expect.objectContaining({
        funding_type: 'gift',
        delta_quota: -500000,
        revenue_usd: 0,
      }),
      { singleUseAuthorization: true }
    )
  )
})

it.each([
  { currency: 'CNY', amount: '3.5', delta: 250000, after: '-$0.5' },
  { currency: 'USD', amount: '0.5', delta: 250000, after: '-$0.5' },
  { currency: 'CNY', amount: '7', delta: 500000, after: '$0' },
  { currency: 'CNY', amount: '14', delta: 1000000, after: '$1' },
])(
  'a negative paid wallet accepts a $amount $currency credit and previews $after before confirmation',
  async ({ currency, amount, delta, after }) => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true } })
    renderPermissions(100, undefined, {
      quota: -500000,
      paid_quota: -500000,
      gift_quota: 0,
    })
    await screen.findByDisplayValue('Managed admin')
    await userEvent.click(
      screen.getByRole('button', { name: 'Adjust paid balance' })
    )
    if (currency === 'USD') {
      await userEvent.click(screen.getByLabelText('Input currency'))
      await userEvent.click(screen.getByRole('option', { name: 'USD' }))
    }
    fireEvent.change(screen.getByLabelText(`Input amount (${currency})`), {
      target: { value: amount },
    })
    expect(screen.getByLabelText('Balance after adjustment')).toHaveTextContent(
      after
    )
    expect(screen.getByRole('button', { name: 'Add to pending' })).toBeEnabled()
    await userEvent.click(
      screen.getByRole('button', { name: 'Add to pending' })
    )
    expect(screen.getByLabelText('Total balance')).toHaveTextContent(after)
    expect(screen.getByLabelText('Paid wallet')).toHaveTextContent(after)
    expect(screen.getByLabelText('Gift wallet')).toHaveTextContent('$0')
    expect(
      screen.getAllByText('Current balance: -$1', { exact: false })
    ).toHaveLength(2)
    expect(put).not.toHaveBeenCalled()
    expect(post).not.toHaveBeenCalled()
    await userEvent.click(
      screen.getByRole('button', { name: 'Submit all changes' })
    )
    const review = await screen.findByRole('alertdialog')
    expect(within(review).getByLabelText('Total balance')).toHaveTextContent(
      after
    )
    expect(within(review).getByLabelText('Paid wallet')).toHaveTextContent(
      after
    )
    expect(put).not.toHaveBeenCalled()
    expect(post).not.toHaveBeenCalled()
    await userEvent.click(
      within(review).getByRole('button', { name: 'Confirm adjustments' })
    )
    await waitFor(() =>
      expect(post).toHaveBeenCalledExactlyOnceWith(
        '/api/user/2/quota_adjust',
        expect.objectContaining({
          funding_type: 'paid',
          delta_quota: delta,
          revenue_usd: delta / 500000,
        }),
        { singleUseAuthorization: true }
      )
    )
    expect(put.mock.calls[0][1]).not.toHaveProperty('quota')
    expect(put.mock.calls[0][1]).not.toHaveProperty('quota_dollars')
  }
)

it('a gift credit improves total debt without moving money out of the paid wallet', async () => {
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  renderPermissions(100, undefined, {
    quota: -500000,
    paid_quota: -500000,
    gift_quota: 0,
  })
  await screen.findByDisplayValue('Managed admin')
  await userEvent.click(
    screen.getByRole('button', { name: 'Adjust gift balance' })
  )
  fireEvent.change(screen.getByLabelText('Input amount (USD)'), {
    target: { value: '0.5' },
  })
  expect(screen.getByRole('button', { name: 'Add to pending' })).toBeEnabled()
  await userEvent.click(screen.getByRole('button', { name: 'Add to pending' }))
  expect(screen.getByLabelText('Total balance')).toHaveTextContent('-$0.5')
  expect(screen.getByLabelText('Paid wallet')).toHaveTextContent('-$1')
  expect(screen.getByLabelText('Gift wallet')).toHaveTextContent('$0.5')
  expect(post).not.toHaveBeenCalled()
  await userEvent.click(
    screen.getByRole('button', { name: 'Submit all changes' })
  )
  const review = await screen.findByRole('alertdialog')
  await userEvent.click(
    within(review).getByRole('button', { name: 'Confirm adjustments' })
  )
  await waitFor(() =>
    expect(post).toHaveBeenCalledExactlyOnceWith(
      '/api/user/2/quota_adjust',
      expect.objectContaining({
        funding_type: 'gift',
        delta_quota: 250000,
        revenue_usd: 0,
      }),
      { singleUseAuthorization: true }
    )
  )
})

it('a negative balance does not block a profile-only edit or send a balance override', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  const post = vi.spyOn(api, 'post')
  renderPermissions(100, true, {
    quota: -500000,
    paid_quota: -500000,
    gift_quota: 0,
  })
  const displayName = await screen.findByDisplayValue('Managed admin')
  fireEvent.change(displayName, { target: { value: 'Renamed debtor' } })
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledExactlyOnceWith(
      '/api/user/',
      expect.objectContaining({ id: 2, display_name: 'Renamed debtor' }),
      {}
    )
  )
  for (const field of ['quota', 'quota_dollars', 'paid_quota', 'gift_quota']) {
    expect(put.mock.calls[0][1]).not.toHaveProperty(field)
  }
  expect(post).not.toHaveBeenCalled()
})

it('invalid loaded profile data shows a validation error instead of silently ignoring submit', async () => {
  const error = vi.spyOn(toast, 'error')
  const put = vi.spyOn(api, 'put')
  const post = vi.spyOn(api, 'post')
  renderPermissions(100, undefined, { username: '' })
  await screen.findByDisplayValue('Managed admin')
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() =>
    expect(error).toHaveBeenCalledWith(
      'Please fix the highlighted fields before saving'
    )
  )
  expect(screen.getByLabelText('Username')).toHaveAttribute(
    'aria-invalid',
    'true'
  )
  expect(put).not.toHaveBeenCalled()
  expect(post).not.toHaveBeenCalled()
})

it.each([
  { name: 'zero adjustment', delta: 0 },
  { name: 'fractional quota', delta: 0.5 },
  { name: 'NaN adjustment', delta: Number.NaN },
  { name: 'infinite adjustment', delta: Number.POSITIVE_INFINITY },
  { name: 'unsafe adjustment', delta: Number.MAX_SAFE_INTEGER + 1 },
  { name: 'resulting balance overflow', delta: Number.MAX_SAFE_INTEGER },
])('wallet previews reject $name', ({ delta }) => {
  const draft: WalletAdjustmentDraft = {
    id: 1,
    funding_type: 'paid',
    delta_quota: delta,
    source_type: 'manual_topup',
    revenue_usd: 0,
    remark: '',
    currency: 'USD',
    amount: '1',
  }
  expect(
    previewWalletAdjustments(
      { ...target, quota: 500000, paid_quota: 500000, gift_quota: 0 },
      [draft]
    ).valid
  ).toBe(false)
})

it('deductions remain blocked while another wallet is negative, even if the total stays non-negative', () => {
  const deduction: WalletAdjustmentDraft = {
    id: 1,
    funding_type: 'gift',
    delta_quota: -250000,
    source_type: 'system_adjustment',
    revenue_usd: 0,
    remark: '',
    currency: 'USD',
    amount: '-0.5',
  }
  const preview = previewWalletAdjustments(
    { ...target, quota: 250000, paid_quota: -500000, gift_quota: 750000 },
    [deduction]
  )
  expect(preview.balances).toEqual({ total: 0, paid: -500000, gift: 500000 })
  expect(preview.valid).toBe(false)
})

it('wallet debt must be cleared before a queued deduction and removing that credit invalidates the queue', () => {
  const debtor = {
    ...target,
    quota: -500000,
    paid_quota: -500000,
    gift_quota: 0,
  }
  const credit: WalletAdjustmentDraft = {
    id: 1,
    funding_type: 'paid',
    delta_quota: 750000,
    source_type: 'manual_topup',
    revenue_usd: 1.5,
    remark: '',
    currency: 'USD',
    amount: '1.5',
  }
  const deduction: WalletAdjustmentDraft = {
    ...credit,
    id: 2,
    delta_quota: -250000,
    source_type: 'refund',
    revenue_usd: -0.5,
    amount: '-0.5',
  }
  const before = { ...debtor }
  expect(previewWalletAdjustments(debtor, [credit, deduction])).toMatchObject({
    balances: { total: 0, paid: 0, gift: 0 },
    valid: true,
  })
  expect(previewWalletAdjustments(debtor, [deduction]).valid).toBe(false)
  expect(previewWalletAdjustments(debtor, [deduction, credit]).valid).toBe(
    false
  )
  expect(debtor).toEqual(before)
})
