/*
Copyright (C) 2023-2026 Xauryan

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

For commercial licensing, please contact support@Xauryan.com
*/
import { useEffect, useState } from 'react'
import { ApiKeysDeleteDialog } from './api-keys-delete-dialog'
import { ApiKeysMutateDrawer } from './api-keys-mutate-drawer'
import { useApiKeys } from './api-keys-provider'
import { CCSwitchDialog } from './dialogs/cc-switch-dialog'

export function ApiKeysDialogs() {
  const { open, setOpen, currentRow, resolvedKey } = useApiKeys()
  const [lastMutateSide, setLastMutateSide] = useState<'left' | 'right'>(
    'right'
  )
  const mutateSide =
    open === 'create' ? 'left' : open === 'update' ? 'right' : lastMutateSide

  useEffect(() => {
    // Tracks the most recently opened drawer side so that, when the drawer
    // animates out, it slides back the way it came in. This is a "remember
    // last value" pattern — the alternative (lifting the setter into the
    // caller of setOpen) would require changing the useApiKeys() context.
    if (open === 'create') {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setLastMutateSide('left')
    } else if (open === 'update') {
      setLastMutateSide('right')
    }
  }, [open])

  return (
    <>
      <ApiKeysMutateDrawer
        open={open === 'create' || open === 'update'}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
        currentRow={open === 'update' ? currentRow || undefined : undefined}
        side={mutateSide}
      />
      <ApiKeysDeleteDialog />
      <CCSwitchDialog
        open={open === 'cc-switch'}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
        tokenKey={resolvedKey}
      />
    </>
  )
}
