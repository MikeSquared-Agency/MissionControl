// v4 Types for MissionControl
// Matches Go types in orchestrator/v4/types.go and Rust types in core/

// ============================================================================
// Workflow Domain
// ============================================================================

export type Phase = 'idea' | 'design' | 'implement' | 'verify' | 'document' | 'release'

export const ALL_PHASES: Phase[] = ['idea', 'design', 'implement', 'verify', 'document', 'release']

export type TaskStatus = 'pending' | 'ready' | 'in_progress' | 'blocked' | 'done'

export interface Task {
  id: string
  name: string
  phase: Phase
  zone: string
  status: TaskStatus
  blocked_reason?: string
  persona: string
  dependencies: string[]
  created_at: number
  updated_at: number
}

export type GateStatus = 'open' | 'closed' | 'awaiting_approval'

export interface GateCriterion {
  description: string
  satisfied: boolean
}

export interface Gate {
  id: string
  phase: Phase
  status: GateStatus
  criteria: GateCriterion[]
  approved_at?: number
  approved_by?: string
}

export interface PhaseInfo {
  phase: Phase
  status: 'complete' | 'current' | 'pending'
}

// ============================================================================
// Knowledge Domain
// ============================================================================

export type FindingType = 'discovery' | 'blocker' | 'decision' | 'concern'

export interface Finding {
  type: FindingType
  summary: string
  details_path?: string
  severity?: 'low' | 'medium' | 'high'
}

export type BudgetStatus = 'healthy' | 'warning' | 'critical' | 'exceeded'

export interface TokenBudget {
  worker_id: string
  budget: number
  used: number
  status: BudgetStatus
  remaining: number
}

export interface SuccessorContext {
  key_decisions: string[]
  gotchas: string[]
  recommended_approach?: string
}

export type HandoffStatus = 'complete' | 'blocked' | 'partial'

export interface Handoff {
  task_id: string
  worker_id: string
  status: HandoffStatus
  findings: Finding[]
  artifacts: string[]
  open_questions?: string[]
  context_for_successor?: SuccessorContext
  blocked_reason?: string
  timestamp: number
}

export interface Checkpoint {
  id: string
  phase: Phase
  created_at: number
  tasks_snapshot: Task[]
  findings_snapshot: Finding[]
  decisions: string[]
}

export interface CheckpointSummary {
  id: string
  phase: Phase
  created_at: number
}

// ============================================================================
// Runtime Domain
// ============================================================================

export type HealthStatus = 'healthy' | 'idle' | 'stuck' | 'unresponsive' | 'dead'

export interface WorkerHealth {
  worker_id: string
  status: HealthStatus
  since_ms?: number
}

// ============================================================================
// API Request/Response Types
// ============================================================================

// GET /api/phases
export interface PhasesResponse {
  current: Phase
  phases: PhaseInfo[]
}

// GET /api/tasks
export interface TasksResponse {
  tasks: Task[]
}

// POST /api/tasks
export interface CreateTaskRequest {
  name: string
  phase?: Phase
  zone: string
  persona: string
  dependencies?: string[]
}

// PUT /api/tasks/:id/status
export interface UpdateTaskStatusRequest {
  status: TaskStatus
  reason?: string
}

// POST /api/handoffs
export interface HandoffRequest {
  task_id: string
  worker_id: string
  status: HandoffStatus
  findings: Finding[]
  artifacts: string[]
  open_questions?: string[]
  context_for_successor?: SuccessorContext
  blocked_reason?: string
}

export interface HandoffResponse {
  valid: boolean
  errors?: string[]
  delta_id?: string
}

// GET /api/checkpoints
export interface CheckpointsResponse {
  checkpoints: CheckpointSummary[]
}

// POST /api/gates/:id/approve
export interface GateApprovalRequest {
  approved_by: string
  comment?: string
}

export interface GateApprovalResponse {
  gate: Gate
  can_proceed: boolean
}

