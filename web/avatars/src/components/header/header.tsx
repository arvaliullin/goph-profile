import { Link } from 'react-router-dom'
import { AppRoute } from '../../consts'
import { loadUserId } from '../../utils/user-id-storage'
import './header.css'

export default function Header() {
  const storedId = loadUserId().trim()
  const galleryPath = storedId ? `/web/gallery/${encodeURIComponent(storedId)}` : null

  return (
    <header className="header">
      <div className="container header__inner">
        <Link className="header__brand" to={AppRoute.Root}>
          GophProfile · Avatars
        </Link>
        <nav className="header__nav" aria-label="Основная навигация">
          <Link className="header__link" to={AppRoute.Root}>
            Главная
          </Link>
          <Link className="header__link" to={AppRoute.Upload}>
            Загрузка
          </Link>
          {galleryPath ? (
            <Link className="header__link" to={galleryPath}>
              Моя галерея
            </Link>
          ) : (
            <span className="header__hint" title="Укажите идентификатор на странице загрузки">
              Галерея: /web/gallery/&lt;user_id&gt;
            </span>
          )}
        </nav>
      </div>
    </header>
  )
}
