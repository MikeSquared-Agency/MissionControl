import { useEffect, useRef, useState } from 'react'

export interface MenuItem {
  label?: string
  icon?: string
  onClick?: () => void
  danger?: boolean
  disabled?: boolean
  divider?: boolean
}

interface ContextMenuProps {
  open: boolean
  items: MenuItem[]
  onClose: () => void
  position: { x: number; y: number }
}

export function ContextMenu({ open, items, onClose, position }: ContextMenuProps) {
  if (!open) return null
  const menuRef = useRef<HTMLDivElement>(null)
  const [adjustedPosition, setAdjustedPosition] = useState(position)

  // Adjust position to stay within viewport
  useEffect(() => {
    if (!menuRef.current) return

    const rect = menuRef.current.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    let x = position.x
    let y = position.y

    // Adjust horizontal position
    if (x + rect.width > viewportWidth - 8) {
      x = viewportWidth - rect.width - 8
    }
    if (x < 8) x = 8

    // Adjust vertical position
    if (y + rect.height > viewportHeight - 8) {
      y = viewportHeight - rect.height - 8
    }
    if (y < 8) y = 8

    setAdjustedPosition({ x, y })
  }, [position])

  // Close on escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose()
      }
    }

    // Delay to prevent immediate close from the click that opened the menu
    const timer = setTimeout(() => {
      document.addEventListener('click', handleClickOutside)
    }, 0)

    return () => {
      clearTimeout(timer)
      document.removeEventListener('click', handleClickOutside)
    }
  }, [onClose])

  return (
    <div
      ref={menuRef}
      className="fixed z-50 min-w-[160px] py-1 bg-gray-800 border border-gray-700/50 rounded-lg shadow-xl"
      style={{
        left: adjustedPosition.x,
        top: adjustedPosition.y
      }}
    >
      {items.map((item, index) => {
        if (item.divider) {
          return (
            <div key={index} className="my-1 border-t border-gray-700/50" />
          )
        }

        return (
          <button
            key={index}
            onClick={() => {
              if (!item.disabled && item.onClick) {
                item.onClick()
                onClose()
              }
            }}
            disabled={item.disabled}
            className={`
              w-full flex items-center gap-2 px-3 py-1.5 text-xs text-left transition-colors
              ${item.disabled
                ? 'text-gray-600 cursor-not-allowed'
                : item.danger
                  ? 'text-red-400 hover:bg-red-500/10'
                  : 'text-gray-300 hover:bg-gray-700/50'
              }
            `}
          >
            {item.icon && (
              <span className="w-4 text-center">{item.icon}</span>
            )}
            <span>{item.label}</span>
          </button>
        )
      })}
    </div>
  )
}

// Hook for managing context menu state
export function useContextMenu() {
  const [menu, setMenu] = useState<{
    items: MenuItem[]
    position: { x: number; y: number }
  } | null>(null)

  const openMenu = (items: MenuItem[], event: React.MouseEvent) => {
    event.preventDefault()
    event.stopPropagation()
    setMenu({
      items,
      position: { x: event.clientX, y: event.clientY }
    })
  }

  const closeMenu = () => setMenu(null)

  return { menu, openMenu, closeMenu }
}
