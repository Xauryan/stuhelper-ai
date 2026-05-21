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
import { create } from 'zustand'

export type UserPermissions = {
  sidebar_settings?: boolean
  sidebar_modules?: Record<string, unknown>
}

export interface AuthUser {
  id: number
  username: string
  display_name?: string
  email?: string
  role: number
  status?: number
  group?: string
  quota?: number
  used_quota?: number
  request_count?: number
  aff_code?: string
  aff_count?: number
  aff_quota?: number
  aff_history_quota?: number
  inviter_id?: number
  github_id?: string
  oidc_id?: string
  wechat_id?: string
  telegram_id?: string
  linux_do_id?: string
  setting?: Record<string, unknown> | string
  stripe_customer?: string
  sidebar_modules?: string
  permissions?: UserPermissions
}

const USER_STORAGE_KEY = 'user'
const USER_ID_STORAGE_KEY = 'uid'

function persistUser(user: AuthUser | null): void {
  if (typeof window === 'undefined') return

  if (user) {
    window.localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(user))
    if (user.id != null) {
      window.localStorage.setItem(USER_ID_STORAGE_KEY, String(user.id))
    } else {
      window.localStorage.removeItem(USER_ID_STORAGE_KEY)
    }
    return
  }

  window.localStorage.removeItem(USER_STORAGE_KEY)
  window.localStorage.removeItem(USER_ID_STORAGE_KEY)
}

interface AuthState {
  auth: {
    user: AuthUser | null
    setUser: (user: AuthUser | null) => void
    reset: () => void
  }
}

export const useAuthStore = create<AuthState>()((set) => {
  // Restore user info from localStorage
  const initUser = (() => {
    try {
      if (typeof window !== 'undefined') {
        const saved = window.localStorage.getItem(USER_STORAGE_KEY)
        return saved ? JSON.parse(saved) : null
      }
    } catch {
      // Clear dirty data when parsing fails
      if (typeof window !== 'undefined') {
        window.localStorage.removeItem(USER_STORAGE_KEY)
        window.localStorage.removeItem(USER_ID_STORAGE_KEY)
      }
    }
    return null
  })()

  return {
    auth: {
      user: initUser,
      setUser: (user) =>
        set((state) => {
          persistUser(user)
          return { ...state, auth: { ...state.auth, user } }
        }),
      reset: () =>
        set((state) => {
          persistUser(null)
          return {
            ...state,
            auth: { ...state.auth, user: null },
          }
        }),
    },
  }
})
