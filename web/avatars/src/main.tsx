import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Provider } from 'react-redux'
import App from './components/app/app'
import { hydrate } from './store/slices/session-slice'
import { store } from './store'
import { loadUserId } from './utils/user-id-storage'
import './index.css'

const saved = loadUserId().trim()
store.dispatch(hydrate(saved || null))

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Provider store={store}>
      <App />
    </Provider>
  </StrictMode>,
)
