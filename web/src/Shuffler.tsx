import { useEffect, useState } from 'react'
import * as sfx from './sfx'

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

const GLYPHS = '▓▒░█<>/[]{}=+*#%&$!?0123456789'
const glyphs = (n: number) =>
  Array.from({ length: n }, () => GLYPHS[Math.floor(Math.random() * GLYPHS.length)]).join('')

// The reel spins at least this long so the pick feels drawn, not fetched.
const MIN_ROLL_MS = 1400

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

/** Slot-machine reel: three rows of churning glyphs, ticking as they spin. */
function Reel() {
  const [rows, setRows] = useState(() => [glyphs(22), glyphs(22), glyphs(22)])

  useEffect(() => {
    if (sfx.reducedMotion()) return
    const timer = setInterval(() => {
      setRows([glyphs(22), glyphs(22), glyphs(22)])
      sfx.tick()
    }, 70)
    return () => clearInterval(timer)
  }, [])

  return (
    <div className="reel" role="status" aria-label="Shuffling">
      <p className="reel-row dim">{rows[0]}</p>
      <p className="reel-row reel-mid">{rows[1]}</p>
      <p className="reel-row dim">{rows[2]}</p>
      <p className="blink reel-label">◆ SHUFFLING</p>
    </div>
  )
}

/** Decodes text from scrambled glyphs, left to right. */
function Decode({ text }: { text: string }) {
  const [shown, setShown] = useState(() => (sfx.reducedMotion() ? text : glyphs(text.length)))

  useEffect(() => {
    if (sfx.reducedMotion()) return
    let i = 0
    const timer = setInterval(() => {
      i += 1 + Math.floor(text.length / 18)
      if (i >= text.length) {
        setShown(text)
        clearInterval(timer)
        return
      }
      setShown(text.slice(0, i) + glyphs(Math.min(5, text.length - i)))
    }, 28)
    return () => clearInterval(timer)
  }, [text])

  return <>{shown}</>
}

export function Shuffler(props: {
  shufflesLeft: number
  resetAt: string
  aiAvailable: boolean
  onSpent: (left: number) => void
}) {
  const [mood, setMood] = useState<Partial<Mood>>({})
  const [result, setResult] = useState<ShuffleResult | null>(null)
  const [rolling, setRolling] = useState(false)
  const [aiAsked, setAiAsked] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [progress, setProgress] = useState<Progress | null>(null)
  const [useAi, setUseAi] = useState(() => localStorage.getItem('ggs_ai') !== 'off')
  const [note, setNote] = useState('')

  const toggleAi = () => {
    sfx.click()
    setUseAi((on) => {
      localStorage.setItem('ggs_ai', on ? 'off' : 'on')
      return !on
    })
  }

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

  const pick = (key: keyof Mood, value: string) => {
    const deselecting = mood[key] === value
    if (deselecting) {
      sfx.deselect()
    } else {
      sfx.select(Object.values(mood).filter(Boolean).length)
    }
    setMood((m) => ({ ...m, [key]: deselecting ? undefined : value }))
  }

  const shuffle = async () => {
    setError(null)
    setResult(null)
    setRolling(true)
    sfx.coin()
    const ai = props.aiAvailable && useAi
    const started = Date.now()

    let res: Response
    try {
      res = await fetch('/api/shuffle', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...mood,
          useAi: ai,
          note: ai && note.trim() ? note.trim() : undefined,
        }),
      })
    } catch {
      setRolling(false)
      sfx.denied()
      sfx.shakeScreen()
      setError('Connection lost. Check the cabinet cables and retry.')
      return
    }
    await sleep(Math.max(0, MIN_ROLL_MS - (Date.now() - started)))
    setRolling(false)

    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      sfx.denied()
      sfx.shakeScreen()
      setError(body.message ?? 'Shuffle failed.')
      if (res.status === 429) props.onSpent(0)
      return
    }
    const r: ShuffleResult = await res.json()
    sfx.reveal()
    sfx.shakeScreen(true)
    setAiAsked(ai)
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

      {rolling ? (
        <Reel />
      ) : result ? (
        <div className="result">
          <img
            className="header-img"
            src={`https://cdn.cloudflare.steamstatic.com/steam/apps/${result.appId}/header.jpg`}
            alt={result.name}
          />
          <h2 className="game-name">
            <Decode text={result.name} />
            {result.usedAi && <span className="ai-badge"> [AI PICK]</span>}
          </h2>
          <p className="why">&gt; {result.why}</p>
          {aiAsked && !result.usedAi && (
            <p className="dim">AI WAS BUSY — ARCADE BRAIN TOOK OVER.</p>
          )}
          <div className="actions">
            <a
              className="btn accent"
              href={`steam://run/${result.appId}`}
              onClick={() => sfx.coin()}
            >
              ▶ PLAY
            </a>
            <button className="btn" onClick={shuffle} disabled={left === 0}>
              ⟳ SHUFFLE AGAIN ({left})
            </button>
            <button
              className="btn dim"
              onClick={() => {
                sfx.click()
                setResult(null)
              }}
            >
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
                    onClick={() => pick(q.key, o.value)}
                  >
                    {o.label}
                  </button>
                ))}
              </div>
            </fieldset>
          ))}
          {props.aiAvailable && (
            <fieldset className="question">
              <legend>AI CO-PILOT</legend>
              <div className="options ai-row">
                <button
                  className={useAi ? 'btn selected' : 'btn'}
                  onClick={toggleAi}
                >
                  AI: {useAi ? 'ON' : 'OFF'}
                </button>
                {useAi && (
                  <input
                    className="note-input"
                    type="text"
                    maxLength={200}
                    placeholder="MOOD NOTE (OPTIONAL)…"
                    value={note}
                    onChange={(e) => setNote(e.target.value)}
                  />
                )}
              </div>
            </fieldset>
          )}
          <button
            className="btn go"
            onClick={shuffle}
            disabled={!answered || left === 0}
          >
            ◆ SHUFFLE ({left} LEFT TODAY)
          </button>
        </>
      )}

      {error && <p className="error">! {error}</p>}
      {left === 0 && !rolling && (
        <p className="dim">OUT OF SHUFFLES — MORE AT {resetLocal} (UTC MIDNIGHT)</p>
      )}
    </section>
  )
}
