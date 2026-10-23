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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { PasswordInput } from '@/components/password-input'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { handleServerError } from '@/lib/handle-server-error'
import { requireServerSuccess } from '@/lib/server-error-message'

import { resetManagedUserPassword } from '../api'
import {
  resetUserPasswordSchema,
  type ResetUserPasswordValues,
} from '../lib/user-form'
import type { User } from '../types'

export function UserPasswordDialog(props: {
  user: User
  onClose: () => void
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [review, setReview] = useState(false)
  const form = useForm<ResetUserPasswordValues>({
    resolver: zodResolver(resetUserPasswordSchema),
    defaultValues: { password: '', confirm_password: '' },
  })
  const reset = useMutation({
    mutationFn: async (values: ResetUserPasswordValues) =>
      requireServerSuccess(
        await resetManagedUserPassword(
          props.user.id,
          values.password,
          values.confirm_password
        )
      ),
    onSuccess: () => {
      form.reset()
      toast.success(t('Password reset successfully'))
      props.onSuccess()
      props.onClose()
    },
    onError: (error) => handleServerError(error, t('Failed to reset password')),
  })
  return (
    <>
      <Dialog
        open={!review}
        onOpenChange={(open) => {
          if (!open && !review) props.onClose()
        }}
        title={t('Reset password')}
        description={`${props.user.username} (#${props.user.id})`}
        footer={
          <>
            <Button variant='outline' onClick={props.onClose}>
              {t('Cancel')}
            </Button>
            <Button type='submit' form='reset-managed-password'>
              {t('Review password reset')}
            </Button>
          </>
        }
      >
        <Form {...form}>
          <form
            id='reset-managed-password'
            className='space-y-4'
            onSubmit={form.handleSubmit(() => setReview(true))}
          >
            <FormField
              control={form.control}
              name='password'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('New Password')}</FormLabel>
                  <FormControl>
                    <PasswordInput {...field} autoComplete='new-password' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='confirm_password'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Confirm Password')}</FormLabel>
                  <FormControl>
                    <PasswordInput {...field} autoComplete='new-password' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </form>
        </Form>
      </Dialog>
      <ConfirmDialog
        open={review}
        onOpenChange={(open) => {
          if (!reset.isPending) setReview(open)
        }}
        title={t('Confirm password reset')}
        desc={t(
          'Reset the password for {{username}}? Existing sessions will be signed out.',
          { username: props.user.username }
        )}
        confirmText={t('Reset password')}
        handleConfirm={() => {
          if (!reset.isPending) reset.mutate(form.getValues())
        }}
        isLoading={reset.isPending}
      />
    </>
  )
}
