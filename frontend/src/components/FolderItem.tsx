'use client'

import { useState } from 'react'
import { Folder as FolderIcon, MoreHorizontal, Edit2, Trash2, ChevronRight, AlertTriangle, X, Clock } from 'lucide-react'
import type { Folder } from '@/types'
import { api } from '@/lib/api'

interface FolderItemProps {
    folder: Folder
    onClick: () => void
    onUpdated: () => void
}

export function FolderItem({ folder, onClick, onUpdated }: FolderItemProps) {
    const [isMenuOpen, setIsMenuOpen] = useState(false)
    const [isRenaming, setIsRenaming] = useState(false)
    const [newName, setNewName] = useState(folder.name)
    const [isDeleting, setIsDeleting] = useState(false)
    const [showDeleteModal, setShowDeleteModal] = useState(false)
    const [isSaving, setIsSaving] = useState(false)

    const handleRename = async (e?: React.FormEvent) => {
        if (e) e.preventDefault()
        if (isSaving) return
        if (!newName.trim() || newName === folder.name) {
            setIsRenaming(false)
            setNewName(folder.name)
            return
        }
        setIsSaving(true)
        try {
            await api.updateFolder(folder.id, newName.trim())
            setIsRenaming(false)
            onUpdated()
        } catch (error) {
            console.error('Failed to rename folder:', error)
            setNewName(folder.name)
            setIsRenaming(false)
        } finally {
            setIsSaving(false)
        }
    }

    const handleBlur = () => {
        if (isSaving) return
        // Use a small delay to allow form submission to complete
        setTimeout(() => {
            if (!isSaving) {
                setIsRenaming(false)
                setNewName(folder.name)
            }
        }, 100)
    }

    const openDeleteModal = (e: React.MouseEvent) => {
        e.stopPropagation()
        setIsMenuOpen(false)
        setShowDeleteModal(true)
    }

    const closeDeleteModal = () => {
        setShowDeleteModal(false)
    }

    const confirmDelete = async () => {
        setIsDeleting(true)
        try {
            await api.deleteFolder(folder.id)
            closeDeleteModal()
            onUpdated()
        } catch (error) {
            console.error('Failed to delete folder:', error)
            setIsDeleting(false)
        }
    }

    const formatDate = (date: string) => {
        return new Date(date).toLocaleDateString('en-US', {
            month: 'short',
            day: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            hour12: false
        })
    }

    if (isRenaming) {
        return (
            <div className="group bg-white dark:bg-slate-800 rounded-xl border border-primary-300 dark:border-primary-600 p-5">
                <form onSubmit={handleRename} className="flex items-center gap-2">
                    <div className="p-2 bg-amber-50 dark:bg-amber-900/30 rounded-lg">
                        <FolderIcon className="w-6 h-6 text-amber-500" />
                    </div>
                    <input
                        type="text"
                        value={newName}
                        onChange={(e) => setNewName(e.target.value)}
                        className="flex-1 px-3 py-1.5 bg-slate-50 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                        autoFocus
                        onBlur={handleBlur}
                        disabled={isSaving}
                    />
                    <button
                        type="submit"
                        disabled={isSaving || !newName.trim()}
                        className="px-3 py-1.5 bg-primary-500 hover:bg-primary-600 text-white text-sm rounded-lg font-medium transition-colors disabled:opacity-50"
                    >
                        {isSaving ? 'Saving...' : 'Saved'}
                    </button>
                    <button
                        type="button"
                        onClick={() => {
                            setIsRenaming(false)
                            setNewName(folder.name)
                        }}
                        disabled={isSaving}
                        className="px-3 py-1.5 text-slate-500 hover:text-slate-700 text-sm font-medium"
                    >
                        取消
                    </button>
                </form>
            </div>
        )
    }

    return (
        <>
            <div
                onClick={onClick}
                className={`group bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-5 cursor-pointer hover:border-amber-300 dark:hover:border-amber-600 hover:shadow-lg transition-all animate-fade-in ${isDeleting ? 'opacity-50 pointer-events-none' : ''}`}
            >
                <div className="flex items-start justify-between mb-3">
                    <div className="p-2 bg-amber-50 dark:bg-amber-900/30 rounded-lg">
                        <FolderIcon className="w-6 h-6 text-amber-500" />
                    </div>
                    <div className="relative">
                        <button
                            onClick={(e) => {
                                e.stopPropagation()
                                setIsMenuOpen(!isMenuOpen)
                            }}
                            className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity"
                        >
                            <MoreHorizontal className="w-5 h-5" />
                        </button>

                        {isMenuOpen && (
                            <>
                                <div
                                    className="fixed inset-0 z-10"
                                    onClick={(e) => {
                                        e.stopPropagation()
                                        setIsMenuOpen(false)
                                    }}
                                />
                                <div className="absolute right-0 top-8 z-20 w-36 bg-white dark:bg-slate-800 rounded-lg shadow-lg border border-slate-200 dark:border-slate-700 py-1">
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation()
                                            setIsMenuOpen(false)
                                            setIsRenaming(true)
                                        }}
                                        className="flex items-center gap-2 w-full px-3 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                    >
                                        <Edit2 className="w-4 h-4" />
                                        Rename
                                    </button>
                                    <button
                                        onClick={openDeleteModal}
                                        className="flex items-center gap-2 w-full px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-slate-100 dark:hover:bg-slate-700"
                                    >
                                        <Trash2 className="w-4 h-4" />
                                        Delete
                                    </button>
                                </div>
                            </>
                        )}
                    </div>
                </div>

                <h3 className="font-semibold text-slate-900 dark:text-white mb-2 truncate">
                    {folder.name}
                </h3>

                <div className="flex items-center justify-between text-sm text-slate-500 dark:text-slate-400">
                    <span>Folder</span>
                    <span>{formatDate(folder.updated_at)}</span>
                </div>

                <div className="mt-4 flex items-center text-amber-500 text-sm font-medium opacity-0 group-hover:opacity-100 transition-opacity">
                    Open folder
                    <ChevronRight className="w-4 h-4 ml-1" />
                </div>
            </div>

            {/* Delete Confirmation Modal */}
            {
                showDeleteModal && (
                    <div className="fixed inset-0 z-50 flex items-center justify-center" onClick={(e) => e.stopPropagation()}>
                        {/* Backdrop */}
                        <div
                            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
                            onClick={closeDeleteModal}
                        />

                        {/* Modal */}
                        <div className="relative bg-white dark:bg-slate-800 rounded-2xl shadow-2xl w-full max-w-md mx-4 p-6 animate-fade-in">
                            {/* Close button */}
                            <button
                                onClick={closeDeleteModal}
                                className="absolute top-4 right-4 p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
                            >
                                <X className="w-5 h-5" />
                            </button>

                            {/* Warning Icon */}
                            <div className="flex items-center justify-center w-12 h-12 mx-auto mb-4 bg-red-100 dark:bg-red-900/30 rounded-full">
                                <AlertTriangle className="w-6 h-6 text-red-500" />
                            </div>

                            {/* Title */}
                            <h3 className="text-xl font-semibold text-center text-slate-900 dark:text-white mb-2">
                                Confirm deletion of the folder
                            </h3>

                            {/* Description */}
                            <p className="text-center text-slate-500 dark:text-slate-400 mb-6">
                                <span className="text-red-500 font-medium">This action cannot be undone!</span>
                                <br />
                                All documents and subfolders within the folder will be permanently deleted.
                            </p>

                            {/* Folder Info */}
                            <div className="bg-slate-50 dark:bg-slate-700/50 rounded-xl p-4 mb-6">
                                <div className="flex items-start gap-3">
                                    <div className="p-2 bg-amber-50 dark:bg-amber-900/30 rounded-lg flex-shrink-0">
                                        <FolderIcon className="w-5 h-5 text-amber-500" />
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <h4 className="font-medium text-slate-900 dark:text-white truncate">
                                            {folder.name}
                                        </h4>
                                        <div className="flex items-center gap-1 mt-1 text-sm text-slate-500 dark:text-slate-400">
                                            <Clock className="w-3.5 h-3.5" />
                                            {formatDate(folder.updated_at)}
                                        </div>
                                    </div>
                                </div>
                            </div>

                            {/* Actions */}
                            <div className="flex gap-3">
                                <button
                                    onClick={closeDeleteModal}
                                    className="flex-1 px-4 py-2.5 bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg font-medium hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={confirmDelete}
                                    disabled={isDeleting}
                                    className="flex-1 px-4 py-2.5 bg-red-500 hover:bg-red-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                                >
                                    {isDeleting ? (
                                        <>
                                            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                                            Deleting...
                                        </>
                                    ) : (
                                        <>
                                            <Trash2 className="w-4 h-4" />
                                            Confirm Deletion
                                        </>
                                    )}
                                </button>
                            </div>
                        </div>
                    </div>
                )
            }
        </>
    )
}
