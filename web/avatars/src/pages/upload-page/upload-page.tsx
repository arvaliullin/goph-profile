import { useCallback, useId, useRef, useState, type DragEvent } from 'react'
import { MAX_FILE_SIZE_BYTES } from '../../consts'
import type { AvatarCreateResponse } from '../../types/avatar'
import { formatApiMessage } from '../../utils/format-api-message'
import { loadUserId, saveUserId } from '../../utils/user-id-storage'
import './upload-page.css'

export default function UploadPage() {
  const fileInputId = useId()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [userId, setUserId] = useState(loadUserId)
  const [preview, setPreview] = useState<string | null>(null)
  const [file, setFile] = useState<File | null>(null)
  const [status, setStatus] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [dragOver, setDragOver] = useState(false)

  const revokePreview = useCallback((url: string | null) => {
    if (url) {
      URL.revokeObjectURL(url)
    }
  }, [])

  const onPick = useCallback(
    (f: File | null) => {
      setFile(f)
      setPreview((prev) => {
        revokePreview(prev)
        if (!f) {
          return null
        }
        return URL.createObjectURL(f)
      })
    },
    [revokePreview],
  )

  const submitApi = async () => {
    if (!userId.trim()) {
      setStatus('Укажите идентификатор пользователя')
      return
    }
    if (!file) {
      setStatus('Выберите файл')
      return
    }
    if (file.size > MAX_FILE_SIZE_BYTES) {
      setStatus(`Файл больше ${MAX_FILE_SIZE_BYTES} байт`)
      return
    }
    saveUserId(userId.trim())
    setBusy(true)
    setStatus(null)
    const fd = new FormData()
    fd.append('file', file)
    try {
      const res = await fetch('/api/v1/avatars', {
        method: 'POST',
        headers: { 'X-User-ID': userId.trim() },
        body: fd,
      })
      const body: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setStatus(formatApiMessage(body, res.statusText || 'Ошибка запроса'))
        return
      }
      const created = body as AvatarCreateResponse
      setStatus(`Создано: ${created.id}${created.status ? ` (${created.status})` : ''}`)
    } catch (e) {
      setStatus(String(e))
    } finally {
      setBusy(false)
    }
  }

  const onDragOver = (e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(true)
  }

  const onDragLeave = (e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(false)
  }

  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(false)
    const f = e.dataTransfer.files?.[0]
    if (f && f.type.startsWith('image/')) {
      onPick(f)
    }
  }

  const openFileDialog = () => {
    fileInputRef.current?.click()
  }

  return (
    <div className="page page--gray">
      <main className="page__main container">
        <section className="upload">
          <h1 className="upload__title">Загрузка аватара</h1>
          <form
            className="upload__form form"
            onSubmit={(e) => {
              e.preventDefault()
            }}
          >
            <div className="form__row">
              <label className="form__label" htmlFor="user-id">
                Ваш идентификатор пользователя
              </label>
              <input
                id="user-id"
                className="form__input"
                value={userId}
                onChange={(e) => setUserId(e.target.value)}
                placeholder="demo-user"
                autoComplete="username"
              />
            </div>

            <div className="form__row">
              <span className="form__label" id={fileInputId + '-label'}>
                Файл (JPEG, PNG, WebP, до 10 МБ)
              </span>
              <input
                ref={fileInputRef}
                id={fileInputId}
                className="visually-hidden"
                type="file"
                accept="image/jpeg,image/png,image/webp"
                aria-labelledby={fileInputId + '-label'}
                onChange={(e) => onPick(e.target.files?.[0] ?? null)}
              />
              <div
                className={`upload__dropzone ${dragOver ? 'upload__dropzone--active' : ''}`}
                onDragOver={onDragOver}
                onDragLeave={onDragLeave}
                onDrop={onDrop}
                onClick={openFileDialog}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    openFileDialog()
                  }
                }}
                role="button"
                tabIndex={0}
                aria-label="Выбрать файл или перетащить сюда"
              >
                {file ? (
                  <span className="upload__dropzone-text">{file.name}</span>
                ) : (
                  <span className="upload__dropzone-text">
                    Перетащите изображение сюда или нажмите, чтобы выбрать
                  </span>
                )}
              </div>
            </div>

            {preview && (
              <div className="upload__preview">
                <img className="upload__preview-img" src={preview} alt="Предпросмотр" />
              </div>
            )}

            <div className="upload__actions">
              <button
                className="button button--primary"
                type="button"
                disabled={busy}
                onClick={submitApi}
              >
                {busy ? 'Загрузка…' : 'Загрузить аватар'}
              </button>
            </div>

            {status && (
              <pre className="upload__status" role="status">
                {status}
              </pre>
            )}
          </form>
        </section>
      </main>
    </div>
  )
}
