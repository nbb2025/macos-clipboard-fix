import { useEffect, useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Events } from '@wailsio/runtime'
import { ClipboardService as API } from '../bindings/github.com/nbb2025/macos-clipboard-fix'
import {
  ArrowDownLeft,
  ArrowLeftRight,
  ArrowRight,
  ArrowUpRight,
  Bluetooth,
  Check,
  CheckCheck,
  CheckCircle2,
  ChevronRight,
  CircleHelp,
  ClipboardCheck,
  Copy,
  Download,
  FileClock,
  FolderLock,
  History,
  Info,
  Laptop,
  Link2,
  LoaderCircle,
  RadioTower,
  RefreshCw,
  RotateCcw,
  Settings2,
  ShieldCheck,
  Smartphone,
  Sparkles,
  Terminal,
  Wifi,
  Wrench,
  X,
  XCircle,
  type LucideIcon,
} from 'lucide-react'
import type { Cache, Check as CheckType, Probe, Progress, Report, Session, Signal } from './types'
import './style.css'

type Page = 'overview' | 'tools' | 'verify' | 'activity'
type Confirmation = {
  title: string
  detail: string
  notes: string[]
  label: string
  run: () => Promise<unknown>
}
const emptySession: Session = { history: [], backups: [], verifications: [] }
const normalizeReport = (r: Awaited<ReturnType<typeof API.Diagnose>>): Report => ({
  ...r,
  checks: r.checks ?? [],
  services: r.services ?? [],
  caches: r.caches ?? [],
  tools: r.tools ?? [],
  findings: r.findings ?? [],
})
const normalizeSession = (s: Awaited<ReturnType<typeof API.GetSession>>): Session => ({
  history: s.history ?? [],
  backups: s.backups ?? [],
  verifications: s.verifications ?? [],
})

const nav: { id: Page; label: string; icon: LucideIcon }[] = [
  { id: 'overview', label: '连接概览', icon: Laptop },
  { id: 'tools', label: '修复工具', icon: Wrench },
  { id: 'verify', label: '跨设备复测', icon: ArrowLeftRight },
  { id: 'activity', label: '活动记录', icon: History },
]
const actions: Record<string, { title: string; detail: string; notes: string[] }> = {
  quick: {
    title: '温和修复',
    detail: '启用共享剪贴板，并刷新接力与本机剪贴板服务。',
    notes: ['当前 Mac 剪贴板会清空，请稍后重新复制。', '完成后，需要在另一台设备实际粘贴确认。'],
  },
  'cache-send': {
    title: '重建发送缓存',
    detail: '适合手机能复制到 Mac，但 Mac 的新内容传不出去的情况。',
    notes: [
      '先保留旧缓存为同目录备份，再重新加载接力服务。',
      'macOS 可能请求允许本程序控制访达，以备份这一份缓存。',
      '新缓存会在后续共享时生成，完成后请用新文字复测。',
    ],
  },
  'cache-receive': {
    title: '重建接收缓存',
    detail: '适合 Mac 一直粘出旧内容，而手机或 iPad 已复制新内容的情况。',
    notes: [
      '保留接收缓存的可恢复备份，再重新加载接力服务。',
      'macOS 可能请求允许本程序控制访达。',
      '完成后请在手机重新复制，再到 Mac 粘贴。',
    ],
  },
  discovery: {
    title: '刷新设备发现',
    detail: '重新启动设备发现与连接服务，让附近设备重新建立联系。',
    notes: [
      '隔空投送和其他连续互通功能可能短暂重连。',
      '设备之间重新交换接力密钥可能需要一段时间。',
    ],
  },
  identity: {
    title: '刷新设备身份',
    detail: '重新启动 Apple 设备身份与本机剪贴板服务。',
    notes: [
      'iMessage、FaceTime 等 Apple 服务可能短暂重连。',
      '当前 Mac 剪贴板会清空，请稍后重新复制。',
    ],
  },
  handoff: {
    title: '重新启用接力',
    detail: '把接力的发送、接收开关关闭后重新开启。',
    notes: [
      '连续互通功能会短暂停用，操作完成后接力保持开启。',
      '可在系统设置中核对开关，手机和 iPad 端也需要开启接力。',
    ],
  },
}
function timeLabel(v: string) {
  return v
    ? new Date(v).toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      })
    : '尚无记录'
}
function bytesLabel(v: number) {
  return v < 1024
    ? `${v} B`
    : v < 1048576
      ? `${(v / 1024).toFixed(1)} KB`
      : `${(v / 1048576).toFixed(1)} MB`
}
function errorText(e: unknown) {
  return e instanceof Error ? e.message : String(e)
}

