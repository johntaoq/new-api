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
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { Pic365Authorize } from '@/features/pic365'

export const Route = createFileRoute('/_authenticated/pic365')({
  validateSearch: z.object({
    state: z.string().regex(/^[A-Za-z0-9_-]{43}$/),
    code_challenge: z.string().regex(/^[A-Za-z0-9_-]{43}$/),
  }),
  component: Page,
})

function Page() {
  const search = Route.useSearch()
  return (
    <Pic365Authorize state={search.state} challenge={search.code_challenge} />
  )
}
