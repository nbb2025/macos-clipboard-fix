export type Check = { id: string; title: string; status: string; detail: string }
export type Cache = {
  kind: string
  path: string
  exists: boolean
  bytes: number
  modified: string
  ageSeconds: number
  detail: string
}
export type Backup = {
  id: string
  kind: string
  original: string
  path: string
  created: string
  bytes: number
  restored: boolean
  ready: boolean
}
export type Report = {
  at: string
  version: string
  model: string
  checks: Check[]
  services: { name: string; label: string; pid: number; running: boolean }[]
  caches: Cache[]
  tools: { name: string; running: boolean }[]
  proxy: { available: boolean; tun: boolean; mode: string; stack: string; appleRoute: string }
  findings: string[]
  suggestedAction: string
}
export type Progress = { action: string; percent: number; message: string; state: string }
export type Probe = { text: string; created: string; matches: boolean }
export type Signal = { title: string; count: number; detail: string }
export type Session = {
  history: { at: string; level: string; message: string }[]
  backups: Backup[]
  verifications: { direction: string; success: boolean; at: string }[]
}
