// Synthesized arcade juice — WebAudio beeps and a screen shake, no asset
// files. The AudioContext is created lazily on the first user gesture so
// autoplay policies never block it.

let ctx: AudioContext | null = null
let on = localStorage.getItem('ggs_snd') !== 'off'

export const sndOn = () => on
export const toggleSnd = () => {
  on = !on
  localStorage.setItem('ggs_snd', on ? 'on' : 'off')
  return on
}

const ac = () => {
  ctx ??= new AudioContext()
  if (ctx.state === 'suspended') void ctx.resume()
  return ctx
}

const beep = (
  freq: number,
  dur: number,
  opts: { type?: OscillatorType; gain?: number; at?: number; slide?: number } = {},
) => {
  if (!on) return
  const { type = 'square', gain = 0.04, at = 0, slide } = opts
  const c = ac()
  const t = c.currentTime + at
  const osc = c.createOscillator()
  const g = c.createGain()
  osc.type = type
  osc.frequency.setValueAtTime(freq, t)
  if (slide) osc.frequency.exponentialRampToValueAtTime(slide, t + dur)
  g.gain.setValueAtTime(gain, t)
  g.gain.exponentialRampToValueAtTime(0.0001, t + dur)
  osc.connect(g).connect(c.destination)
  osc.start(t)
  osc.stop(t + dur + 0.01)
}

// C-major pentatonic — answering the questionnaire walks up the scale, so a
// filled mood form is literally a little melody.
const scale = [523.25, 587.33, 659.25, 783.99, 880]
export const select = (step: number) => beep(scale[Math.min(step, scale.length - 1)], 0.09)
export const deselect = () => beep(329.63, 0.07)

export const click = () => beep(659.25, 0.04, { gain: 0.03 })
export const coin = () => {
  beep(987.77, 0.08)
  beep(1318.51, 0.35, { at: 0.08, type: 'triangle', gain: 0.05 })
}
export const tick = () => beep(1800 + Math.random() * 400, 0.02, { gain: 0.015 })
export const reveal = () =>
  [523.25, 659.25, 783.99, 1046.5].forEach((f, i) => beep(f, 0.12, { at: i * 0.07, gain: 0.05 }))
export const denied = () => beep(220, 0.3, { type: 'sawtooth', gain: 0.05, slide: 98 })

export const reducedMotion = () =>
  window.matchMedia('(prefers-reduced-motion: reduce)').matches

// Whacks the cabinet: brief shake (plus a degauss color-wobble while the CRT
// effect is on — see App.css). Class lives on <main class="screen">.
export const shakeScreen = (hard = false) => {
  if (reducedMotion()) return
  const el = document.querySelector('.screen')
  if (!el) return
  el.classList.remove('shaking', 'shaking-hard')
  void (el as HTMLElement).offsetWidth // restart the animation
  el.classList.add(hard ? 'shaking-hard' : 'shaking')
  window.setTimeout(() => el.classList.remove('shaking', 'shaking-hard'), 600)
}
