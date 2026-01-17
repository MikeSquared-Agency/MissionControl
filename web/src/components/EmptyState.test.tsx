import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { NoAgentsRunning, NoZones, NoConversation, NoFindings } from './EmptyState'

describe('EmptyState Components', () => {
  describe('NoAgentsRunning', () => {
    it('should render empty agents message', () => {
      render(<NoAgentsRunning />)
      expect(screen.getByText('No agents running')).toBeInTheDocument()
    })
  })

  describe('NoZones', () => {
    it('should render empty zones message', () => {
      render(<NoZones />)
      expect(screen.getByText('No zones created')).toBeInTheDocument()
    })
  })

  describe('NoConversation', () => {
    it('should render empty conversation message', () => {
      render(<NoConversation />)
      expect(screen.getByText('No messages yet')).toBeInTheDocument()
    })
  })

  describe('NoFindings', () => {
    it('should render empty findings message', () => {
      render(<NoFindings />)
      expect(screen.getByText('No findings yet')).toBeInTheDocument()
    })
  })
})
