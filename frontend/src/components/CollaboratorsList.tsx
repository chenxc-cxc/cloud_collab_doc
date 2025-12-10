'use client'

import type { Collaborator } from '@/types'

interface CollaboratorsListProps {
    collaborators: Collaborator[]
}

export default function CollaboratorsList({ collaborators }: CollaboratorsListProps) {
    if (collaborators.length === 0) {
        return null
    }

    return (
        <div className="flex items-center -space-x-2">
            {collaborators.slice(0, 5).map((collaborator) => (
                <div
                    key={collaborator.userId}
                    title={collaborator.name}
                    className="relative"
                >
                    <div
                        className="w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium ring-2 ring-white dark:ring-slate-900"
                        style={{ backgroundColor: collaborator.color }}
                    >
                        {collaborator.name.charAt(0).toUpperCase()}
                    </div>
                    {/* Online indicator */}
                    <span className="absolute bottom-0 right-0 w-2.5 h-2.5 bg-green-500 border-2 border-white dark:border-slate-900 rounded-full" />
                </div>
            ))}

            {collaborators.length > 5 && (
                <div className="w-8 h-8 rounded-full bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-xs font-medium text-slate-600 dark:text-slate-400 ring-2 ring-white dark:ring-slate-900">
                    +{collaborators.length - 5}
                </div>
            )}
        </div>
    )
}
