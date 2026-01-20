import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Project, WizardFormData, PathCheckResult } from '../types/project'

interface ProjectState {
  // State
  projects: Project[]
  currentProject: string | null
  wizardOpen: boolean

  // Actions
  setProjects: (projects: Project[]) => void
  setCurrentProject: (path: string | null) => void
  addProject: (project: Project) => void
  removeProject: (path: string) => void
  openWizard: () => void
  closeWizard: () => void
}

export const useProjectStore = create<ProjectState>()(
  persist(
    (set) => ({
      projects: [],
      currentProject: null,
      wizardOpen: false,

      setProjects: (projects) => set({ projects }),

      setCurrentProject: (path) => set({ currentProject: path }),

      addProject: (project) =>
        set((state) => ({
          projects: [
            ...state.projects.filter((p) => p.path !== project.path),
            project
          ]
        })),

      removeProject: (path) =>
        set((state) => ({
          projects: state.projects.filter((p) => p.path !== path),
          currentProject: state.currentProject === path ? null : state.currentProject
        })),

      openWizard: () => set({ wizardOpen: true }),

      closeWizard: () => set({ wizardOpen: false })
    }),
    {
      name: 'mission-control-projects',
      partialize: (state) => ({
        projects: state.projects,
        currentProject: state.currentProject
      })
    }
  )
)

// Selectors
export const useProjects = () => useProjectStore((s) => s.projects)
export const useCurrentProject = () => useProjectStore((s) => s.currentProject)
export const useWizardOpen = () => useProjectStore((s) => s.wizardOpen)

// API base
const API_BASE = '/api'

// Fetch all projects from global config
export async function fetchProjects(): Promise<Project[]> {
  const res = await fetch(`${API_BASE}/projects`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  const data = await res.json()
  return data.projects || []
}

// Create a new project
export async function createProject(form: WizardFormData): Promise<Project> {
  const res = await fetch(`${API_BASE}/projects`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(form)
  })
  if (!res.ok) {
    const errorText = await res.text()
    try {
      const errorJson = JSON.parse(errorText)
      throw new Error(errorJson.error || errorText)
    } catch {
      throw new Error(errorText)
    }
  }
  return res.json()
}

// Delete a project from the list (not from disk)
export async function deleteProject(path: string): Promise<void> {
  const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(path)}`, {
    method: 'DELETE'
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
}

// Check path status (exists, has .git, has .mission)
export async function checkPath(path: string): Promise<PathCheckResult> {
  const res = await fetch(`${API_BASE}/projects/check?path=${encodeURIComponent(path)}`)
  if (!res.ok) {
    // Return defaults on error
    return { exists: false, hasGit: false, hasMission: false }
  }
  return res.json()
}
