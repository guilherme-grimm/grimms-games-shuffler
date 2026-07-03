import { useCallback, useEffect, useState } from 'react'
import { Shuffler } from './Shuffler'
import './App.css'

type Me = {
  steamId: string
  personaName: string
  avatarUrl: string
  lastSyncAt: string | null
  libraryCount: number
  shufflesLeft: number
  resetAt: string
  aiAvailable: boolean
}

type Phase =
  | { kind: 'loading' }
  | { kind: 'anon' }
  | { kind: 'ready'; me: Me; syncError: string | null; syncing: boolean }

export default function App() {
  const [phase, setPhase] = useState<Phase>({ kind: 'loading' })
  const [crt, setCrt] = useState(() => localStorage.getItem('ggs_crt') !== 'off')

  const toggleCrt = () => {
    setCrt((on) => {
      localStorage.setItem('ggs_crt', on ? 'off' : 'on')
      return !on
    })
  }

  const loadMe = useCallback(async () => {
    const res = await fetch('/api/me')
    if (res.status === 401) {
      setPhase({ kind: 'anon' })
      return
    }
    if (!res.ok) throw new Error(`me: ${res.status}`)
    const me: Me = await res.json()
    setPhase({ kind: 'ready', me, syncError: null, syncing: false })
  }, [])

  useEffect(() => {
    loadMe().catch(() => setPhase({ kind: 'anon' }))
  }, [loadMe])

  const sync = async () => {
    setPhase((p) => (p.kind === 'ready' ? { ...p, syncing: true, syncError: null } : p))
    const res = await fetch('/api/sync', { method: 'POST' })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ message: 'Sync failed.' }))
      setPhase((p) =>
        p.kind === 'ready'
          ? { ...p, syncing: false, syncError: body.message ?? 'Sync failed.' }
          : p,
      )
      return
    }
    await loadMe()
  }

  const logout = async () => {
    await fetch('/auth/logout', { method: 'POST' })
    setPhase({ kind: 'anon' })
  }

  return (
    <main className={crt ? 'screen crt-fx' : 'screen'}>
      <button
        className="crt-toggle"
        onClick={toggleCrt}
        title="CRT effect can be tiring on the eyes — toggle it off anytime"
      >
        CRT: {crt ? 'ON' : 'OFF'}
      </button>
      <h1 className="title">GGS :: GRIMM'S GAMES SHUFFLER</h1>

      {phase.kind === 'loading' && <p className="blink">BOOTING…</p>}

      {phase.kind === 'anon' && (
        <section className="panel">
          <p>INSERT COIN TO CONTINUE</p>
          <a className="btn" href="/auth/steam/login">
            ▶ SIGN IN THROUGH STEAM
          </a>
        </section>
      )}

      {phase.kind === 'ready' && (
        <section className="panel">
          <header className="player-row">
            <img className="avatar" src={phase.me.avatarUrl} alt="" />
            <div>
              <p className="persona">{phase.me.personaName}</p>
              <p className="dim">
                LIBRARY: {phase.me.libraryCount} GAMES
                {phase.me.lastSyncAt &&
                  ` · SYNCED ${new Date(phase.me.lastSyncAt).toLocaleString()}`}
              </p>
            </div>
          </header>

          {phase.syncError && <p className="error">! {phase.syncError}</p>}

          <div className="actions">
            <button className="btn dim" onClick={sync} disabled={phase.syncing}>
              {phase.syncing ? 'SYNCING…' : '⟳ RESYNC'}
            </button>
            <button className="btn dim" onClick={logout}>
              EJECT
            </button>
          </div>
        </section>
      )}

      {phase.kind === 'ready' && phase.me.libraryCount > 0 && (
        <Shuffler
          shufflesLeft={phase.me.shufflesLeft}
          resetAt={phase.me.resetAt}
          aiAvailable={phase.me.aiAvailable}
          onSpent={(shufflesLeft) =>
            setPhase((p) =>
              p.kind === 'ready' ? { ...p, me: { ...p.me, shufflesLeft } } : p,
            )
          }
        />
      )}
    </main>
  )
}
