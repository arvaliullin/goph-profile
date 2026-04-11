/** Маршруты SPA; GET /web/* отдаёт Nginx как index.html. */
export enum AppRoute {
  Root = '/',
  Upload = '/web/upload',
  Gallery = '/web/gallery/:userId',
}

export const USER_STORAGE_KEY = 'goph-profile-user-id'

/** Лимит размера файла по спецификации (10 MiB). */
export const MAX_FILE_SIZE_BYTES = 10 * 1024 * 1024
