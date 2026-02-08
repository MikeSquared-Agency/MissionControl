// Project types for MissionControl
// Used by Project Wizard and global config management

import type { Stage } from './workflow'
import { ALL_STAGES } from './workflow'

// Project stored in ~/.mission-control/config.json
export interface Project {
  path: string
  name: string
  lastOpened: string // ISO 8601
  mode?: 'online' | 'offline'
  ollamaModel?: string
}

// Global config structure
export interface GlobalConfig {
  projects: Project[]
  lastProject: string | null
  preferences: {
    theme: 'dark' | 'light'
  }
}

// Wizard step state
export type WizardStep = 'setup' | 'matrix'

// Workflow matrix cell
export interface MatrixCell {
  stage: Stage
  zone: string
  persona: string
  enabled: boolean
}

// Wizard form data sent to API
export interface WizardFormData {
  path: string
  initGit: boolean
  enableOpenClaw: boolean
  matrix: MatrixCell[]
  mode: 'online' | 'offline'
  ollamaModel?: string
}

// Audience type (UI-only, for matrix defaults)
export type Audience = 'personal' | 'customers'

// Personas organized by stage
export const STAGE_PERSONAS: Record<Stage, string[]> = {
  discovery: ['researcher'],
  goal: ['analyst'],
  requirements: ['requirements-engineer'],
  planning: ['architect'],
  design: ['designer'],
  implement: ['developer', 'debugger'],
  verify: ['reviewer', 'security', 'tester'],
  validate: ['qa'],
  document: ['docs'],
  release: ['devops']
}

// Default zones
export const DEFAULT_ZONES = ['frontend', 'backend', 'database', 'shared']

// Personas disabled by default for personal projects
export const PERSONAL_DISABLED_PERSONAS = ['security', 'qa', 'devops']

// Build initial matrix based on audience
export function buildInitialMatrix(audience: Audience): MatrixCell[] {
  const cells: MatrixCell[] = []

  for (const stage of ALL_STAGES) {
    for (const zone of DEFAULT_ZONES) {
      for (const persona of STAGE_PERSONAS[stage]) {
        cells.push({
          stage,
          zone,
          persona,
          enabled: audience === 'customers' || !PERSONAL_DISABLED_PERSONAS.includes(persona)
        })
      }
    }
  }
  return cells
}

// Path check response from API
export interface PathCheckResult {
  exists: boolean
  hasGit: boolean
  hasMission: boolean
}

// Directory entry for browsing
export interface DirEntry {
  name: string
  isDir: boolean
}

// Browse response from API
export interface BrowseResult {
  path: string
  entries: DirEntry[]
}
