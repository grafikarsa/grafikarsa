import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

interface UIState {
    viewMode: 'admin' | 'user';
    setViewMode: (mode: 'admin' | 'user') => void;
    sidebarCollapsed: boolean;
    setSidebarCollapsed: (collapsed: boolean) => void;
    toggleSidebar: () => void;
}

export const useUIStore = create<UIState>()(
    persist(
        (set) => ({
            viewMode: 'admin', // Default preference
            setViewMode: (mode) => set({ viewMode: mode }),
            sidebarCollapsed: false,
            setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
            toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
        }),
        {
            name: 'ui-storage',
            storage: createJSONStorage(() => localStorage),
        }
    )
);
