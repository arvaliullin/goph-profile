import './avatar-card.css'

type AvatarCardProps = {
  id: string
  busy: boolean
  onDelete: () => void
}

export default function AvatarCard({ id, busy, onDelete }: AvatarCardProps) {
  const src = `/api/v1/avatars/${encodeURIComponent(id)}?size=300x300`

  return (
    <article className="avatar-card">
      <div className="avatar-card__id" title={id}>
        {id}
      </div>
      <div className="avatar-card__preview">
        <img className="avatar-card__img" src={src} alt="" width={300} height={300} />
      </div>
      <button
        className="button avatar-card__delete"
        type="button"
        disabled={busy}
        onClick={onDelete}
      >
        Удалить
      </button>
    </article>
  )
}
