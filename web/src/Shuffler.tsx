import { useEffect, useState } from 'react'

export type Mood = {
  energy: string
  time: string
  familiarity: string
  brain?: string
}

export type ShuffleResult = {
  appId: number
  name: string
  why: string
  usedAi: boolean
  shufflesLeft: number
  relaxed?: string[]
}

type Progress = { enriched: number; total: number }

const QUESTIONS: {
  key: keyof Mood
  label: string
  optional?: boolean
  options: { value: string; label: string }[]
}[] = [
  {
    key: 'energy',
    label: 'ENERGY LEVEL',
    options: [
      { value: 'chill', label: '~ CHILL' },
      { value: 'balanced', label: '= BALANCED' },
      { value: 'adrenaline', label: '! ADRENALINE' },
    ],
  },
  {
    key: 'time',
    label: 'TIME IN HAND',
    options: [
      { value: 'quick', label: 'QUICK ROUND' },
      { value: 'medium', label: 'A FEW HOURS' },
      { value: 'long', label: 'LOST WEEKEND' },
    ],
  },
  {
    key: 'familiarity',
    label: 'FAMILIARITY',
    options: [
      { value: 'favorite', label: 'OLD FAVORITE' },
      { value: 'backlog', label: 'BACKLOG SHAME' },
      { value: 'surprise', label: 'SURPRISE ME' },
    ],
  },
  {
    key: 'brain',
    label: 'BRAIN MODE (OPTIONAL)',
    optional: true,
    options: [
      { value: 'story', label: 'STORY' },
      { value: 'puzzle', label: 'PUZZLE' },
      { value: 'reflex', label: 'REFLEXES' },
    ],
  },
]

export function Shuffler(props: {
  shufflesLeft: number
  resetAt: string
  onSpent: (left: number) => void
}) {
  const [mood, setMood] = useState<Partial<Mood>>({})
  const [result, setResult] = useState<ShuffleResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [progress, setProgress] = useState<Progress | null>(null)

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>
    let stop = false
    const poll = async () => {
      const res = await fetch('/api/library/status')
      if (!res.ok) return
      const p: Progress = await res.json()
      setProgress(p)
      if (!stop && p.enriched < p.total) timer = setTimeout(poll, 5000)
    }
    poll().catch(() => {})
    return () => {
      stop = true
      clearTimeout(timer)
    }
  }, [])

  const answered = QUESTIONS.filter((q) => !q.optional).every((q) => mood[q.key])
  const left = props.shufflesLeft

  const shuffle = async () => {
    setBusy(true)
    setError(null)
    const res = await fetch('/api/shuffle', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(mood),
    })
    setBusy(false)
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      setError(body.message ?? 'Shuffle failed.')
      if (res.status === 429) props.onSpent(0)
      return
    }
    const r: ShuffleResult = await res.json()
    setResult(r)
    props.onSpent(r.shufflesLeft)
  }

  const resetLocal = new Date(props.resetAt).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
  })

  return (
    <section className="panel">
      {progress && progress.enriched < progress.total && (
        <p className="dim blink">
          CATALOGING: {progress.enriched}/{progress.total}
        </p>
      )}

      {result ? (
        <div className="result">
          <img
            className="header-img"
            src={`https://cdn.cloudflare.steamstatic.com/steam/apps/${result.appId}/header.jpg`}
            alt={result.name}
          />
          <h2 className="game-name">{result.name}</h2>
          <p className="why">&gt; {result.why}</p>
          <div className="actions">
            <a
              className="btn"
              href={`steam://run/${result.appId}`}
            >
              ▶ PLAY
            </a>
            <button
              className="btn"
              onClick={shuffle}
              disabled={busy || left === 0}
            >
              ⟳ SHUFFLE AGAIN ({left})
            </button>
            <button className="btn dim" onClick={() => setResult(null)}>
              CHANGE MOOD
            </button>
          </div>
        </div>
      ) : (
        <>
          {QUESTIONS.map((q) => (
            <fieldset className="question" key={q.key}>
              <legend>{q.label}</legend>
              <div className="options">
                {q.options.map((o) => (
                  <button
                    key={o.value}
                    className={mood[q.key] === o.value ? 'btn selected' : 'btn'}
                    onClick={() =>
                      setMood((m) => ({
                        ...m,
                        [q.key]: m[q.key] === o.value ? undefined : o.value,
                      }))
                    }
                  >
                    {o.label}
                  </button>
                ))}
              </div>
            </fieldset>
          ))}
          <button
            className="btn go"
            onClick={shuffle}
            disabled={!answered || busy || left === 0}
          >
            {busy ? 'SHUFFLING…' : `◆ SHUFFLE (${left} LEFT TODAY)`}
          </button>
        </>
      )}

      {error && <p className="error">! {error}</p>}
      {left === 0 && (
        <p className="dim">OUT OF SHUFFLES — MORE AT {resetLocal} (UTC MIDNIGHT)</p>
      )}
    </section>
  )
}
