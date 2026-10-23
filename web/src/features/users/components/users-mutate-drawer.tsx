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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  SideDrawerSection,
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { PermissionMatrix } from '@/components/permission-matrix'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Textarea } from '@/components/ui/textarea'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  EMPTY_PERMISSION_CATALOG,
  hasPermission,
  normalizeAdminPermissions,
  type AdminPermissionMatrix,
} from '@/lib/admin-permissions'
import { handleServerError } from '@/lib/handle-server-error'
import { accountPasswordSchema } from '@/lib/password-policy'
import { ROLE } from '@/lib/roles'
import { AuthOperationError } from '@/lib/secure-verification'
import { requireServerSuccess } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import {
  createUser,
  updateUser,
  getUser,
  getGroups,
  getPermissionCatalog,
  adjustUserWallet,
} from '../api'
import { BINDING_FIELDS, ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import {
  userFormSchema,
  type UserFormValues,
  USER_FORM_DEFAULT_VALUES,
  transformFormDataToPayload,
  transformUserToFormDefaults,
} from '../lib'
import {
  previewWalletAdjustments,
  type WalletAdjustmentDraft,
} from '../lib/wallet-adjustments'
import type { User } from '../types'
import { UserWalletAdjustments } from './user-wallet-adjustments'
import { useUsers } from './users-provider'

type UsersMutateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: User
}

