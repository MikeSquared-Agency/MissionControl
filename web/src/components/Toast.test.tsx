import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, act } from '@testing-library/react'
import { ToastContainer } from './Toast'
import { useToast } from '../stores/useToast'

describe('ToastContainer', () => {
  beforeEach(() => {
    // Clear all toasts before each test
    useToast.setState({ toasts: [] })
  })

  it('should render nothing when no toasts', () => {
    const { container } = render(<ToastContainer />)
    expect(container.firstChild).toBeNull()
  })

  it('should render a success toast', () => {
    act(() => {
      useToast.getState().addToast({ type: 'success', message: 'Operation successful' })
    })

    render(<ToastContainer />)
    expect(screen.getByText('Operation successful')).toBeInTheDocument()
  })

  it('should render an error toast', () => {
    act(() => {
      useToast.getState().addToast({ type: 'error', message: 'Something went wrong' })
    })

    render(<ToastContainer />)
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })

  it('should render an info toast', () => {
    act(() => {
      useToast.getState().addToast({ type: 'info', message: 'Just letting you know' })
    })

    render(<ToastContainer />)
    expect(screen.getByText('Just letting you know')).toBeInTheDocument()
  })

  it('should render multiple toasts', () => {
    act(() => {
      useToast.getState().addToast({ type: 'success', message: 'First toast' })
      useToast.getState().addToast({ type: 'error', message: 'Second toast' })
    })

    render(<ToastContainer />)
    expect(screen.getByText('First toast')).toBeInTheDocument()
    expect(screen.getByText('Second toast')).toBeInTheDocument()
  })
})

describe('useToast store', () => {
  beforeEach(() => {
    useToast.setState({ toasts: [] })
  })

  it('should add a toast with unique id', () => {
    useToast.getState().addToast({ type: 'success', message: 'Test message' })
    const toasts = useToast.getState().toasts

    expect(toasts).toHaveLength(1)
    expect(toasts[0].id).toBeDefined()
    expect(toasts[0].type).toBe('success')
    expect(toasts[0].message).toBe('Test message')
  })

  it('should remove a toast by id', () => {
    useToast.getState().addToast({ type: 'success', message: 'To remove', duration: 0 })
    const id = useToast.getState().toasts[0].id

    useToast.getState().removeToast(id)
    expect(useToast.getState().toasts).toHaveLength(0)
  })

  it('should not affect other toasts when removing one', () => {
    useToast.getState().addToast({ type: 'success', message: 'First', duration: 0 })
    useToast.getState().addToast({ type: 'error', message: 'Second', duration: 0 })

    const firstId = useToast.getState().toasts[0].id
    useToast.getState().removeToast(firstId)

    expect(useToast.getState().toasts).toHaveLength(1)
    expect(useToast.getState().toasts[0].message).toBe('Second')
  })
})