// ============================================================================
// WebSocket Event Types
// ============================================================================

// Workflow events
export interface PhaseChangedEvent {
  type: 'phase_changed'
  phase: Phase
  previous: Phase
}

export interface TaskCreatedEvent {
  type: 'task_created'
  task: Task
}

export interface TaskUpdatedEvent {
  type: 'task_updated'
  task_id: string
  status: TaskStatus
  previous: TaskStatus
}

export interface GateStatusEvent {
  type: 'gate_status'
  phase: Phase
  status: GateStatus
  criteria?: GateCriterion[]
}

// Knowledge events
export interface TokenWarningEvent {
  type: 'token_warning' | 'token_critical'
  worker_id: string
  usage: number
  budget: number
  status: BudgetStatus
  remaining: number
}

export interface CheckpointCreatedEvent {
  type: 'checkpoint_created'
  checkpoint_id: string
  phase: Phase
}

export interface HandoffReceivedEvent {
  type: 'handoff_received'
  task_id: string
  worker_id: string
  status: HandoffStatus
}

export interface HandoffValidatedEvent {
  type: 'handoff_validated'
  task_id: string
  valid: boolean
  errors?: string[]
}

// Runtime events
export interface AgentHealthEvent {
  type: 'agent_health'
  agent_id: string
  health: HealthStatus
}

export interface AgentStuckEvent {
  type: 'agent_stuck'
  agent_id: string
  since_ms: number
}

// Initial state event
export interface V4StateEvent {
  type: 'v4_state'
  state: {
    current_phase: Phase
    phases: PhaseInfo[]
    tasks: Task[]
    checkpoints: CheckpointSummary[]
  }
}

// Union type for all v4 events
export type V4Event =
  | PhaseChangedEvent
  | TaskCreatedEvent
  | TaskUpdatedEvent
  | GateStatusEvent
  | TokenWarningEvent
  | CheckpointCreatedEvent
  | HandoffReceivedEvent
  | HandoffValidatedEvent
  | AgentHealthEvent
  | AgentStuckEvent
  | V4StateEvent

// ============================================================================
// Helper Functions
// ============================================================================

export function getNextPhase(phase: Phase): Phase | null {
  const index = ALL_PHASES.indexOf(phase)
  if (index === -1 || index === ALL_PHASES.length - 1) {
    return null
  }
  return ALL_PHASES[index + 1]
}

export function getPreviousPhase(phase: Phase): Phase | null {
  const index = ALL_PHASES.indexOf(phase)
  if (index <= 0) {
    return null
  }
  return ALL_PHASES[index - 1]
}

export function getPhaseLabel(phase: Phase): string {
  return phase.charAt(0).toUpperCase() + phase.slice(1)
}

export function getTaskStatusColor(status: TaskStatus): string {
  switch (status) {
    case 'pending': return '#6b7280'  // gray
    case 'ready': return '#3b82f6'    // blue
    case 'in_progress': return '#f59e0b' // amber
    case 'blocked': return '#ef4444'  // red
    case 'done': return '#22c55e'     // green
  }
}

export function getGateStatusColor(status: GateStatus): string {
  switch (status) {
    case 'open': return '#22c55e'           // green
    case 'closed': return '#ef4444'         // red
    case 'awaiting_approval': return '#f59e0b' // amber
  }
}

export function getBudgetStatusColor(status: BudgetStatus): string {
  switch (status) {
    case 'healthy': return '#22c55e'   // green
    case 'warning': return '#f59e0b'   // amber
    case 'critical': return '#ef4444'  // red
    case 'exceeded': return '#7f1d1d'  // dark red
  }
}

export function getHealthStatusColor(status: HealthStatus): string {
  switch (status) {
    case 'healthy': return '#22c55e'      // green
    case 'idle': return '#6b7280'         // gray
    case 'stuck': return '#f59e0b'        // amber
    case 'unresponsive': return '#ef4444' // red
    case 'dead': return '#7f1d1d'         // dark red
  }
}
