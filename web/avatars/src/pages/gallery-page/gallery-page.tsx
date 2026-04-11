import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import AvatarCard from '../../components/avatar-card/avatar-card'
import { AppRoute } from '../../consts'
import { useAppSelector } from '../../store/hooks'
import type { AvatarListItem } from '../../types/avatar'
import './gallery-page.css'

export default function GalleryPage() {
  const { userId: routeUserId } = useParams<{ userId?: string }>()
  const sessionUserId = useAppSelector((s) => s.session.userId)

  const isMyGalleryRoute = routeUserId === undefined
  const displayUserId = isMyGalleryRoute
    ? (sessionUserId?.trim() ?? '')
    : (routeUserId ?? '')

  const canDelete =
    Boolean(sessionUserId?.trim()) &&
    Boolean(displayUserId) &&
    sessionUserId!.trim() === displayUserId

  const [items, setItems] = useState<AvatarListItem[] | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  useEffect(() => {
    if (isMyGalleryRoute && !sessionUserId?.trim()) {
      setItems(null)
      setErr(null)
      return
    }
    if (!displayUserId) {
      setItems(null)
      setErr(null)
      return
    }

    let cancelled = false
    ;(async () => {
      setErr(null)
      setItems(null)
      try {
        const res = await fetch(`/api/v1/users/${encodeURIComponent(displayUserId)}/avatars`)
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
  }, [displayUserId, isMyGalleryRoute, sessionUserId])

  const del = async (id: string) => {
    const me = sessionUserId?.trim()
    if (!me) {
      setErr('Войдите в шапке, чтобы удалять аватары')
      return
    }
    setBusyId(id)
    setErr(null)
    try {
      const res = await fetch(`/api/v1/avatars/${encodeURIComponent(id)}`, {
        method: 'DELETE',
        headers: { 'X-User-ID': me },
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
    const me = sessionUserId?.trim()
    if (!me) {
      setErr('Войдите в шапке, чтобы удалять аватары')
      return
    }
    setBusyId('latest')
    setErr(null)
    try {
      const res = await fetch(`/api/v1/users/${encodeURIComponent(displayUserId)}/avatar`, {
        method: 'DELETE',
        headers: { 'X-User-ID': me },
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

  if (isMyGalleryRoute && !sessionUserId?.trim()) {
    return (
      <div className="page page--gray">
        <main className="page__main container">
          <section className="gallery">
            <h1 className="gallery__title">Моя галерея</h1>
            <p className="gallery__auth-hint">
              Войдите по идентификатору в шапке, чтобы открыть свою галерею.
            </p>
            <p>
              <Link className="gallery__link-home" to={AppRoute.Root}>
                На главную
              </Link>
            </p>
          </section>
        </main>
      </div>
    )
  }

  return (
    <div className="page page--gray">
      <main className="page__main container">
        <section className="gallery">
          <h1 className="gallery__title">Галерея: {displayUserId}</h1>

          {canDelete && (
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
          )}

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
                  onDelete={canDelete ? () => del(it.id) : undefined}
                />
              ))}
            </div>
          )}
        </section>
      </main>
    </div>
  )
}