export function UsersMutateDrawer({
  open,
  onOpenChange,
  currentRow,
}: UsersMutateDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow
  const { triggerRefresh, requestVerification } = useUsers()
  const currentUser = useAuthStore((s) => s.auth.user)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [walletUser, setWalletUser] = useState<User | null>(null)
  const [pending, setPending] = useState<WalletAdjustmentDraft[]>([])
  const [review, setReview] = useState<UserFormValues | null>(null)
  const [partialFailure, setPartialFailure] = useState(false)
  const submitting = useRef(false)
  // Matrix as loaded from the server; an unchanged matrix is not resubmitted so
  // routine edits of an administrator do not require step-up verification.
  const loadedPermissions = useRef<AdminPermissionMatrix | undefined>(undefined)

  // Fetch groups
  const { data: groupsData } = useQuery({
    queryKey: ['groups'],
    queryFn: async () => requireServerSuccess(await getGroups()),
    staleTime: 5 * 60 * 1000,
  })

  const groups = groupsData?.data || []

  // Permission catalog is owned by the backend; fetched once and reused.
  const { data: permissionCatalog = EMPTY_PERMISSION_CATALOG } = useQuery({
    queryKey: ['admin-permission-catalog'],
    queryFn: async () => requireServerSuccess(await getPermissionCatalog()),
    staleTime: 5 * 60 * 1000,
  })

  const form = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema),
    defaultValues: USER_FORM_DEFAULT_VALUES,
  })

  // Load existing data when updating
  useEffect(() => {
    let cancelled = false
    setWalletUser(null)
    setPending([])
    setReview(null)
    setPartialFailure(false)
    if (open && isUpdate && currentRow) {
      // For update, fetch fresh data
      getUser(currentRow.id)
        .then((result) => {
          if (cancelled) return
          if (result.success && result.data) {
            setWalletUser(result.data)
            loadedPermissions.current = result.data.admin_permissions
            form.reset(transformUserToFormDefaults(result.data))
          } else {
            handleServerError(result, t('Failed to load'))
          }
        })
        .catch((error) => {
          if (!cancelled) handleServerError(error, t('Failed to load'))
        })
    } else if (open && !isUpdate) {
      // For create, reset to defaults
      form.reset(USER_FORM_DEFAULT_VALUES)
    }
    return () => {
      cancelled = true
    }
  }, [open, isUpdate, currentRow, form, t])

  const selectedRole = form.watch('role')
  const canEditAdminPermissions = currentUser?.role === ROLE.SUPER_ADMIN
  const targetIsAdmin = (selectedRole ?? currentRow?.role ?? 0) >= ROLE.ADMIN

  const pendingValid =
    !!walletUser && previewWalletAdjustments(walletUser, pending).valid

  const onSubmit = async (data: UserFormValues, confirmed = false) => {
    if (submitting.current) return
    if (pending.length > 0) {
      if (!pendingValid) return
      if (!confirmed) {
        setReview(data)
        return
      }
    }
    if (!isUpdate) {
      if (!accountPasswordSchema.safeParse(data.password ?? '').success) {
        form.setError('password', {
          type: 'manual',
          message: t('Password must contain between 8 and 128 characters.'),
        })
        return
      }
    }

    submitting.current = true
    setIsSubmitting(true)
    try {
      const payload = transformFormDataToPayload(
        data,
        currentRow?.id,
        permissionCatalog
      )
      if (
        isUpdate &&
        payload.admin_permissions &&
        JSON.stringify(payload.admin_permissions) ===
          JSON.stringify(
            normalizeAdminPermissions(
              loadedPermissions.current,
              permissionCatalog
            )
          )
      ) {
        delete payload.admin_permissions
      }
      // Rewriting admin permissions or creating an
      // administrator changes who can sign in or what they may do.
      let proofToken: string | undefined
      if (isUpdate && currentRow) {
        if (payload.admin_permissions) {
          const proof = await requestVerification({
            scope: 'admin.user.update',
            context: { user_id: currentRow.id },
            title: t('Verify to update user credentials'),
            description: t(
              'Confirm your identity before changing the account {{username}}.',
              { username: currentRow.username }
            ),
          })
          if (!proof) return
          proofToken = proof.proof_token
        }
      } else if ((payload.role ?? 0) >= ROLE.ADMIN) {
        const proof = await requestVerification({
          scope: 'admin.user.create',
          context: { role: payload.role ?? ROLE.ADMIN },
          title: t('Verify to create administrator'),
          description: t(
            'Confirm your identity before creating an administrator account.'
          ),
        })
        if (!proof) return
        proofToken = proof.proof_token
      }
      const result = isUpdate
        ? await updateUser(
            payload as typeof payload & { id: number },
            proofToken
          )
        : await createUser(payload, proofToken)

      if (result.success) {
        loadedPermissions.current = data.admin_permissions
        let applied = 0
        try {
          for (const item of pending) {
            if (!currentRow) {
              throw new Error('User is required for wallet adjustments')
            }
            requireServerSuccess(
              await adjustUserWallet(currentRow.id, {
                funding_type: item.funding_type,
                delta_quota: item.delta_quota,
                source_type: item.source_type,
                remark: item.remark,
                revenue_usd: item.revenue_usd,
              })
            )
            applied++
          }
        } catch (error) {
          setPending(pending.slice(applied))
          setPartialFailure(applied > 0)
          // Refresh only balances: retain the user's profile edits and unsent drafts.
          await refreshWalletData()
          triggerRefresh()
          handleServerError(error, t('Failed to adjust quota'))
          return
        }
        setPending([])
        toast.success(
          isUpdate
            ? t(SUCCESS_MESSAGES.USER_UPDATED)
            : t(SUCCESS_MESSAGES.USER_CREATED)
        )
        onOpenChange(false)
        triggerRefresh()
      } else {
        handleServerError(result, t(ERROR_MESSAGES.CREATE_FAILED))
      }
    } catch (error) {
      handleServerError(
        AuthOperationError.from(error),
        t(ERROR_MESSAGES.UNEXPECTED)
      )
    } finally {
      submitting.current = false
      setIsSubmitting(false)
      setReview(null)
    }
  }

  const refreshWalletData = async () => {
    if (!currentRow) return
    try {
      const result = requireServerSuccess(await getUser(currentRow.id))
      setWalletUser(result.data ?? null)
    } catch (error) {
      setWalletUser(null)
      handleServerError(error, t('Failed to load'))
    }
  }

  return (
    <>
      <Sheet
        open={open}
        onOpenChange={(v) => {
          if (submitting.current) return
          onOpenChange(v)
          if (!v) {
            form.reset()
          }
        }}
      >
        <SheetContent
          className={sideDrawerContentClassName('sm:max-w-[680px]')}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>
              {isUpdate ? t('Update') : t('Create')} {t('User')}
            </SheetTitle>
            <SheetDescription>
              {isUpdate
                ? t('Update the user by providing necessary info.')
                : t('Add a new user by providing necessary info.')}
            </SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form
              id='user-form'
              onSubmit={form.handleSubmit(
                (data) => onSubmit(data),
                () =>
                  toast.error(
                    t('Please fix the highlighted fields before saving')
                  )
              )}
              className={sideDrawerFormClassName()}
            >
              {/* Basic Information */}
              <SideDrawerSection>
                <h3 className='text-sm font-medium'>
                  {t('Basic Information')}
                </h3>

                <FormField
                  control={form.control}
                  name='username'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Username')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          placeholder={t('Enter username')}
                          disabled={isUpdate}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {!isUpdate && (
                  <FormField
                    control={form.control}
                    name='role'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Role')}</FormLabel>
                        <Select
                          items={[
                            { value: '1', label: t('Common User') },
                            { value: '10', label: t('Admin') },
                          ]}
                          onValueChange={(value) =>
                            value !== null &&
                            field.onChange(Number.parseInt(value))
                          }
                          value={String(field.value)}
                        >
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder={t('Select a role')} />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent alignItemWithTrigger={false}>
                            <SelectGroup>
                              <SelectItem value='1'>
                                {t('Common User')}
                              </SelectItem>
                              <SelectItem value='10'>{t('Admin')}</SelectItem>
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                        <FormDescription>
                          {t("Set the user's role (cannot be Root)")}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}

                <FormField
                  control={form.control}
                  name='display_name'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Display Name')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          placeholder={t('Enter display name')}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Leave empty to use username')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {!isUpdate && (
                  <FormField
                    control={form.control}
                    name='password'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Password')}</FormLabel>
                        <FormControl>
                          <Input
                            {...field}
                            type='password'
                            placeholder={
                              isUpdate
                                ? t('Leave empty to keep unchanged')
                                : t('Enter password (8–128 characters)')
                            }
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
              </SideDrawerSection>

              {/* Group & Quota Settings (Update only) */}
              {isUpdate && (
                <SideDrawerSection>
                  <h3 className='text-sm font-medium'>{t('Group & Quota')}</h3>

                  <FormField
                    control={form.control}
                    name='group'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Group')}</FormLabel>
                        <FormControl>
                          <Combobox
                            options={groups.map((group) => ({
                              value: group,
                              label: group,
                            }))}
                            onValueChange={field.onChange}
                            value={field.value}
                            className='w-full'
                            placeholder={t('Select a group')}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  {walletUser ? (
                    <UserWalletAdjustments
                      key={walletUser.id}
                      user={walletUser}
                      pending={pending}
                      onChange={setPending}
                      disabled={isSubmitting}
                    />
                  ) : (
                    <Button
                      type='button'
                      variant='outline'
                      onClick={refreshWalletData}
                    >
                      {t('Reload balances')}
                    </Button>
                  )}
                  {partialFailure && (
                    <p role='alert' className='text-destructive text-sm'>
                      {t(
                        'Some adjustments were submitted. Remaining: {{count}}',
                        { count: pending.length }
                      )}
                    </p>
                  )}

                  <FormField
                    control={form.control}
                    name='remark'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Remark')}</FormLabel>
                        <FormControl>
                          <Textarea
                            {...field}
                            placeholder={t(
                              'Admin notes (only visible to admins)'
                            )}
                            rows={3}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </SideDrawerSection>
              )}

              {canEditAdminPermissions &&
                targetIsAdmin &&
                permissionCatalog.resources.length > 0 && (
                  <SideDrawerSection>
                    <h3 className='text-sm font-medium'>
                      {t('Admin Permissions')}
                    </h3>
                    <p className='text-muted-foreground text-xs'>
                      {t(
                        'Default administrator permissions can be overridden for this user.'
                      )}
                    </p>
                    <FormField
                      control={form.control}
                      name='admin_permissions'
                      render={({ field }) => {
                        const selected = normalizeAdminPermissions(
                          field.value,
                          permissionCatalog
                        )
                        return (
                          <FormItem>
                            <PermissionMatrix
                              resources={permissionCatalog.resources}
                              value={selected}
                              onChange={field.onChange}
                            />
                            <FormMessage />
                          </FormItem>
                        )
                      }}
                    />
                    {currentUser && (
                      <p className='text-muted-foreground text-xs'>
                        {hasPermission(
                          currentUser,
                          ADMIN_PERMISSION_RESOURCES.CHANNEL,
                          ADMIN_PERMISSION_ACTIONS.SENSITIVE_WRITE
                        )
                          ? t(
                              'Your account can edit sensitive channel settings.'
                            )
                          : t(
                              'Your account cannot edit sensitive channel settings.'
                            )}
                      </p>
                    )}
                  </SideDrawerSection>
                )}

              {/* Binding Information (Read-only) */}
              {isUpdate && (
                <SideDrawerSection>
                  <h3 className='text-sm font-medium'>
                    {t('Binding Information')}
                  </h3>
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'Third-party account bindings (read-only, managed by user in Security & Access)'
                    )}
                  </p>

                  <div className='flex flex-col gap-3'>
                    {BINDING_FIELDS.map(({ key, label }) => (
                      <div key={key}>
                        <Label className='text-muted-foreground text-xs'>
                          {t(label)}
                        </Label>
                        <Input
                          value={
                            (currentRow?.[key as keyof User] as string) || '-'
                          }
                          disabled
                          className='mt-1'
                        />
                      </div>
                    ))}
                  </div>
                </SideDrawerSection>
              )}
            </form>
          </Form>
          <SheetFooter className={sideDrawerFooterClassName()}>
            <SheetClose
              render={<Button variant='outline' disabled={isSubmitting} />}
            >
              {t('Close')}
            </SheetClose>
            <Button
              form='user-form'
              type='submit'
              disabled={isSubmitting || (pending.length > 0 && !pendingValid)}
            >
              {isSubmitting
                ? t('Saving...')
                : t(pending.length > 0 ? 'Submit all changes' : 'Save changes')}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      {walletUser && (
        <ConfirmDialog
          open={open && review !== null}
          onOpenChange={(value) => {
            if (!value && !submitting.current) setReview(null)
          }}
          title={t('Confirm wallet adjustments')}
          desc={
            <div className='max-h-[65dvh] space-y-4 overflow-y-auto'>
              <p>
                {walletUser.username} (#{walletUser.id})
              </p>
              <UserWalletAdjustments user={walletUser} pending={pending} />
            </div>
          }
          className='sm:max-w-[680px]'
          confirmText={t('Confirm adjustments')}
          disabled={!pendingValid}
          isLoading={isSubmitting}
          handleConfirm={() => {
            if (review) void onSubmit(review, true)
          }}
        />
      )}
    </>
  )
}
