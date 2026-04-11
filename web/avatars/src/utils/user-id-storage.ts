import { USER_STORAGE_KEY } from '../consts'

export function loadUserId(): string {
  return localStorage.getItem(USER_STORAGE_KEY) ?? ''
}

export function saveUserId(id: string) {
  localStorage.setItem(USER_STORAGE_KEY, id)
}
