import { toast as sonnerToast } from "sonner"

type ToastOptions = {
  title?: string
  description?: string
  variant?: 'default' | 'destructive'
}

// Module-level so its identity is stable: components can list it as a hook
// dependency without re-running effects on every render.
function toast({ title, description, variant }: ToastOptions) {
  if (variant === 'destructive') {
    sonnerToast.error(title, {
      description,
    })
  } else {
    sonnerToast.success(title, {
      description,
    })
  }
}

export function useToast() {
  return { toast }
}
