import { useEffect, useId, useState } from 'react'
import { useAppDispatch } from '../../store/hooks'
import { login } from '../../store/slices/session-slice'
import './login-modal.css'

type LoginModalProps = {
  open: boolean
  onClose: () => void
}

export default function LoginModal({ open, onClose }: LoginModalProps) {
  const dispatch = useAppDispatch()
  const titleId = useId()
  const inputId = useId()
  const [value, setValue] = useState('')

  useEffect(() => {
    if (open) {
      setValue('')
    }
  }, [open])

  useEffect(() => {
    if (!open) {
      return
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) {
    return null
  }

  const submit = () => {
    const id = value.trim()
    if (!id) {
      return
    }
    dispatch(login(id))
    onClose()
  }

  return (
    <div
      className="login-modal"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
    >
      <button
        type="button"
        className="login-modal__backdrop"
        aria-label="Закрыть"
        onClick={onClose}
      />
      <div className="login-modal__card">
        <h2 className="login-modal__title" id={titleId}>
          Вход по идентификатору
        </h2>
        <p className="login-modal__hint">
          Укажите тот же user id, что используется в заголовке{' '}
          <code className="login-modal__code">X-User-ID</code> при запросах к API.
        </p>
        <label className="form__label" htmlFor={inputId}>
          User ID
        </label>
        <input
          id={inputId}
          className="form__input login-modal__input"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              submit()
            }
          }}
          placeholder="например, demo-user"
          autoComplete="username"
          autoFocus
        />
        <div className="login-modal__actions">
          <button type="button" className="button" onClick={onClose}>
            Отмена
          </button>
          <button type="button" className="button button--primary" onClick={submit}>
            Продолжить
          </button>
        </div>
      </div>
    </div>
  )
}