function DeviceScene() {
  return (
    <svg
      className="device-scene"
      viewBox="0 0 460 260"
      role="img"
      aria-label="Mac、iPhone 和 iPad 之间的剪贴板连接"
    >
      <defs>
        <linearGradient id="screen" x2="1" y2="1">
          <stop stopColor="#e6f3e9" />
          <stop offset="1" stopColor="#f8fbf8" />
        </linearGradient>
        <filter id="shadow">
          <feDropShadow dx="0" dy="8" stdDeviation="8" floodColor="#164b2a" floodOpacity=".08" />
        </filter>
      </defs>
      <ellipse cx="238" cy="225" rx="197" ry="16" fill="#d7e6d9" opacity=".45" />
      <g filter="url(#shadow)">
        <rect
          x="29"
          y="47"
          width="228"
          height="157"
          rx="12"
          fill="#fff"
          stroke="#afc3b4"
          strokeWidth="1.6"
        />
        <rect x="38" y="57" width="210" height="135" rx="5" fill="url(#screen)" />
        <circle cx="143" cy="52" r="2" fill="#bfd0c3" />
        <path d="M13 204H273L263 214H23Z" fill="#fff" stroke="#b7c9bb" />
        <path d="M110 204h68l-4 4h-60Z" fill="#d9e4dc" />
      </g>
      <g transform="translate(92 83) rotate(-5 45 44)" filter="url(#shadow)">
        <rect width="91" height="87" rx="13" fill="white" stroke="#dbe6dc" />
        <rect
          x="22"
          y="17"
          width="47"
          height="52"
          rx="7"
          fill="#e2f2e6"
          stroke="#31835c"
          strokeWidth="2"
        />
        <rect
          x="34"
          y="12"
          width="23"
          height="10"
          rx="4"
          fill="#fff"
          stroke="#31835c"
          strokeWidth="2"
        />
        <path
          d="M33 43l9 9 18-20"
          fill="none"
          stroke="#31835c"
          strokeWidth="3"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </g>
      <path
        d="M244 106C281 106 267 79 302 79M259 163C298 163 277 192 330 192"
        stroke="#75ab87"
        strokeWidth="2"
        strokeDasharray="5 5"
        fill="none"
      />
      <g filter="url(#shadow)">
        <rect
          x="305"
          y="29"
          width="70"
          height="143"
          rx="14"
          fill="#fff"
          stroke="#afc3b4"
          strokeWidth="1.5"
        />
        <rect x="311" y="35" width="58" height="130" rx="10" fill="#ecf5ed" />
        <rect x="329" y="39" width="23" height="5" rx="3" fill="#b4cbb9" />
        <rect x="325" y="76" width="31" height="34" rx="6" fill="#fff" />
        <path
          d="M333 91l6 6 10-13"
          fill="none"
          stroke="#42805c"
          strokeWidth="2.5"
          strokeLinecap="round"
        />
      </g>
      <g filter="url(#shadow)">
        <rect
          x="339"
          y="133"
          width="104"
          height="111"
          rx="12"
          fill="#fff"
          stroke="#afc3b4"
          strokeWidth="1.5"
        />
        <rect x="346" y="140" width="90" height="96" rx="6" fill="#e4f0e7" />
        <path
          d="M368 172h44m-44 9h31m-31 9h37"
          stroke="#9ab7a1"
          strokeWidth="3"
          strokeLinecap="round"
        />
        <rect x="367" y="201" width="45" height="17" rx="5" fill="#fff" />
        <path d="M377 209h23" stroke="#5b926b" strokeWidth="2" strokeLinecap="round" />
      </g>
      <g transform="translate(45 9)">
        <rect width="67" height="29" rx="9" fill="#fff" stroke="#d5e4d7" />
        <text x="33" y="19" textAnchor="middle" fill="#55765e" fontSize="13" fontFamily="system-ui">
          ⌘ C
        </text>
      </g>
      <g transform="translate(201 19)">
        <circle r="13" fill="#2e7553" />
        <path d="M-5 0l3 3 7-7" fill="none" stroke="#fff" strokeWidth="2" strokeLinecap="round" />
      </g>
    </svg>
  )
}
function CheckCard({ check, onOpen }: { check: CheckType; onOpen: () => void }) {
  const icons: Record<string, LucideIcon> = {
    handoff: Link2,
    wifi: Wifi,
    bluetooth: Bluetooth,
    awdl: RadioTower,
  }
  const Icon = icons[check.id] || Info
  return (
    <button className="check-card" onClick={onOpen}>
      <div className="check-card-top">
        <span className="icon-box">
          <Icon size={19} />
        </span>
        <span className={`mini-status ${check.status}`}>
          <i />
          {check.status === 'ok' ? '已就绪' : check.status === 'warning' ? '需检查' : '待核对'}
        </span>
      </div>
      <h3>{check.title}</h3>
      <p>{check.detail}</p>
    </button>
  )
}
function CacheRow({ cache, kind }: { cache: Cache | undefined; kind: 'send' | 'receive' }) {
  const send = kind === 'send'
  return (
    <div className="cache-row">
      <span className="cache-icon">
        {send ? <ArrowUpRight size={19} /> : <ArrowDownLeft size={19} />}
      </span>
      <div>
        <strong>{send ? '发送缓存' : '接收缓存'}</strong>
        <p>{cache?.exists ? `更新于 ${timeLabel(cache.modified)}` : cache?.detail || '等待诊断'}</p>
      </div>
      <span className="cache-size">{cache?.exists ? bytesLabel(cache.bytes) : '—'}</span>
    </div>
  )
}

