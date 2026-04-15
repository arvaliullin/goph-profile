import { useState } from 'react'
import { Link } from 'react-router-dom'
import { AppRoute } from '../../consts'
import LoginModal from '../login-modal/login-modal'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { logout } from '../../store/slices/session-slice'
import './header.css'

function truncateId(s: string, max = 28) {
  if (s.length <= max) {
    return s
  }
  return `${s.slice(0, max - 1)}…`
}

export default function Header() {
  const dispatch = useAppDispatch()
  const userId = useAppSelector((s) => s.session.userId)?.trim() ?? ''
  const [loginOpen, setLoginOpen] = useState(false)

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
          {userId ? (
            <Link className="header__link" to={AppRoute.MyGallery}>
              Моя галерея
            </Link>
          ) : (
            <span className="header__hint" title="Войдите, чтобы открыть свою галерею без ввода id">
              Моя галерея - после входа
            </span>
          )}
        </nav>
        <div className="header__session">
          {userId ? (
            <>
              <span className="header__badge" title={userId}>
                {truncateId(userId)}
              </span>
              <button
                type="button"
                className="header__btn header__btn--ghost"
                onClick={() => dispatch(logout())}
              >
                Выйти
              </button>
            </>
          ) : (
            <button
              type="button"
              className="header__btn header__btn--primary"
              onClick={() => setLoginOpen(true)}
            >
              Войти
            </button>
          )}
        </div>
      </div>
      <LoginModal open={loginOpen} onClose={() => setLoginOpen(false)} />
    </header>
  )
}
