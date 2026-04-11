export enum AppRoute {
  Root = '/',
  Upload = '/web/upload',
  MyGallery = '/web/gallery',
  GalleryByUser = '/web/gallery/:userId',
}

export const USER_STORAGE_KEY = 'goph-profile-user-id'

export const MAX_FILE_SIZE_BYTES = 10 * 1024 * 1024
