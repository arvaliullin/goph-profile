import { Link } from 'react-router-dom'
import { AppRoute } from '../../consts'
import './not-found.css'

export default function NotFoundPage() {
  return (
    <div className="page page--gray">
      <main className="page__main container">
        <section className="not-found">
          <h1 className="not-found__title">404</h1>
          <p className="not-found__text">Страница не найдена.</p>
          <Link className="button button--primary not-found__link" to={AppRoute.Upload}>
            На загрузку
          </Link>
        </section>
      </main>
    </div>
  )
}
