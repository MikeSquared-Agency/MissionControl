import { useState } from 'react'
import { useToast, type Toast as ToastType, type ToastType as ToastVariant } from '../stores/useToast'

export function ToastContainer() {
  const toasts = useToast((s) => s.toasts)

  if (toasts.length === 0) return null

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} />
      ))}
    </div>
  )
}

interface ToastItemProps {
  toast: ToastType
}

function ToastItem({ toast }: ToastItemProps) {
  const removeToast = useToast((s) => s.removeToast)
  const [isExiting, setIsExiting] = useState(false)

  const handleClose = () => {
    setIsExiting(true)
    setTimeout(() => {
      removeToast(toast.id)
    }, 150)
  }

  const icons: Record<ToastVariant, string> = {
    success: '✓',
    error: '✕',
    info: 'ℹ',
    warning: '⚠'
  }

  const styles: Record<ToastVariant, string> = {
    success: 'bg-green-500/10 border-green-500/30 text-green-400',
    error: 'bg-red-500/10 border-red-500/30 text-red-400',
    info: 'bg-blue-500/10 border-blue-500/30 text-blue-400',
    warning: 'bg-amber-500/10 border-amber-500/30 text-amber-400'
  }

  const iconStyles: Record<ToastVariant, string> = {
    success: 'bg-green-500/20 text-green-400',
    error: 'bg-red-500/20 text-red-400',
    info: 'bg-blue-500/20 text-blue-400',
    warning: 'bg-amber-500/20 text-amber-400'
  }

  return (
    <div
      className={`
        pointer-events-auto flex items-center gap-3 px-4 py-3
        bg-gray-900 border rounded-lg shadow-lg
        transition-all duration-150
        ${styles[toast.type]}
        ${isExiting ? 'opacity-0 translate-x-4' : 'opacity-100 translate-x-0'}
      `}
      style={{
        animation: isExiting ? undefined : 'slideIn 0.15s ease-out'
      }}
    >
      {/* Icon */}
      <div className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold ${iconStyles[toast.type]}`}>
        {icons[toast.type]}
      </div>

      {/* Message */}
      <p className="text-sm text-gray-200 flex-1">{toast.message}</p>

      {/* Close button */}
      <button
        onClick={handleClose}
        className="p-1 text-gray-500 hover:text-gray-300 transition-colors"
      >
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  )
}

// Add the animation to index.css or include it inline
// @keyframes slideIn {
//   from { opacity: 0; transform: translateX(1rem); }
//   to { opacity: 1; transform: translateX(0); }
// }
