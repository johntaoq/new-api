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
import { useQuery } from '@tanstack/react-query'
import { lazy, Suspense } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { useAuthStore } from '@/stores/auth-store'

import { getLogDetail } from '../../api'
import type { UsageLog } from '../../data/schema'

const DetailsDialog = lazy(() =>
  import('./details-dialog').then((module) => ({
    default: module.DetailsDialog,
  }))
)

interface LazyDetailsDialogProps {
  log: UsageLog
  isAdmin: boolean
  isRoot: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function LazyDetailsDialog(props: LazyDetailsDialogProps) {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const detail = useQuery({
    queryKey: [
      'log-detail',
      user?.id,
      user?.role,
      props.isAdmin,
      props.isRoot,
      props.log.user_id,
      props.log.detail_id,
      props.log.request_id,
      props.log.created_at,
      props.log.type,
    ],
    queryFn: ({ signal }) => getLogDetail(props.log, props.isAdmin, signal),
    enabled: props.open && props.log.summary === true,
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: false,
    refetchOnWindowFocus: false,
    meta: { errorToast: false },
  })
  if (!props.open) return null

  const pendingDialog = (
    <Dialog
      title={t('Log Details')}
      description={t('Click to view full details')}
      descriptionClassName='sr-only'
      open={props.open}
      onOpenChange={props.onOpenChange}
      contentHeight='min(72dvh, 720px)'
    >
      {detail.isError ? (
        <ErrorState
          title={t('Failed to load logs')}
          onRetry={() => void detail.refetch()}
        />
      ) : (
        <LoadingState />
      )}
    </Dialog>
  )
  if (props.log.summary && !detail.data) return pendingDialog

  return (
    <Suspense fallback={pendingDialog}>
      <DetailsDialog {...props} log={detail.data ?? props.log} />
    </Suspense>
  )
}
