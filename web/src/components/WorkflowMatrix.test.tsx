import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { WorkflowMatrix } from './WorkflowMatrix'
import { buildInitialMatrix, DEFAULT_ZONES, STAGE_PERSONAS } from '../types/project'
import type { MatrixCell } from '../types/project'
import { useStore } from '../stores/useStore'
import { DEFAULT_PERSONAS } from '../types'

// Mock the store
vi.mock('../stores/useStore', () => ({
  useStore: vi.fn()
}))

describe('WorkflowMatrix', () => {
  const createTestMatrix = (): MatrixCell[] => buildInitialMatrix('customers')

  beforeEach(() => {
    // Default mock: all personas enabled
    vi.mocked(useStore).mockImplementation((selector) => {
      const state = { personas: DEFAULT_PERSONAS }
      return selector(state as never)
    })
  })

  it('should render all stages and zones', () => {
    const cells = createTestMatrix()
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Check stages are rendered (as uppercase labels)
    expect(screen.getByText('Discovery')).toBeInTheDocument()
    expect(screen.getByText('Goal')).toBeInTheDocument()
    expect(screen.getByText('Requirements')).toBeInTheDocument()
    expect(screen.getByText('Planning')).toBeInTheDocument()
    expect(screen.getByText('Design')).toBeInTheDocument()
    expect(screen.getByText('Implement')).toBeInTheDocument()
    expect(screen.getByText('Verify')).toBeInTheDocument()
    expect(screen.getByText('Validate')).toBeInTheDocument()
    expect(screen.getByText('Document')).toBeInTheDocument()
    expect(screen.getByText('Release')).toBeInTheDocument()

    // Check zones are rendered
    expect(screen.getByText('Frontend')).toBeInTheDocument()
    expect(screen.getByText('Backend')).toBeInTheDocument()
    expect(screen.getByText('Database')).toBeInTheDocument()
    expect(screen.getByText('Shared')).toBeInTheDocument()
  })

  it('should render all personas for each stage', () => {
    const cells = createTestMatrix()
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Check personas are rendered
    expect(screen.getByText('Researcher')).toBeInTheDocument()
    expect(screen.getByText('Analyst')).toBeInTheDocument()
    expect(screen.getByText('Requirements-engineer')).toBeInTheDocument()
    expect(screen.getByText('Architect')).toBeInTheDocument()
    expect(screen.getByText('Designer')).toBeInTheDocument()
    expect(screen.getByText('Developer')).toBeInTheDocument()
    expect(screen.getByText('Debugger')).toBeInTheDocument()
    expect(screen.getByText('Reviewer')).toBeInTheDocument()
    expect(screen.getByText('Security')).toBeInTheDocument()
    expect(screen.getByText('Tester')).toBeInTheDocument()
    expect(screen.getByText('Qa')).toBeInTheDocument()
    expect(screen.getByText('Docs')).toBeInTheDocument()
    expect(screen.getByText('Devops')).toBeInTheDocument()
  })

  it('should toggle single cell on click', () => {
    const cells = createTestMatrix()
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Find a cell and click it - looking for the enabled checkmark
    const allCheckmarks = screen.getAllByText('\u2713')
    fireEvent.click(allCheckmarks[0].closest('td')!)

    expect(onChange).toHaveBeenCalledTimes(1)

    // Verify the callback was called with an updated cells array
    const updatedCells = onChange.mock.calls[0][0] as MatrixCell[]
    expect(updatedCells).toHaveLength(cells.length)

    // At least one cell should be toggled
    const changedCells = updatedCells.filter((c, i) => c.enabled !== cells[i].enabled)
    expect(changedCells.length).toBeGreaterThan(0)
  })

  it('should toggle entire stage row on header click', () => {
    const cells = createTestMatrix()
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Click on the "Discovery" stage header row
    const discoveryHeader = screen.getByText('Discovery')
    fireEvent.click(discoveryHeader.closest('td')!)

    expect(onChange).toHaveBeenCalledTimes(1)

    const updatedCells = onChange.mock.calls[0][0] as MatrixCell[]

    // All discovery stage cells should have the same enabled state
    const discoveryCells = updatedCells.filter((c) => c.stage === 'discovery')
    const firstEnabled = discoveryCells[0].enabled
    expect(discoveryCells.every((c) => c.enabled === firstEnabled)).toBe(true)
  })

  it('should toggle entire zone column on header click', () => {
    const cells = createTestMatrix()
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Click on the "Frontend" zone header
    const frontendHeader = screen.getByText('Frontend')
    fireEvent.click(frontendHeader)

    expect(onChange).toHaveBeenCalledTimes(1)

    const updatedCells = onChange.mock.calls[0][0] as MatrixCell[]

    // All frontend zone cells should have the same enabled state
    const frontendCells = updatedCells.filter((c) => c.zone === 'frontend')
    const firstEnabled = frontendCells[0].enabled
    expect(frontendCells.every((c) => c.enabled === firstEnabled)).toBe(true)
  })

  it('should show correct stage state indicator (all enabled)', () => {
    // All cells enabled
    const cells = createTestMatrix()
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Should show checkmark for fully enabled stages
    const checkmarks = screen.getAllByText('\u2713')
    expect(checkmarks.length).toBeGreaterThan(0)
  })

  it('should show correct stage state indicator (none enabled)', () => {
    // All cells disabled
    const cells = createTestMatrix().map((c) => ({ ...c, enabled: false }))
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Should show circle for disabled
    const emptyCircles = screen.getAllByText('\u25CB')
    expect(emptyCircles.length).toBeGreaterThan(0)
  })

  it('should show correct stage state indicator (some enabled)', () => {
    // Mix of enabled/disabled
    const cells = createTestMatrix()
    cells[0].enabled = false // Disable first cell
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Should show half-circle for partially enabled
    expect(screen.getByText('\u25D0')).toBeInTheDocument()
  })

  it('should call onChange with updated state', () => {
    const cells = createTestMatrix()
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Click any cell
    const firstCheckmark = screen.getAllByText('\u2713')[0]
    fireEvent.click(firstCheckmark.closest('td')!)

    expect(onChange).toHaveBeenCalledWith(expect.any(Array))

    const updatedCells = onChange.mock.calls[0][0] as MatrixCell[]
    expect(updatedCells).toHaveLength(cells.length)

    // Each cell should have the correct structure
    updatedCells.forEach((cell) => {
      expect(cell).toHaveProperty('stage')
      expect(cell).toHaveProperty('zone')
      expect(cell).toHaveProperty('persona')
      expect(cell).toHaveProperty('enabled')
    })
  })

  it('should render legend with instructions', () => {
    const cells = createTestMatrix()
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    expect(screen.getByText(/= enabled/)).toBeInTheDocument()
    expect(screen.getByText(/= disabled/)).toBeInTheDocument()
    expect(screen.getByText(/Click any cell/)).toBeInTheDocument()
  })

  it('should preserve other cells when toggling one', () => {
    const cells = createTestMatrix()
    const onChange = vi.fn()

    render(<WorkflowMatrix cells={cells} onChange={onChange} />)

    // Click to toggle - find a cell td (not a header)
    // Get all tds that are clickable data cells (have checkmark)
    const dataCells = screen.getAllByText('\u2713').filter(el => {
      const td = el.closest('td')
      return td && !td.hasAttribute('colspan') // Exclude header rows
    })

    fireEvent.click(dataCells[0].closest('td')!)

    const updatedCells = onChange.mock.calls[0][0] as MatrixCell[]

    // Count how many cells changed
    let changedCount = 0
    for (let i = 0; i < cells.length; i++) {
      if (cells[i].enabled !== updatedCells[i].enabled) {
        changedCount++
      }
    }

    // Only one cell should have changed (single cell click)
    expect(changedCount).toBe(1)
  })

  describe('with respectPersonaSettings enabled', () => {
    it('should show disabled indicator for disabled personas', () => {
      // Mock store with security persona disabled
      const personas = DEFAULT_PERSONAS.map(p => ({
        ...p,
        enabled: p.id !== 'security'
      }))
      vi.mocked(useStore).mockImplementation((selector) => {
        const state = { personas }
        return selector(state as never)
      })

      const cells = createTestMatrix()
      const onChange = vi.fn()

      render(<WorkflowMatrix cells={cells} onChange={onChange} respectPersonaSettings />)

      // Security row should show (disabled) indicator
      expect(screen.getByText('(disabled)')).toBeInTheDocument()
    })

    it('should not call onChange when clicking disabled persona cell', () => {
      // Mock store with security persona disabled
      const personas = DEFAULT_PERSONAS.map(p => ({
        ...p,
        enabled: p.id !== 'security'
      }))
      vi.mocked(useStore).mockImplementation((selector) => {
        const state = { personas }
        return selector(state as never)
      })

      const cells = createTestMatrix()
      const onChange = vi.fn()

      render(<WorkflowMatrix cells={cells} onChange={onChange} respectPersonaSettings />)

      // Find the Security row and click a cell in it
      const securityRow = screen.getByText('Security').closest('tr')
      const securityCells = securityRow?.querySelectorAll('td')

      // Click the second td (first data cell after label)
      if (securityCells && securityCells[1]) {
        fireEvent.click(securityCells[1])
      }

      // onChange should not be called for disabled persona
      expect(onChange).not.toHaveBeenCalled()
    })

    it('should apply opacity to disabled persona rows', () => {
      // Mock store with qa persona disabled
      const personas = DEFAULT_PERSONAS.map(p => ({
        ...p,
        enabled: p.id !== 'qa'
      }))
      vi.mocked(useStore).mockImplementation((selector) => {
        const state = { personas }
        return selector(state as never)
      })

      const cells = createTestMatrix()
      const onChange = vi.fn()

      render(<WorkflowMatrix cells={cells} onChange={onChange} respectPersonaSettings />)

      // QA row should have opacity class
      const qaRow = screen.getByText('Qa').closest('tr')
      expect(qaRow).toHaveClass('opacity-40')
    })

    it('should work normally when respectPersonaSettings is false', () => {
      // Mock store with security disabled
      const personas = DEFAULT_PERSONAS.map(p => ({
        ...p,
        enabled: p.id !== 'security'
      }))
      vi.mocked(useStore).mockImplementation((selector) => {
        const state = { personas }
        return selector(state as never)
      })

      const cells = createTestMatrix()
      const onChange = vi.fn()

      // Without respectPersonaSettings prop
      render(<WorkflowMatrix cells={cells} onChange={onChange} />)

      // Should NOT show (disabled) indicator
      expect(screen.queryByText('(disabled)')).not.toBeInTheDocument()

      // Should be able to click Security cells
      const securityRow = screen.getByText('Security').closest('tr')
      const securityCells = securityRow?.querySelectorAll('td')

      if (securityCells && securityCells[1]) {
        fireEvent.click(securityCells[1])
      }

      // onChange should be called
      expect(onChange).toHaveBeenCalled()
    })
  })
})
