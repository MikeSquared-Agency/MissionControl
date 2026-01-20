// Project types for MissionControl
// Used by Project Wizard and global config management

import type { Phase } from './workflow'

// Project stored in ~/.mission-control/config.json
export interface Project {
  path: string
  name: string
  lastOpened: string // ISO 8601
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
  phase: Phase
  zone: string
  persona: string
  enabled: boolean
}

// Wizard form data sent to API
export interface WizardFormData {
  path: string
  initGit: boolean
  enableKing: boolean
  matrix: MatrixCell[]
}

// Audience type (UI-only, for matrix defaults)
export type Audience = 'personal' | 'customers'

// Personas organized by phase
export const PHASE_PERSONAS: Record<Phase, string[]> = {
  idea: ['researcher'],
  design: ['designer', 'architect'],
  implement: ['developer', 'debugger'],
  verify: ['reviewer', 'security', 'tester', 'qa'],
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
  const phases: Phase[] = ['idea', 'design', 'implement', 'verify', 'document', 'release']

  for (const phase of phases) {
    for (const zone of DEFAULT_ZONES) {
      for (const persona of PHASE_PERSONAS[phase]) {
        cells.push({
          phase,
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
