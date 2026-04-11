/** Элемент списка GET /api/v1/users/{user_id}/avatars */
export type AvatarListItem = {
  id: string
}

/** Ответ POST /api/v1/avatars (POST /web/upload в проде проксируется Nginx сюда же) */
export type AvatarCreateResponse = {
  id: string
  user_id?: string
  url?: string
  status?: string
  created_at?: string
}

/** Тело ошибки API в формате спецификации */
export type ApiErrorBody = {
  error?: string
  details?: string
  max_size?: number
}
