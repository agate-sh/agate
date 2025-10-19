import { useState, useCallback } from 'react';

/**
 * Dialog state management hook
 * Provides a simple stack-based dialog system for overlays
 */
export function useDialog() {
  const [dialogs, setDialogs] = useState<React.ReactNode[]>([]);

  const push = useCallback((dialog: React.ReactNode) => {
    setDialogs((prev) => [...prev, dialog]);
  }, []);

  const pop = useCallback(() => {
    setDialogs((prev) => prev.slice(0, -1));
  }, []);

  const clear = useCallback(() => {
    setDialogs([]);
  }, []);

  return {
    dialogs,
    push,
    pop,
    clear,
    isOpen: dialogs.length > 0,
    current: dialogs[dialogs.length - 1] || null,
  };
}

export type DialogContext = ReturnType<typeof useDialog>;
