import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AppRoute } from '../../consts'
import { loadUserId } from '../../utils/user-id-storage'
import './home-page.css'

export default function HomePage() {
  const navigate = useNavigate()
  const storedId = loadUserId().trim()
  const [galleryUserId, setGalleryUserId] = useState(storedId)

  const openGallery = () => {
    const id = galleryUserId.trim()
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

            {storedId ? (
              <Link
                className="home__card"
                to={`/web/gallery/${encodeURIComponent(storedId)}`}
              >
                <span className="home__card-title">Моя галерея</span>
                <span className="home__card-desc">Пользователь «{storedId}»</span>
              </Link>
            ) : (
              <div className="home__card home__card--muted">
                <span className="home__card-title">Моя галерея</span>
                <span className="home__card-desc">
                  Сохраните идентификатор на странице загрузки
                </span>
              </div>
            )}
          </nav>

          <div className="home__gallery-open">
            <label className="form__label" htmlFor="home-gallery-user">
              Открыть галерею по user id
            </label>
            <div className="home__gallery-row">
              <input
                id="home-gallery-user"
                className="form__input home__gallery-input"
                value={galleryUserId}
                onChange={(e) => setGalleryUserId(e.target.value)}
                placeholder="например, demo-user"
                autoComplete="username"
              />
              <button className="button button--primary" type="button" onClick={openGallery}>
                Перейти
              </button>
            </div>
          </div>
        </section>
      </main>
    </div>
  )
}
