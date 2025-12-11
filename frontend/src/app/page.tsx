'use client'

import { useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { FileText, Plus, Users, Clock, ChevronRight, Trash2, FolderPlus, Home, AlertTriangle, X, FolderTree } from 'lucide-react'
import { useStore } from '@/lib/store'
import { api } from '@/lib/api'
import { AuthGuard } from '@/components/AuthGuard'
import { UserMenu } from '@/components/UserMenu'
import NotificationBell from '@/components/NotificationBell'
import { FolderItem } from '@/components/FolderItem'
import FolderTreeSidebar from '@/components/FolderTreeSidebar'
import type { Document, Folder, FolderContents } from '@/types'

export default function HomePage() {
    const router = useRouter()
    const searchParams = useSearchParams()
    const { currentUser, setCurrentUser } = useStore()
    const [contents, setContents] = useState<FolderContents | null>(null)
    const [loading, setLoading] = useState(true)
    const [creating, setCreating] = useState(false)
    const [creatingFolder, setCreatingFolder] = useState(false)
    const [currentFolderId, setCurrentFolderId] = useState<string | undefined>(undefined)
    const [breadcrumbs, setBreadcrumbs] = useState<Folder[]>([])
    const [showNewFolderInput, setShowNewFolderInput] = useState(false)
    const [newFolderName, setNewFolderName] = useState('')
    const [initialized, setInitialized] = useState(false)
    const [deleteModalOpen, setDeleteModalOpen] = useState(false)
    const [documentToDelete, setDocumentToDelete] = useState<Document | null>(null)
    const [deleting, setDeleting] = useState(false)
    const [showFolderTree, setShowFolderTree] = useState(true)

    // Read folder from URL on initial load
    useEffect(() => {
        const folderFromUrl = searchParams.get('folder')
        if (folderFromUrl && !initialized) {
            setCurrentFolderId(folderFromUrl)
            // Build breadcrumbs for the folder - we'll load it with the folder data
        }
        setInitialized(true)
    }, [searchParams, initialized])

    useEffect(() => {
        if (initialized) {
            loadData()
        }
    }, [currentFolderId, initialized])

    const loadData = async () => {
        try {
            setLoading(true)
            const [user, folderContents] = await Promise.all([
                api.getCurrentUser(),
                api.getFolderContents(currentFolderId)
            ])
            setCurrentUser(user)
            setContents(folderContents)

            // If we have a folder from URL but no breadcrumbs, load the full path
            if (currentFolderId && breadcrumbs.length === 0) {
                try {
                    const path = await api.getFolderPath(currentFolderId)
                    setBreadcrumbs(path)
                } catch (e) {
                    // Fallback to just the current folder if path fails
                    if (folderContents.folder) {
                        setBreadcrumbs([folderContents.folder])
                    }
                }
            }
        } catch (error) {
            console.error('Failed to load data:', error)
        } finally {
            setLoading(false)
        }
    }

    const navigateToFolder = async (folder: Folder | null) => {
        if (folder === null) {
            // Navigate to root
            setCurrentFolderId(undefined)
            setBreadcrumbs([])
            router.replace('/', { scroll: false })
        } else {
            setCurrentFolderId(folder.id)
            router.replace(`/?folder=${folder.id}`, { scroll: false })
            // Update breadcrumbs
            const existingIndex = breadcrumbs.findIndex(b => b.id === folder.id)
            if (existingIndex >= 0) {
                // Navigate back in breadcrumbs
                setBreadcrumbs(breadcrumbs.slice(0, existingIndex + 1))
            } else {
                // Add to breadcrumbs
                setBreadcrumbs([...breadcrumbs, folder])
            }
        }
    }

    // Handle folder selection from the tree sidebar
    const handleFolderTreeSelect = async (folderId?: string) => {
        if (!folderId) {
            // Navigate to root
            setCurrentFolderId(undefined)
            setBreadcrumbs([])
            router.replace('/', { scroll: false })
        } else {
            setCurrentFolderId(folderId)
            router.replace(`/?folder=${folderId}`, { scroll: false })
            // Load the full path for breadcrumbs
            try {
                const path = await api.getFolderPath(folderId)
                setBreadcrumbs(path)
            } catch (e) {
                console.error('Failed to load folder path:', e)
            }
        }
    }

    const createDocument = async () => {
        setCreating(true)
        try {
            const doc = await api.createDocument('Untitled Document')
            // If we're in a folder, move the document there
            if (currentFolderId) {
                await api.moveDocument(doc.id, currentFolderId)
            }
            router.push(`/doc/${doc.id}`)
        } catch (error) {
            console.error('Failed to create document:', error)
            setCreating(false)
        }
    }

    const createFolder = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!newFolderName.trim()) return

        setCreatingFolder(true)
        try {
            await api.createFolder(newFolderName.trim(), currentFolderId)
            setNewFolderName('')
            setShowNewFolderInput(false)
            await loadData()
        } catch (error) {
            console.error('Failed to create folder:', error)
        } finally {
            setCreatingFolder(false)
        }
    }

    const openDeleteModal = (e: React.MouseEvent, doc: Document) => {
        e.stopPropagation()
        setDocumentToDelete(doc)
        setDeleteModalOpen(true)
    }

    const closeDeleteModal = () => {
        setDeleteModalOpen(false)
        setDocumentToDelete(null)
    }

    const confirmDelete = async () => {
        if (!documentToDelete) return

        setDeleting(true)
        try {
            await api.deleteDocument(documentToDelete.id)
            closeDeleteModal()
            await loadData()
        } catch (error) {
            console.error('Failed to delete document:', error)
        } finally {
            setDeleting(false)
        }
    }

    const formatDate = (date: string) => {
        return new Date(date).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        })
    }

    const getRoleColor = (permission: string) => {
        switch (permission) {
            case 'owner': return 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300'
            case 'edit': return 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300'
            case 'comment': return 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
            default: return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
        }
    }

    if (loading) {
        return (
            <AuthGuard>
                <div className="flex items-center justify-center min-h-screen">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500"></div>
                </div>
            </AuthGuard>
        )
    }

    const folders = contents?.folders || []
    const documents = contents?.documents || []

    return (
        <AuthGuard>
            {/* Folder Tree Sidebar */}
            <FolderTreeSidebar
                isOpen={showFolderTree}
                onToggle={() => setShowFolderTree(!showFolderTree)}
                currentFolderId={currentFolderId}
                onFolderSelect={handleFolderTreeSelect}
            />

            <div className={`min-h-screen transition-all duration-300 ${showFolderTree ? 'ml-64' : 'ml-0'}`}>
                <div className="max-w-6xl mx-auto px-4 py-8">
                    {/* Header */}
                    <header className="flex items-center justify-between mb-8">
                        <div className="flex items-center gap-4">
                            {/* Folder tree toggle */}
                            <button
                                onClick={() => setShowFolderTree(!showFolderTree)}
                                className={`p-2 rounded-lg transition-colors ${showFolderTree
                                    ? 'bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400'
                                    : 'hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-400'
                                    }`}
                                title={showFolderTree ? '隐藏文件夹树' : '显示文件夹树'}
                            >
                                <FolderTree className="w-5 h-5" />
                            </button>
                            <div>
                                <h1 className="text-3xl font-bold text-slate-900 dark:text-white">
                                    Collaborative Docs
                                </h1>
                                <p className="text-slate-500 dark:text-slate-400 mt-1">
                                    Welcome back, {currentUser?.name || 'User'}
                                </p>
                            </div>
                        </div>

                        <div className="flex items-center gap-3">
                            <button
                                onClick={() => setShowNewFolderInput(true)}
                                className="flex items-center gap-2 px-4 py-2 bg-amber-500 hover:bg-amber-600 text-white rounded-lg font-medium transition-colors"
                            >
                                <FolderPlus className="w-5 h-5" />
                                New Folder
                            </button>
                            <button
                                onClick={createDocument}
                                disabled={creating}
                                className="flex items-center gap-2 px-4 py-2 bg-primary-500 hover:bg-primary-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
                            >
                                <Plus className="w-5 h-5" />
                                {creating ? 'Creating...' : 'New Document'}
                            </button>
                            <NotificationBell />
                            <UserMenu />
                        </div>
                    </header>

                    {/* Breadcrumbs */}
                    <nav className="flex items-center gap-2 mb-6 text-sm">
                        <button
                            onClick={() => navigateToFolder(null)}
                            className={`flex items-center gap-1 px-2 py-1 rounded-md transition-colors ${!currentFolderId
                                ? 'text-primary-600 dark:text-primary-400 font-medium'
                                : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                                }`}
                        >
                            <Home className="w-4 h-4" />
                            Home
                        </button>
                        {breadcrumbs.map((folder, index) => (
                            <div key={folder.id} className="flex items-center gap-2">
                                <ChevronRight className="w-4 h-4 text-slate-400" />
                                <button
                                    onClick={() => navigateToFolder(folder)}
                                    className={`px-2 py-1 rounded-md transition-colors ${index === breadcrumbs.length - 1
                                        ? 'text-primary-600 dark:text-primary-400 font-medium'
                                        : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                                        }`}
                                >
                                    {folder.name}
                                </button>
                            </div>
                        ))}
                    </nav>

                    {/* New Folder Input */}
                    {showNewFolderInput && (
                        <div className="mb-6 bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 animate-fade-in">
                            <form onSubmit={createFolder} className="flex items-center gap-3">
                                <FolderPlus className="w-6 h-6 text-amber-500" />
                                <input
                                    type="text"
                                    value={newFolderName}
                                    onChange={(e) => setNewFolderName(e.target.value)}
                                    placeholder="Enter folder name..."
                                    className="flex-1 px-3 py-2 bg-slate-50 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                                    autoFocus
                                />
                                <button
                                    type="submit"
                                    disabled={creatingFolder || !newFolderName.trim()}
                                    className="px-4 py-2 bg-amber-500 hover:bg-amber-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
                                >
                                    {creatingFolder ? 'Creating...' : 'Create'}
                                </button>
                                <button
                                    type="button"
                                    onClick={() => {
                                        setShowNewFolderInput(false)
                                        setNewFolderName('')
                                    }}
                                    className="px-4 py-2 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
                                >
                                    Cancel
                                </button>
                            </form>
                        </div>
                    )}

                    {/* Content List */}
                    <div className="space-y-4">
                        <h2 className="text-lg font-semibold text-slate-700 dark:text-slate-300">
                            {currentFolderId ? contents?.folder?.name || 'Folder' : 'Your Documents'}
                        </h2>

                        {folders.length === 0 && documents.length === 0 ? (
                            <div className="text-center py-16 bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700">
                                <FileText className="w-16 h-16 mx-auto text-slate-300 dark:text-slate-600 mb-4" />
                                <h3 className="text-lg font-medium text-slate-600 dark:text-slate-400">
                                    {currentFolderId ? 'This folder is empty' : 'No documents yet'}
                                </h3>
                                <p className="text-slate-500 dark:text-slate-500 mt-1">
                                    Create your first {currentFolderId ? 'item' : 'document'} to get started
                                </p>
                            </div>
                        ) : (
                            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                                {/* Folders first */}
                                {folders.map((folder) => (
                                    <FolderItem
                                        key={folder.id}
                                        folder={folder}
                                        onClick={() => navigateToFolder(folder)}
                                        onUpdated={loadData}
                                    />
                                ))}

                                {/* Then documents */}
                                {documents.map((doc) => (
                                    <div
                                        key={doc.id}
                                        onClick={() => router.push(`/doc/${doc.id}`)}
                                        className="group bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-5 cursor-pointer hover:border-primary-300 dark:hover:border-primary-600 hover:shadow-lg transition-all animate-fade-in"
                                    >
                                        <div className="flex items-start justify-between mb-3">
                                            <div className="p-2 bg-primary-50 dark:bg-primary-900/30 rounded-lg">
                                                <FileText className="w-6 h-6 text-primary-500" />
                                            </div>
                                            <div className="flex items-center gap-2">
                                                <span className={`px-2 py-1 rounded-full text-xs font-medium ${getRoleColor(doc.permission || 'view')}`}>
                                                    {doc.permission}
                                                </span>
                                                {doc.permission === 'owner' && (
                                                    <button
                                                        onClick={(e) => openDeleteModal(e, doc)}
                                                        className="p-1 text-slate-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity"
                                                    >
                                                        <Trash2 className="w-4 h-4" />
                                                    </button>
                                                )}
                                            </div>
                                        </div>

                                        <h3 className="font-semibold text-slate-900 dark:text-white mb-2 truncate">
                                            {doc.title}
                                        </h3>

                                        <div className="flex items-center justify-between text-sm text-slate-500 dark:text-slate-400">
                                            <div className="flex items-center gap-1">
                                                <Users className="w-4 h-4" />
                                                <span>{doc.owner?.name || 'Unknown'}</span>
                                            </div>
                                            <div className="flex items-center gap-1">
                                                <Clock className="w-4 h-4" />
                                                <span>{formatDate(doc.updated_at)}</span>
                                            </div>
                                        </div>

                                        <div className="mt-4 flex items-center text-primary-500 text-sm font-medium opacity-0 group-hover:opacity-100 transition-opacity">
                                            Open document
                                            <ChevronRight className="w-4 h-4 ml-1" />
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </div>

                {/* Delete Confirmation Modal */}
                {deleteModalOpen && documentToDelete && (
                    <div className="fixed inset-0 z-50 flex items-center justify-center">
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
                                确认删除文档
                            </h3>

                            {/* Description */}
                            <p className="text-center text-slate-500 dark:text-slate-400 mb-6">
                                此操作无法撤销，文档将被永久删除。
                            </p>

                            {/* Document Info */}
                            <div className="bg-slate-50 dark:bg-slate-700/50 rounded-xl p-4 mb-6">
                                <div className="flex items-start gap-3">
                                    <div className="p-2 bg-primary-50 dark:bg-primary-900/30 rounded-lg flex-shrink-0">
                                        <FileText className="w-5 h-5 text-primary-500" />
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <h4 className="font-medium text-slate-900 dark:text-white truncate">
                                            {documentToDelete.title}
                                        </h4>
                                        <div className="flex items-center gap-4 mt-1 text-sm text-slate-500 dark:text-slate-400">
                                            <span className="flex items-center gap-1">
                                                <Users className="w-3.5 h-3.5" />
                                                {documentToDelete.owner?.name || 'Unknown'}
                                            </span>
                                            <span className="flex items-center gap-1">
                                                <Clock className="w-3.5 h-3.5" />
                                                {formatDate(documentToDelete.updated_at)}
                                            </span>
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
                                    取消
                                </button>
                                <button
                                    onClick={confirmDelete}
                                    disabled={deleting}
                                    className="flex-1 px-4 py-2.5 bg-red-500 hover:bg-red-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                                >
                                    {deleting ? (
                                        <>
                                            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                                            删除中...
                                        </>
                                    ) : (
                                        <>
                                            <Trash2 className="w-4 h-4" />
                                            确认删除
                                        </>
                                    )}
                                </button>
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </AuthGuard>
    )
}
