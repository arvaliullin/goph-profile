import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AppRoute } from '../../consts'
import { useAppSelector } from '../../store/hooks'
import './home-page.css'

export default function HomePage() {
  const navigate = useNavigate()
  const sessionUserId = useAppSelector((s) => s.session.userId)?.trim() ?? ''
  const [otherGalleryId, setOtherGalleryId] = useState('')

  const openOtherGallery = () => {
    const id = otherGalleryId.trim()
    if (!id) {
      return
    }
    navigate(`/web/gallery/${encodeURIComponent(id)}`)
  }

  return (
    <div className="page page--gray">
      <main className="page__main container">
        <section className="home">
          <h1 className="home__title">GophProfile · Аватары</h1>
          <p className="home__lead">
            Загрузка изображений и просмотр галереи по идентификатору пользователя.
          </p>

          <nav className="home__nav" aria-label="Разделы">
            <Link className="home__card" to={AppRoute.Upload}>
              <span className="home__card-title">Загрузка</span>
              <span className="home__card-desc">Отправить новый аватар</span>
            </Link>

            {sessionUserId ? (
              <Link className="home__card" to={AppRoute.MyGallery}>
                <span className="home__card-title">Моя галерея</span>
                <span className="home__card-desc">Пользователь «{sessionUserId}»</span>
              </Link>
            ) : (
              <div className="home__card home__card--muted">
                <span className="home__card-title">Моя галерея</span>
                <span className="home__card-desc">Войдите по идентификатору в шапке</span>
              </div>
            )}
          </nav>

          <details className="home__details">
            <summary className="home__details-summary">Открыть чужую галерею</summary>
            <div className="home__details-body">
              <label className="form__label" htmlFor="home-gallery-user">
                User id
              </label>
              <div className="home__gallery-row">
                <input
                  id="home-gallery-user"
                  className="form__input home__gallery-input"
                  value={otherGalleryId}
                  onChange={(e) => setOtherGalleryId(e.target.value)}
                  placeholder="например, demo-user"
                  autoComplete="username"
                />
                <button className="button button--primary" type="button" onClick={openOtherGallery}>
                  Перейти
                </button>
              </div>
            </div>
          </details>
        </section>
      </main>
    </div>
  )
}
