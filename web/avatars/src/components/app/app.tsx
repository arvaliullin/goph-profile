import { BrowserRouter, Navigate, Route, Routes, useParams } from 'react-router-dom'
import { AppRoute } from '../../consts'
import GalleryPage from '../../pages/gallery-page/gallery-page'
import HomePage from '../../pages/home-page/home-page'
import NotFoundPage from '../../pages/not-found/not-found'
import UploadPage from '../../pages/upload-page/upload-page'
import Header from '../header/header'
import './app.css'

function LegacyGalleryRedirect() {
  const { userId } = useParams()
  return <Navigate to={`/web/gallery/${userId ?? ''}`} replace />
}

export default function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <Header />
        <Routes>
          <Route path={AppRoute.Root} element={<HomePage />} />
          <Route path={AppRoute.Upload} element={<UploadPage />} />
          <Route path={AppRoute.Gallery} element={<GalleryPage />} />
          <Route path="/upload" element={<Navigate to="/web/upload" replace />} />
          <Route path="/gallery/:userId" element={<LegacyGalleryRedirect />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </div>
    </BrowserRouter>
  )
}
