export type AvatarListItem = {
  id: string
}

export type AvatarCreateResponse = {
  id: string
  user_id?: string
  url?: string
  status?: string
  created_at?: string
}

export type ApiErrorBody = {
  error?: string
  details?: string
  max_size?: number
}
