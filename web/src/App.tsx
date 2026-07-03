import { useCallback, useEffect, useState } from 'react'
import './App.css'

type Me = {
  steamId: string
  personaName: string
  avatarUrl: string
  lastSyncAt: string | null
  libraryCount: number
}

type Phase =
  | { kind: 'loading' }
  | { kind: 'anon' }
  | { kind: 'ready'; me: Me; syncError: string | null; syncing: boolean }

export default function App() {
  const [phase, setPhase] = useState<Phase>({ kind: 'loading' })

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
    <main className="crt">
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
            <button className="btn" onClick={sync} disabled={phase.syncing}>
              {phase.syncing ? 'SYNCING…' : '⟳ RESYNC LIBRARY'}
            </button>
            <button className="btn dim" onClick={logout}>
              EJECT
            </button>
          </div>

          <p className="dim">MOOD QUESTIONNAIRE COMING ONLINE SOON…</p>
        </section>
      )}
    </main>
  )
}
