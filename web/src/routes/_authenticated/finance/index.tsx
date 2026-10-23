import { createFileRoute, redirect } from '@tanstack/react-router'

import { FinanceCenter } from '@/features/finance'
import { hasPermission } from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/finance/')({
  beforeLoad: () => {
    const user = useAuthStore.getState().auth.user
    if (
      !user ||
      user.role < ROLE.ADMIN ||
      (!hasPermission(user, 'finance', 'read') &&
        !hasPermission(user, 'finance', 'audit'))
    ) {
      throw redirect({ to: '/403' })
    }
  },
  component: FinanceCenter,
})
