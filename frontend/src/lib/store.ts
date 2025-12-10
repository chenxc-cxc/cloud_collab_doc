import { create } from 'zustand'
import type { User, Collaborator } from '@/types'

interface StoreState {
    // User state
    currentUser: User | null
    setCurrentUser: (user: User | null) => void

    // Collaboration state
    collaborators: Collaborator[]
    setCollaborators: (collaborators: Collaborator[]) => void
    addCollaborator: (collaborator: Collaborator) => void
    removeCollaborator: (userId: string) => void
    updateCollaborator: (userId: string, updates: Partial<Collaborator>) => void

    // Connection state
    isConnected: boolean
    setIsConnected: (connected: boolean) => void

    // UI state
    sidebarOpen: boolean
    setSidebarOpen: (open: boolean) => void
}

export const useStore = create<StoreState>((set) => ({
    // User state
    currentUser: null,
    setCurrentUser: (user) => set({ currentUser: user }),

    // Collaboration state
    collaborators: [],
    setCollaborators: (collaborators) => set({ collaborators }),
    addCollaborator: (collaborator) =>
        set((state) => ({
            collaborators: [...state.collaborators.filter(c => c.userId !== collaborator.userId), collaborator]
        })),
    removeCollaborator: (userId) =>
        set((state) => ({
            collaborators: state.collaborators.filter(c => c.userId !== userId)
        })),
    updateCollaborator: (userId, updates) =>
        set((state) => ({
            collaborators: state.collaborators.map(c =>
                c.userId === userId ? { ...c, ...updates } : c
            )
        })),

    // Connection state
    isConnected: false,
    setIsConnected: (connected) => set({ isConnected: connected }),

    // UI state
    sidebarOpen: true,
    setSidebarOpen: (open) => set({ sidebarOpen: open }),
}))

// Generate a random color for cursor
export function getRandomColor(): string {
    const colors = [
        '#F44336', '#E91E63', '#9C27B0', '#673AB7',
        '#3F51B5', '#2196F3', '#03A9F4', '#00BCD4',
        '#009688', '#4CAF50', '#8BC34A', '#CDDC39',
        '#FFC107', '#FF9800', '#FF5722', '#795548'
    ]
    return colors[Math.floor(Math.random() * colors.length)]
}
