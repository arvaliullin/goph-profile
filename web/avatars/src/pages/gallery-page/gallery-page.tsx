import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import AvatarCard from '../../components/avatar-card/avatar-card'
import { USER_STORAGE_KEY } from '../../consts'
import type { AvatarListItem } from '../../types/avatar'
import './gallery-page.css'

export default function GalleryPage() {
  const { userId = '' } = useParams()
  const [me, setMe] = useState(() => localStorage.getItem(USER_STORAGE_KEY) ?? '')
  const [items, setItems] = useState<AvatarListItem[] | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      setErr(null)
      try {
        const res = await fetch(`/api/v1/users/${encodeURIComponent(userId)}/avatars`)
        if (!res.ok) {
          setErr(await res.text())
          return
        }
        const data: unknown = await res.json()
        if (!cancelled && Array.isArray(data)) {
          setItems(data as AvatarListItem[])
        }
      } catch (e) {
        if (!cancelled) {
          setErr(String(e))
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [userId])

  const del = async (id: string) => {
    if (!me.trim()) {
      setErr('Укажите свой User ID на странице загрузки (хранится в браузере)')
      return
    }
    setBusyId(id)
    setErr(null)
    try {
      const res = await fetch(`/api/v1/avatars/${encodeURIComponent(id)}`, {
        method: 'DELETE',
        headers: { 'X-User-ID': me.trim() },
      })
      if (!res.ok) {
        setErr(await res.text())
        return
      }
      setItems((prev) => (prev ? prev.filter((x) => x.id !== id) : prev))
    } catch (e) {
      setErr(String(e))
    } finally {
      setBusyId(null)
    }
  }

  const delLatest = async () => {
    if (!me.trim()) {
      setErr('Укажите свой User ID на странице загрузки')
      return
    }
    setBusyId('latest')
    setErr(null)
    try {
      const res = await fetch(`/api/v1/users/${encodeURIComponent(userId)}/avatar`, {
        method: 'DELETE',
        headers: { 'X-User-ID': me.trim() },
      })
      if (!res.ok) {
        setErr(await res.text())
        return
      }
      setItems((prev) => (prev && prev.length > 0 ? prev.slice(1) : prev))
    } catch (e) {
      setErr(String(e))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className="page page--gray">
      <main className="page__main container">
        <section className="gallery">
          <h1 className="gallery__title">Галерея: {userId}</h1>

          <div className="gallery__me form__row">
            <label className="form__label" htmlFor="gallery-me">
              Ваш User ID для удаления (X-User-ID)
            </label>
            <input
              id="gallery-me"
              className="form__input gallery__me-input"
              value={me}
              onChange={(e) => {
                const v = e.target.value
                setMe(v)
                localStorage.setItem(USER_STORAGE_KEY, v)
              }}
              autoComplete="username"
            />
          </div>

          <p className="gallery__toolbar">
            <button
              className="button"
              type="button"
              onClick={delLatest}
              disabled={busyId !== null}
            >
              Удалить последний аватар
            </button>
          </p>

          {err && (
            <pre className="gallery__error" role="alert">
              {err}
            </pre>
          )}

          {!items && !err && <p className="gallery__loading">Загрузка…</p>}
          {items && items.length === 0 && <p className="gallery__empty">Нет аватаров</p>}
          {items && items.length > 0 && (
            <div className="gallery__grid">
              {items.map((it) => (
                <AvatarCard
                  key={it.id}
                  id={it.id}
                  busy={busyId !== null}
                  onDelete={() => del(it.id)}
                />
              ))}
            </div>
          )}
        </section>
      </main>
    </div>
  )
}
