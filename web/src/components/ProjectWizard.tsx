import { useState, useEffect, useCallback } from 'react'
import { Modal } from './Modal'
import { Spinner } from './Spinner'
import { TypingIndicator } from './TypingIndicator'
import { WorkflowMatrix } from './WorkflowMatrix'
import { useProjectStore, createProject, checkPath } from '../stores/useProjectStore'
import { toast } from '../stores/useToast'
import type { WizardStep, MatrixCell, Audience, PathCheckResult } from '../types/project'
import { buildInitialMatrix } from '../types/project'

export function ProjectWizard() {
  const wizardOpen = useProjectStore((s) => s.wizardOpen)
  const closeWizard = useProjectStore((s) => s.closeWizard)
  const addProject = useProjectStore((s) => s.addProject)
  const setCurrentProject = useProjectStore((s) => s.setCurrentProject)

  const [step, setStep] = useState<WizardStep>('setup')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [pathStatus, setPathStatus] = useState<PathCheckResult>({
    exists: false,
    hasGit: false,
    hasMission: false
  })

  // Form state
  const [path, setPath] = useState('')
  const [audience, setAudience] = useState<Audience>('personal')
  const [initGit, setInitGit] = useState(true)
  const [enableKing, setEnableKing] = useState(true)
  const [matrix, setMatrix] = useState<MatrixCell[]>([])

  // Reset on open
  useEffect(() => {
    if (wizardOpen) {
      setStep('setup')
      setPath('')
      setAudience('personal')
      setInitGit(true)
      setEnableKing(true)
      setMatrix(buildInitialMatrix('personal'))
      setError('')
      setPathStatus({ exists: false, hasGit: false, hasMission: false })
    }
  }, [wizardOpen])

  // Update matrix when audience changes
  useEffect(() => {
    setMatrix(buildInitialMatrix(audience))
  }, [audience])

  // Check path status (detect .git, .mission)
  const doCheckPath = useCallback(async (p: string) => {
    if (!p.trim()) {
      setPathStatus({ exists: false, hasGit: false, hasMission: false })
      return
    }
    try {
      const result = await checkPath(p)
      setPathStatus(result)
      if (result.hasGit) {
        setInitGit(false)
      }
    } catch {
      setPathStatus({ exists: false, hasGit: false, hasMission: false })
    }
  }, [])

  // Debounced path check
  useEffect(() => {
    const timer = setTimeout(() => doCheckPath(path), 300)
    return () => clearTimeout(timer)
  }, [path, doCheckPath])

  const handleContinue = () => {
    if (pathStatus.hasMission) {
      // Open existing project instead
      setCurrentProject(path)
      closeWizard()
      toast.success('Opened existing project')
      return
    }
    setStep('matrix')
  }

  const handleSubmit = async () => {
    setLoading(true)
    setError('')

    try {
      const project = await createProject({
        path,
        initGit: !pathStatus.hasGit && initGit,
        enableKing,
        matrix
      })
      addProject(project)
      setCurrentProject(project.path)
      closeWizard()
      toast.success(`Project "${project.name}" created`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create project')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal open={wizardOpen} onClose={closeWizard} title="New Project" width="lg">
      {step === 'setup' ? (
        <div className="space-y-6">
          {/* Conversational prompt */}
          <div className="flex items-start gap-3">
            <span className="text-2xl">⏺</span>
            <TypingIndicator text="Let's set up your project." delay={300} />
          </div>

          {/* Path input */}
          <div>
            <label className="block text-[11px] text-gray-500 mb-1.5">Where?</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder="/Users/you/projects/myapp"
                className="flex-1 px-3 py-2 text-sm font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 focus:outline-none focus:border-gray-600"
                autoFocus
              />
            </div>
            {pathStatus.hasMission && (
              <p className="mt-1 text-[10px] text-amber-400">
                Project already exists at this path. Click Continue to open it.
              </p>
            )}
            {pathStatus.hasGit && !pathStatus.hasMission && (
              <p className="mt-1 text-[10px] text-blue-400">
                Git repository detected. "Initialize git" will be skipped.
              </p>
            )}
            {!pathStatus.exists && path.trim() && (
              <p className="mt-1 text-[10px] text-gray-500">This folder will be created.</p>
            )}
          </div>

          {/* Audience selector */}
          <div>
            <label className="block text-[11px] text-gray-500 mb-1.5">Who's it for?</label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setAudience('personal')}
                className={`flex-1 py-2 text-sm font-medium rounded transition-colors ${
                  audience === 'personal'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
                }`}
              >
                Personal
              </button>
              <button
                type="button"
                onClick={() => setAudience('customers')}
                className={`flex-1 py-2 text-sm font-medium rounded transition-colors ${
                  audience === 'customers'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
                }`}
              >
                Customers
              </button>
            </div>
            <p className="mt-1 text-[10px] text-gray-600">
              {audience === 'personal'
                ? 'Personal projects skip Security, QA, and DevOps by default'
                : 'Customer-facing projects enable all workflow steps'}
            </p>
          </div>

          {/* Checkboxes */}
          <div className="space-y-3">
            <label className="flex items-center gap-3">
              <input
                type="checkbox"
                checked={initGit}
                onChange={(e) => setInitGit(e.target.checked)}
                disabled={pathStatus.hasGit || pathStatus.hasMission}
                className="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600 focus:ring-blue-500 focus:ring-offset-gray-900"
              />
              <span
                className={`text-sm ${pathStatus.hasGit ? 'text-gray-600' : 'text-gray-300'}`}
              >
                Initialize git
              </span>
            </label>
            <label className="flex items-center gap-3">
              <input
                type="checkbox"
                checked={enableKing}
                onChange={(e) => setEnableKing(e.target.checked)}
                className="w-4 h-4 rounded border-gray-600 bg-gray-800 text-blue-600 focus:ring-blue-500 focus:ring-offset-gray-900"
              />
              <span className="text-sm text-gray-300">Enable King (recommended)</span>
            </label>
          </div>

          {/* Actions */}
          <div className="flex gap-2 pt-2">
            <button
              type="button"
              onClick={closeWizard}
              className="flex-1 py-2 text-sm text-gray-400 bg-gray-800 hover:bg-gray-700 rounded transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleContinue}
              disabled={!path.trim()}
              className="flex-1 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:cursor-not-allowed rounded transition-colors"
            >
              {pathStatus.hasMission ? 'Open Project' : 'Continue →'}
            </button>
          </div>
        </div>
      ) : (
        <div className="space-y-6">
          {/* Conversational prompt */}
          <div className="flex items-start gap-3">
            <span className="text-2xl">⏺</span>
            <TypingIndicator text="Here's your workflow. Click any cell to toggle." delay={300} />
          </div>

          {/* Matrix */}
          <WorkflowMatrix cells={matrix} onChange={setMatrix} />

          {/* Error */}
          {error && (
            <div className="px-3 py-2 text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded">
              {error}
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-2 pt-2">
            <button
              type="button"
              onClick={() => setStep('setup')}
              className="flex-1 py-2 text-sm text-gray-400 bg-gray-800 hover:bg-gray-700 rounded transition-colors"
            >
              ← Back
            </button>
            <button
              type="button"
              onClick={handleSubmit}
              disabled={loading}
              className="flex-1 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:cursor-not-allowed rounded transition-colors flex items-center justify-center gap-2"
            >
              {loading && <Spinner size="sm" />}
              {loading ? 'Creating...' : 'Create Project'}
            </button>
          </div>
        </div>
      )}
    </Modal>
  )
}