export default function App() {
  const [page, setPage] = useState<Page>('overview'),
    [report, setReport] = useState<Report | null>(null),
    [session, setSession] = useState<Session>(emptySession)
  const [scanning, setScanning] = useState(false),
    [busy, setBusy] = useState(false),
    [progress, setProgress] = useState<Progress | null>(null)
  const [notice, setNotice] = useState<{ text: string; error?: boolean } | null>(null),
    [confirm, setConfirm] = useState<Confirmation | null>(null)
  const [probe, setProbe] = useState<Probe | null>(null),
    [signals, setSignals] = useState<Signal[]>([]),
    [symptom, setSymptom] = useState('unsure')
  const [guide, setGuide] = useState(false),
    [seconds, setSeconds] = useState<number | null>(null),
    [comparisonBefore, setComparisonBefore] = useState<Report['proxy'] | null>(null)
  const notify = (text: string, error = false) => setNotice({ text, error })
  const refreshSession = async () => setSession(normalizeSession(await API.GetSession()))
  const scan = async () => {
    setScanning(true)
    try {
      setReport(normalizeReport(await API.Diagnose()))
      await refreshSession()
    } catch (e) {
      notify(errorText(e), true)
    } finally {
      setScanning(false)
    }
  }
  useEffect(() => {
    void scan()
    const off = Events.On('clipboard:progress', (e) => setProgress(e.data as Progress))
    return () => {
      off()
    }
  }, [])
  useEffect(() => {
    if (!notice) return
    const t = window.setTimeout(() => setNotice(null), notice.error ? 10000 : 4500)
    return () => clearTimeout(t)
  }, [notice])
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'r') {
        e.preventDefault()
        if (!busy && !scanning) void scan()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [busy, scanning])
  useEffect(() => {
    if (seconds === null) return
    if (seconds <= 0) {
      notify('对照计时结束，请在代理工具中恢复原设置。')
      setSeconds(null)
      void scan()
      return
    }
    const t = setTimeout(() => setSeconds((s) => (s === null ? null : s - 1)), 1000)
    return () => clearTimeout(t)
  }, [seconds])
  const execute = async (run: () => Promise<unknown>, message = '步骤已完成，请跨设备复测') => {
    setConfirm(null)
    setBusy(true)
    setProgress(null)
    setProbe(null)
    try {
      await run()
      notify(message)
      setReport(normalizeReport(await API.Diagnose()))
    } catch (e) {
      notify(errorText(e), true)
    } finally {
      await refreshSession().catch(() => {})
      setBusy(false)
    }
  }
  const repair = (action: string) =>
    setConfirm({
      ...actions[action],
      label: action.startsWith('cache') ? '备份并重建' : '开始执行',
      run: () => API.Repair(action),
    })
  const copyProbe = async () => {
    setBusy(true)
    try {
      setProbe(await API.CreateProbe())
      await refreshSession()
      notify('测试文字已写入并核对。请直接到另一台设备粘贴。')
    } catch (e) {
      notify(errorText(e), true)
    } finally {
      setBusy(false)
    }
  }
  const mark = async (direction: string, success: boolean) => {
    try {
      if (success && direction !== 'iphone-mac') {
        const p = await API.CheckProbe()
        setProbe(p)
        if (!p.matches) {
          notify('Mac 测试文字已被替换，请生成新文字后复测。', true)
          return
        }
      }
      await API.RecordVerification(direction, success)
      await refreshSession()
    } catch (e) {
      notify(errorText(e), true)
    }
  }
  const exportReport = async () => {
    try {
      if (await API.ExportReport()) notify('诊断摘要已导出')
    } catch (e) {
      notify(errorText(e), true)
    }
  }
  const openSettings = (id: string) =>
    API.OpenSettings(id === 'awdl' ? 'wifi' : id).catch((e) => notify(errorText(e), true))
  const suggested =
    symptom === 'send'
      ? 'cache-send'
      : symptom === 'receive'
        ? 'cache-receive'
        : report?.suggestedAction || 'quick'
  const successes = session.verifications.filter((v) => v.success).length,
    running = report?.services.filter((s) => s.running).length || 0,
    locked = busy || scanning

  return (
    <div className="app-shell">
      <div className="window-bar">
        <span>
          我的 Mac <ChevronRight size={12} /> 连续互通
        </span>
        <button onClick={exportReport} disabled={locked}>
          <Download size={13} />
          导出摘要
        </button>
      </div>
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-logo">
            <ClipboardCheck size={25} />
          </span>
          <div>
            <strong>Clipboard Fix</strong>
            <small>通用剪贴板修复</small>
          </div>
        </div>
        <div className="nav-label">工作空间</div>
        <nav aria-label="主导航">
          {nav.map((n) => (
            <button
              key={n.id}
              className={page === n.id ? 'active' : ''}
              onClick={() => setPage(n.id)}
            >
              <n.icon size={18} />
              {n.label}
              {n.id === 'verify' && successes > 0 && <span className="nav-count">{successes}</span>}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="local-note">
            <ShieldCheck size={17} />
            <span>在你的 Mac 上本地运行</span>
          </div>
          <small>
            版本 0.1.0 <span>Go + Wails</span>
          </small>
        </div>
      </aside>
      <main className="main-area">
        <header className="page-header">
          <div>
            <p className="eyebrow">
              {page === 'overview' ? 'YOUR DEVICES, CONNECTED' : 'CLIPBOARD FIX'}
            </p>
            <h1>{nav.find((n) => n.id === page)?.label}</h1>
          </div>
          <button className="button secondary" onClick={scan} disabled={locked}>
            <RefreshCw size={16} className={scanning ? 'spin' : ''} />
            {scanning ? '正在诊断' : '重新诊断'}
            <kbd>⌘ R</kbd>
          </button>
        </header>
        {page === 'overview' && (
          <>
            <section className="hero">
              <div className="hero-copy">
                <span className={`status-pill ${successes ? 'success' : ''}`}>
                  <i />
                  {successes ? `已确认 ${successes} 个方向可用` : '等待跨设备复测'}
                </span>
                <h2>
                  让复制，
                  <br />
                  重新连起来。
                </h2>
                <p>
                  检查 Mac 与 iPhone、iPad 之间的连接，
                  <br />
                  一步步找回顺畅的通用剪贴板。
                </p>
                <button className="button primary" onClick={() => setPage('verify')}>
                  <Copy size={16} />
                  开始跨设备复测
                  <ArrowRight size={15} />
                </button>
                <div className="hero-meta">
                  <span className="tiny-dot" />
                  {report
                    ? `macOS ${report.version} · ${report.model}`
                    : '正在读取这台 Mac 的状态…'}
                </div>
              </div>
              <DeviceScene />
            </section>
            <div className="section-heading">
              <h2>基础连接</h2>
              <span>先确认设备能发现彼此</span>
            </div>
            <section className="checks-grid" aria-label="基础条件诊断">
              {report?.checks.length
                ? report.checks.map((c) => (
                    <CheckCard key={c.id} check={c} onOpen={() => void openSettings(c.id)} />
                  ))
                : ['接力', 'Wi-Fi', '蓝牙', '附近连接'].map((x) => (
                    <div className="check-card skeleton-card" key={x}>
                      <span className="skeleton-icon" />
                      <h3>{x}</h3>
                      <p>{scanning ? '正在检查…' : '等待诊断'}</p>
                    </div>
                  ))}
            </section>
            <div className="overview-columns">
              <section className="panel next-panel">
                <div className="panel-title">
                  <span className="icon-soft">
                    <Wrench size={18} />
                  </span>
                  <h2>按故障方向修复</h2>
                </div>
                <p className="muted">先选择你遇到的情况。</p>
                <div className="symptoms" role="radiogroup" aria-label="故障方向">
                  {[
                    ['send', 'Mac 发不出去'],
                    ['receive', 'Mac 接收不到'],
                    ['both', '两个方向都失败'],
                  ].map(([id, text]) => (
                    <button
                      role="radio"
                      aria-checked={symptom === id}
                      className={symptom === id ? 'selected' : ''}
                      onClick={() => setSymptom(id)}
                      key={id}
                    >
                      {text}
                    </button>
                  ))}
                </div>
                <div className="recommendation">
                  <span>建议操作</span>
                  <strong>{actions[suggested].title}</strong>
                  <p>{actions[suggested].detail}</p>
                </div>
                <button
                  className="button primary full"
                  disabled={locked || !report}
                  onClick={() => repair(suggested)}
                >
                  {suggested.startsWith('cache') ? (
                    <FolderLock size={16} />
                  ) : (
                    <Sparkles size={16} />
                  )}{' '}
                  {suggested.startsWith('cache') ? '备份并重建缓存' : '开始温和修复'}
                  <ArrowRight size={16} />
                </button>
              </section>
              <section className="panel">
                <div className="panel-title">
                  <FileClock size={18} />
                  <h2>剪贴板缓存</h2>
                  <button className="text-button" onClick={() => setPage('tools')}>
                    管理备份
                    <ChevronRight size={13} />
                  </button>
                </div>
                <CacheRow cache={report?.caches[0]} kind="send" />
                <CacheRow cache={report?.caches[1]} kind="receive" />
                <div className="service-summary">
                  <span className="service-dots">
                    {(report?.services || Array(5).fill(null)).map((s, i) => (
                      <i className={s?.running ? 'up' : ''} key={i} />
                    ))}
                  </span>
                  <span>{report ? `${running} / 5 项相关服务正在运行` : '正在读取服务状态'}</span>
                  <button className="text-button" onClick={() => setPage('activity')}>
                    查看
                    <ChevronRight size={12} />
                  </button>
                </div>
                <p className="fine-print">缓存时间用于辅助排查，实际粘贴结果才是恢复依据。</p>
              </section>
            </div>
            {report && (
              <div className="insight">
                <Info size={16} />
                <p>{report.findings[0]}</p>
              </div>
            )}
          </>
        )}
        {page === 'tools' && (
          <>
            <div className="page-intro">
              <p>从轻量刷新到缓存重建，按当前症状选择。</p>
              <span className="tag">
                <ShieldCheck size={13} />
                缓存操作保留备份
              </span>
            </div>
            <section className="tool-grid">
              {(
                [
                  'cache-send',
                  'cache-receive',
                  'quick',
                  'discovery',
                  'identity',
                  'handoff',
                ] as const
              ).map((id, i) => {
                const d = actions[id],
                  Icon = [ArrowUpRight, ArrowDownLeft, Sparkles, RadioTower, ShieldCheck, Link2][i]
                return (
                  <article className={`panel tool-card ${i === 0 ? 'featured' : ''}`} key={id}>
                    <div className="tool-top">
                      <span className="icon-soft">
                        <Icon size={21} />
                      </span>
                      {i === 0 && <span className="tag">单向发送排查</span>}
                    </div>
                    <h2>{d.title}</h2>
                    <p>{d.detail}</p>
                    <button
                      className={`button ${i === 0 ? 'primary' : 'secondary'}`}
                      onClick={() => repair(id)}
                      disabled={locked}
                    >
                      {id.startsWith('cache') ? '备份并重建' : '执行刷新'}
                      <ArrowRight size={14} />
                    </button>
                  </article>
                )
              })}
            </section>
            <section className="panel backup-panel">
              <div className="panel-title">
                <RotateCcw size={18} />
                <h2>可恢复备份</h2>
                <span className="muted small">{session.backups.length} 份记录</span>
              </div>
              {session.backups.length === 0 ? (
                <div className="empty-row">
                  <FolderLock size={24} />
                  <div>
                    <strong>还没有备份</strong>
                    <p>重建缓存时自动保留原文件，并在这里登记。</p>
                  </div>
                </div>
              ) : (
                [...session.backups].reverse().map((b) => (
                  <div className="backup-row" key={b.id}>
                    <div>
                      <strong>{b.kind === 'send' ? '发送缓存' : '接收缓存'}</strong>
                      <p>
                        {timeLabel(b.created)} · {bytesLabel(b.bytes)}
                      </p>
                    </div>
                    <span className="muted small">
                      {b.restored ? '已恢复' : b.ready ? '备份可用' : '备份未完成'}
                    </span>
                    <button
                      className="button secondary small-button"
                      disabled={locked || b.restored || !b.ready}
                      onClick={() =>
                        setConfirm({
                          title: '恢复缓存备份',
                          detail: '恢复所选的旧缓存，并重新加载接力服务。',
                          notes: [
                            '如果原位置已有新的缓存，会先保留它的副本。',
                            '恢复后请重新进行跨设备复测。',
                          ],
                          label: '恢复此备份',
                          run: () => API.RestoreBackup(b.id),
                        })
                      }
                    >
                      <RotateCcw size={13} />
                      恢复
                    </button>
                  </div>
                ))
              )}
            </section>
            <section className="panel additional-panel">
              <div className="panel-title">
                <Settings2 size={18} />
                <h2>更多排查</h2>
              </div>
              <div className="additional-grid">
                <div>
                  <h3>系统与手机设置</h3>
                  <p>
                    核对 Mac 的接力、Wi-Fi 和蓝牙。手机端：设置 → 通用 → 隔空播放与连续互通 → 接力。
                  </p>
                  <div className="button-row">
                    <button
                      className="button secondary small-button"
                      onClick={() => void openSettings('handoff')}
                    >
                      接力设置
                    </button>
                    <button
                      className="button secondary small-button"
                      onClick={() => void openSettings('bluetooth')}
                    >
                      蓝牙设置
                    </button>
                    <button className="text-button" onClick={() => void openSettings('automation')}>
                      访达权限
                    </button>
                  </div>
                </div>
                <div>
                  <h3>剪贴板监控工具</h3>
                  <p>可临时退出工具做对照，再重新打开。</p>
                  {report?.tools.map((t) => (
                    <div className="monitor-row" key={t.name}>
                      <span>
                        <i className={`tiny-dot ${t.running ? '' : 'off'}`} />
                        {t.name}
                      </span>
                      <button
                        className="text-button"
                        disabled={locked}
                        onClick={() =>
                          t.running
                            ? setConfirm({
                                title: `暂时退出 ${t.name}`,
                                detail: '用于排查第三方工具对剪贴板的影响。',
                                notes: [
                                  '该工具的剪贴板监控功能会暂时停用。',
                                  '复测完成后可在这里重新打开。',
                                ],
                                label: '暂时退出',
                                run: () => API.SetToolRunning(t.name, false),
                              })
                            : void execute(
                                () => API.SetToolRunning(t.name, true),
                                `已重新打开 ${t.name}`,
                              )
                        }
                      >
                        {t.running ? '暂时退出' : '重新打开'}
                        <ChevronRight size={12} />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
              <div className="network-row">
                <div>
                  <span className="icon-soft">
                    <Wifi size={18} />
                  </span>
                  <div>
                    <h3>代理与网络对照</h3>
                    <p>
                      {report?.proxy.available
                        ? `Clash · TUN ${report.proxy.tun ? '开启' : '关闭'} · ${report.proxy.mode} · ${report.proxy.stack}`
                        : '未发现支持的本地 Clash 控制器，可在代理工具中手动检查。'}
                    </p>
                  </div>
                </div>
                <button className="button secondary small-button" onClick={() => setGuide(true)}>
                  打开对照指南
                  <ArrowRight size={13} />
                </button>
              </div>
            </section>
          </>
        )}
        {page === 'verify' && (
          <>
            <div className="page-intro">
              <p>设备保持解锁、放在附近，用一段新的纯文字验证。</p>
              <span className="tag">
                <Smartphone size={13} />
                需要另一台设备实际粘贴
              </span>
            </div>
            <section className="panel probe-panel">
              <div className="probe-top">
                <span className="step-number">01</span>
                <div>
                  <h2>生成一段新的测试文字</h2>
                  <p>写入后会核对本机剪贴板。测试期间先别复制其他内容。</p>
                </div>
              </div>
              <div className={`probe-code ${probe ? 'has-code' : ''}`}>
                <code>{probe?.text || '准备好，再复制一次。'}</code>
                {probe && (
                  <span className={probe.matches ? 'probe-match' : 'probe-mismatch'}>
                    {probe.matches ? <CheckCircle2 size={14} /> : <Info size={14} />}{' '}
                    {probe.matches ? 'Mac 已核对' : '已被替换'}
                  </span>
                )}
              </div>
              <div className="button-row">
                <button className="button primary" disabled={locked} onClick={copyProbe}>
                  <Copy size={16} />
                  {probe ? '生成并复制新文字' : '生成并复制测试文字'}
                </button>
                {probe && (
                  <button
                    className="button secondary"
                    disabled={locked}
                    onClick={() =>
                      API.CheckProbe()
                        .then((p) => {
                          setProbe(p)
                          notify(
                            p.matches
                              ? 'Mac 剪贴板仍是这段测试文字'
                              : '测试文字已被替换，请重新生成',
                            !p.matches,
                          )
                        })
                        .catch((e) => notify(errorText(e), true))
                    }
                  >
                    <CheckCheck size={16} />
                    核对本机内容
                  </button>
                )}
              </div>
            </section>
            <section className="panel verification-panel">
              <div className="probe-top">
                <span className="step-number">02</span>
                <div>
                  <h2>在另一台设备粘贴，并记录结果</h2>
                  <p>打开备忘录的空白位置，直接粘贴。请核对文字与上方完全一致。</p>
                </div>
              </div>
              {[
                ['mac-iphone', 'Mac → iPhone', '在 iPhone 备忘录中粘贴'],
                ['mac-ipad', 'Mac → iPad', '在 iPad 备忘录中粘贴'],
                ['iphone-mac', 'iPhone / iPad → Mac', '在手机重新复制普通文字，再到 Mac 粘贴'],
              ].map(([id, title, hint]) => {
                const v = session.verifications.find((x) => x.direction === id)
                return (
                  <div className="verification-row" key={id}>
                    <span className={`direction-icon ${v?.success ? 'verified' : ''}`}>
                      {v?.success ? <Check size={20} /> : <ArrowLeftRight size={20} />}
                    </span>
                    <div>
                      <strong>{title}</strong>
                      <p>
                        {v
                          ? `${v.success ? '用户已确认成功' : '用户确认仍失败'} · ${timeLabel(v.at)}`
                          : hint}
                      </p>
                    </div>
                    <div className="verification-actions">
                      <button
                        className={`button secondary small-button ${v?.success ? 'chosen-success' : ''}`}
                        disabled={locked || (!probe && id !== 'iphone-mac')}
                        onClick={() => void mark(id, true)}
                      >
                        <Check size={14} />
                        已成功
                      </button>
                      <button
                        className="text-button"
                        disabled={locked || (!probe && id !== 'iphone-mac')}
                        onClick={() => void mark(id, false)}
                      >
                        仍然失败
                      </button>
                    </div>
                  </div>
                )
              })}
            </section>
            <div className="hint-banner">
              <CircleHelp size={18} />
              <div>
                <strong>如果还是旧内容或空白</strong>
                <p>
                  重新生成测试文字，核对本机内容；再确认两端接力已开启、使用同一个 Apple
                  账户。也可分别重启设备后再试。
                </p>
              </div>
              <button className="text-button" onClick={() => setPage('tools')}>
                返回修复工具
                <ArrowRight size={14} />
              </button>
            </div>
          </>
        )}
        {page === 'activity' && (
          <>
            <div className="page-intro">
              <p>查看真实执行结果，保留下一次排查需要的摘要。</p>
              <button
                className="button secondary small-button"
                disabled={locked}
                onClick={async () => {
                  setBusy(true)
                  try {
                    setSignals((await API.CollectSignals()) ?? [])
                    await refreshSession()
                  } catch (e) {
                    notify(errorText(e), true)
                  } finally {
                    setBusy(false)
                  }
                }}
              >
                <Terminal size={14} />
                采样接力日志
              </button>
            </div>
            {signals.length > 0 && (
              <section className="signals-grid">
                {signals.map((s) => (
                  <div className="panel signal-card" key={s.title}>
                    <span>{s.title}</span>
                    <strong>{s.count}</strong>
                    <p>{s.detail}</p>
                  </div>
                ))}
              </section>
            )}
            <section className="panel service-panel">
              <div className="panel-title">
                <RadioTower size={18} />
                <h2>系统服务</h2>
                <span className="muted small">仅显示当前用户的服务</span>
              </div>
              {report?.services.map((s) => (
                <div className="service-row" key={s.name}>
                  <span>
                    <i className={`tiny-dot ${s.running ? '' : 'off'}`} />
                    <code>{s.name}</code>
                  </span>
                  <span className="muted">{s.running ? `PID ${s.pid}` : '尚未运行'}</span>
                </div>
              ))}
            </section>
            <section className="panel history-panel">
              <div className="panel-title">
                <History size={18} />
                <h2>活动记录</h2>
                <button className="text-button" onClick={exportReport}>
                  <Download size={13} />
                  导出摘要
                </button>
              </div>
              <div className="event-list">
                {session.history.length === 0 ? (
                  <div className="empty-row">
                    <Info size={23} />
                    <p>完成诊断或修复后，记录会显示在这里。</p>
                  </div>
                ) : (
                  [...session.history].reverse().map((h, i) => (
                    <div
                      className={`event-row ${h.level === 'error' ? 'event-error' : ''}`}
                      key={`${h.at}-${i}`}
                    >
                      <span className="event-indicator">
                        {h.level === 'error' ? (
                          <XCircle size={14} />
                        ) : h.level === 'done' ? (
                          <CheckCircle2 size={14} />
                        ) : (
                          <i />
                        )}
                      </span>
                      <p>{h.message}</p>
                      <time>{timeLabel(h.at)}</time>
                    </div>
                  ))
                )}
              </div>
              <div className="history-footer">
                <ShieldCheck size={13} />
                报告包含诊断摘要、缓存元数据和操作记录，剪贴板正文不会写入报告。
              </div>
            </section>
          </>
        )}
        <footer className="page-footer">
          <span>
            <span className="tiny-dot" />
            本机诊断 · 按需修复
          </span>
          <span>{report ? `上次检查 ${timeLabel(report.at)}` : '等待诊断'}</span>
        </footer>
      </main>
      {busy && (
        <div className="operation-bar" role="status" aria-live="polite">
          <LoaderCircle className="spin" size={18} />
          <div>
            <strong>{progress?.message || '正在处理，请稍候…'}</strong>
            <div className="progress-track">
              <i style={{ width: `${progress?.percent || 12}%` }} />
            </div>
          </div>
          <span>{progress?.percent || 0}%</span>
        </div>
      )}
      {notice && (
        <div
          role={notice.error ? 'alert' : 'status'}
          className={`toast ${notice.error ? 'toast-error' : ''}`}
        >
          {notice.error ? <Info size={18} /> : <CheckCircle2 size={18} />}
          <p>{notice.text}</p>
          <button aria-label="关闭提示" onClick={() => setNotice(null)}>
            <X size={15} />
          </button>
        </div>
      )}
      <Dialog.Root open={!!confirm} onOpenChange={(open) => !open && setConfirm(null)}>
        <Dialog.Portal>
          <Dialog.Overlay className="dialog-overlay" />
          <Dialog.Content className="dialog-content">
            <span className="dialog-icon">
              <Wrench size={24} />
            </span>
            <Dialog.Title>{confirm?.title}</Dialog.Title>
            <Dialog.Description>{confirm?.detail}</Dialog.Description>
            <ul className="dialog-notes">
              {confirm?.notes.map((n) => (
                <li key={n}>
                  <CheckCircle2 size={15} />
                  <span>{n}</span>
                </li>
              ))}
            </ul>
            <div className="dialog-actions">
              <Dialog.Close asChild>
                <button className="button secondary">取消</button>
              </Dialog.Close>
              <button
                className="button primary"
                onClick={() => confirm && void execute(confirm.run)}
              >
                {confirm?.label}
                <ArrowRight size={15} />
              </button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
      <Dialog.Root open={guide} onOpenChange={setGuide}>
        <Dialog.Portal>
          <Dialog.Overlay className="dialog-overlay" />
          <Dialog.Content className="dialog-content wide-dialog">
            <span className="dialog-icon">
              <Wifi size={24} />
            </span>
            <Dialog.Title>代理与网络对照</Dialog.Title>
            <Dialog.Description>在代理工具中手动调整并复测，结束后恢复原设置。</Dialog.Description>
            <div className="baseline">
              <span>对照前状态</span>
              <strong>
                {comparisonBefore?.available
                  ? `TUN ${comparisonBefore.tun ? '开启' : '关闭'} · ${comparisonBefore.mode}`
                  : report?.proxy.available
                    ? `TUN ${report.proxy.tun ? '开启' : '关闭'} · ${report.proxy.mode}`
                    : '请自行记录代理原状态'}
              </strong>
            </div>
            <ol className="guide-steps">
              <li>在 Clash 等代理工具中暂时关闭 TUN。期间网络出口会改变。</li>
              <li>生成新测试文字，直接到手机或 iPad 的备忘录粘贴。</li>
              <li>复测完成后，在代理工具中恢复原来的 TUN 与模式，再重新诊断。</li>
            </ol>
            <p className="fine-print">程序只读取代理状态；计时器用于提醒手动恢复。</p>
            <div className="dialog-actions">
              <Dialog.Close asChild>
                <button className="button secondary">关闭</button>
              </Dialog.Close>
              <button
                className="button primary"
                onClick={() => {
                  setComparisonBefore(report?.proxy || null)
                  setSeconds(90)
                  setGuide(false)
                  setPage('verify')
                  notify('已开始 90 秒手动对照计时；结束后请恢复代理原设置。')
                }}
              >
                开始对照计时
                <ArrowRight size={15} />
              </button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
      {seconds !== null && (
        <div className="comparison-timer" role="status">
          <Wifi size={15} />
          <strong>{seconds}s</strong>
          <span>网络对照 · 结束后手动恢复代理</span>
          <button
            aria-label="结束计时并提醒恢复代理"
            onClick={() => {
              setSeconds(null)
              notify('请在代理工具中恢复原设置，再重新诊断。')
              void scan()
            }}
          >
            <X size={14} />
          </button>
        </div>
      )}
    </div>
  )
}
