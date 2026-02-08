import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { ProjectWizard } from './ProjectWizard'
import { useProjectStore } from '../stores/useProjectStore'

// Mock the API functions
vi.mock('../stores/useProjectStore', async () => {
  const actual = await vi.importActual('../stores/useProjectStore')
  return {
    ...actual,
    createProject: vi.fn(),
    checkPath: vi.fn()
  }
})

import { createProject, checkPath } from '../stores/useProjectStore'

const mockCreateProject = vi.mocked(createProject)
const mockCheckPath = vi.mocked(checkPath)

describe('ProjectWizard', () => {
  beforeEach(() => {
    // Reset store and mocks
    useProjectStore.setState({
      projects: [],
      currentProject: null,
      wizardOpen: true
    })
    vi.clearAllMocks()
    mockCheckPath.mockResolvedValue({ exists: false, hasGit: false, hasMission: false })
  })

  describe('Setup Step', () => {
    it('should render setup form when wizard opens', () => {
      render(<ProjectWizard />)

      expect(screen.getByPlaceholderText(/projects/i)).toBeInTheDocument()
      expect(screen.getByText('Personal')).toBeInTheDocument()
      expect(screen.getByText('Customers')).toBeInTheDocument()
      expect(screen.getByText(/Initialize git/i)).toBeInTheDocument()
      expect(screen.getByText(/Enable OpenClaw/i)).toBeInTheDocument()
    })

    it('should disable continue button when path is empty', () => {
      render(<ProjectWizard />)

      const continueButton = screen.getByRole('button', { name: /Continue/i })
      expect(continueButton).toBeDisabled()
    })

    it('should enable continue button when path is filled', async () => {
      render(<ProjectWizard />)

      const input = screen.getByPlaceholderText(/projects/i)
      await act(async () => {
        fireEvent.change(input, { target: { value: '/test/project' } })
      })

      await waitFor(() => {
        const continueButton = screen.getByRole('button', { name: /Continue/i })
        expect(continueButton).not.toBeDisabled()
      })
    })

    it('should detect existing git repository and show message', async () => {
      mockCheckPath.mockResolvedValue({ exists: true, hasGit: true, hasMission: false })

      render(<ProjectWizard />)

      const input = screen.getByPlaceholderText(/projects/i)
      await act(async () => {
        fireEvent.change(input, { target: { value: '/existing/git/repo' } })
      })

      await waitFor(() => {
        expect(screen.getByText(/Git repository detected/i)).toBeInTheDocument()
      })
    })

    it('should detect existing .mission folder and show "Open Project"', async () => {
      mockCheckPath.mockResolvedValue({ exists: true, hasGit: true, hasMission: true })

      render(<ProjectWizard />)

      const input = screen.getByPlaceholderText(/projects/i)
      await act(async () => {
        fireEvent.change(input, { target: { value: '/existing/mission/project' } })
      })

      await waitFor(() => {
        expect(screen.getByText(/Found existing MissionControl project/i)).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /Import Project/i })).toBeInTheDocument()
      })
    })

    it('should switch audience between personal and customers', async () => {
      render(<ProjectWizard />)

      const customersButton = screen.getByText('Customers')
      await act(async () => {
        fireEvent.click(customersButton)
      })

      expect(screen.getByText(/Customer-facing projects enable all workflow steps/i)).toBeInTheDocument()

      const personalButton = screen.getByText('Personal')
      await act(async () => {
        fireEvent.click(personalButton)
      })

      expect(screen.getByText(/Personal projects skip Security, QA, and DevOps/i)).toBeInTheDocument()
    })
  })

  describe('Matrix Step', () => {
    it('should navigate to matrix step on continue', async () => {
      render(<ProjectWizard />)

      const input = screen.getByPlaceholderText(/projects/i)
      await act(async () => {
        fireEvent.change(input, { target: { value: '/test/project' } })
      })

      await waitFor(() => {
        const continueButton = screen.getByRole('button', { name: /Continue/i })
        expect(continueButton).not.toBeDisabled()
      })

      const continueButton = screen.getByRole('button', { name: /Continue/i })
      await act(async () => {
        fireEvent.click(continueButton)
      })

      // Check for matrix table with zone headers (verifies we're on matrix step)
      expect(screen.getByText('Frontend')).toBeInTheDocument()
      expect(screen.getByText('Backend')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Create Project/i })).toBeInTheDocument()
    })

    it('should navigate back to setup step', async () => {
      render(<ProjectWizard />)

      // Go to matrix step
      const input = screen.getByPlaceholderText(/projects/i)
      await act(async () => {
        fireEvent.change(input, { target: { value: '/test/project' } })
      })

      await waitFor(() => {
        const continueButton = screen.getByRole('button', { name: /Continue/i })
        expect(continueButton).not.toBeDisabled()
      })

      const continueButton = screen.getByRole('button', { name: /Continue/i })
      await act(async () => {
        fireEvent.click(continueButton)
      })

      // Go back
      const backButton = screen.getByRole('button', { name: /Back/i })
      await act(async () => {
        fireEvent.click(backButton)
      })

      expect(screen.getByPlaceholderText(/projects/i)).toBeInTheDocument()
    })
  })

  describe('Project Creation', () => {
    it('should call createProject API on submit', async () => {
      mockCreateProject.mockResolvedValue({
        path: '/test/project',
        name: 'project',
        lastOpened: new Date().toISOString()
      })

      render(<ProjectWizard />)

      // Fill path and continue
      const input = screen.getByPlaceholderText(/projects/i)
      await act(async () => {
        fireEvent.change(input, { target: { value: '/test/project' } })
      })

      await waitFor(() => {
        const continueButton = screen.getByRole('button', { name: /Continue/i })
        expect(continueButton).not.toBeDisabled()
      })

      const continueButton = screen.getByRole('button', { name: /Continue/i })
      await act(async () => {
        fireEvent.click(continueButton)
      })

      // Submit
      const createButton = screen.getByRole('button', { name: /Create Project/i })
      await act(async () => {
        fireEvent.click(createButton)
      })

      await waitFor(() => {
        expect(mockCreateProject).toHaveBeenCalledWith(
          expect.objectContaining({
            path: '/test/project'
          })
        )
      })
    })

    it('should show loading state during creation', async () => {
      mockCreateProject.mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve({
          path: '/test/project',
          name: 'project',
          lastOpened: new Date().toISOString()
        }), 100))
      )

      render(<ProjectWizard />)

      // Fill path and continue
      const input = screen.getByPlaceholderText(/projects/i)
      await act(async () => {
        fireEvent.change(input, { target: { value: '/test/project' } })
      })

      await waitFor(() => {
        const continueButton = screen.getByRole('button', { name: /Continue/i })
        expect(continueButton).not.toBeDisabled()
      })

      const continueButton = screen.getByRole('button', { name: /Continue/i })
      await act(async () => {
        fireEvent.click(continueButton)
      })

      // Submit
      const createButton = screen.getByRole('button', { name: /Create Project/i })
      await act(async () => {
        fireEvent.click(createButton)
      })

      expect(screen.getByText(/Creating/i)).toBeInTheDocument()
    })

    it('should display error message on failure', async () => {
      mockCreateProject.mockRejectedValue(new Error('Failed to create project'))

      render(<ProjectWizard />)

      // Fill path and continue
      const input = screen.getByPlaceholderText(/projects/i)
      await act(async () => {
        fireEvent.change(input, { target: { value: '/test/project' } })
      })

      await waitFor(() => {
        const continueButton = screen.getByRole('button', { name: /Continue/i })
        expect(continueButton).not.toBeDisabled()
      })

      const continueButton = screen.getByRole('button', { name: /Continue/i })
      await act(async () => {
        fireEvent.click(continueButton)
      })

      // Submit
      const createButton = screen.getByRole('button', { name: /Create Project/i })
      await act(async () => {
        fireEvent.click(createButton)
      })

      await waitFor(() => {
        expect(screen.getByText(/Failed to create project/i)).toBeInTheDocument()
      })
    })
  })

  describe('Wizard Open/Close', () => {
    it('should reset form state when wizard reopens', async () => {
      const { rerender } = render(<ProjectWizard />)

      // Fill the path
      const input = screen.getByPlaceholderText(/projects/i)
      await act(async () => {
        fireEvent.change(input, { target: { value: '/test/project' } })
      })

      // Close and reopen
      act(() => {
        useProjectStore.getState().closeWizard()
      })
      rerender(<ProjectWizard />)

      act(() => {
        useProjectStore.getState().openWizard()
      })
      rerender(<ProjectWizard />)

      // Check that input is reset
      const newInput = screen.getByPlaceholderText(/projects/i)
      expect(newInput).toHaveValue('')
    })

    it('should not render when wizard is closed', () => {
      act(() => {
        useProjectStore.setState({ wizardOpen: false })
      })

      render(<ProjectWizard />)

      expect(screen.queryByText(/set up your project/i)).not.toBeInTheDocument()
    })
  })
})
