import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

interface UIState {
    viewMode: 'admin' | 'user';
    setViewMode: (mode: 'admin' | 'user') => void;
}

export const useUIStore = create<UIState>()(
    persist(
        (set) => ({
            viewMode: 'admin', // Default preference
            setViewMode: (mode) => set({ viewMode: mode }),
        }),
        {
            name: 'ui-storage',
            storage: createJSONStorage(() => localStorage),
        }
    )
);
