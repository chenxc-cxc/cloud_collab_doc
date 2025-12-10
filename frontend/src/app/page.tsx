'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { FileText, Plus, Users, Clock, ChevronRight, Trash2 } from 'lucide-react'
import { useStore } from '@/lib/store'
import { api } from '@/lib/api'
import { AuthGuard } from '@/components/AuthGuard'
import { UserMenu } from '@/components/UserMenu'
import NotificationBell from '@/components/NotificationBell'
import type { Document } from '@/types'

export default function HomePage() {
    const router = useRouter()
    const { currentUser, setCurrentUser } = useStore()
    const [documents, setDocuments] = useState<Document[]>([])
    const [loading, setLoading] = useState(true)
    const [creating, setCreating] = useState(false)

    useEffect(() => {
        loadData()
    }, [])

    const loadData = async () => {
        try {
            const [user, docs] = await Promise.all([
                api.getCurrentUser(),
                api.listDocuments()
            ])
            setCurrentUser(user)
            setDocuments(docs)
        } catch (error) {
            console.error('Failed to load data:', error)
        } finally {
            setLoading(false)
        }
    }

    const createDocument = async () => {
        setCreating(true)
        try {
            const doc = await api.createDocument('Untitled Document')
            router.push(`/doc/${doc.id}`)
        } catch (error) {
            console.error('Failed to create document:', error)
            setCreating(false)
        }
    }

    const deleteDocument = async (e: React.MouseEvent, docId: string) => {
        e.stopPropagation()
        if (!confirm('Are you sure you want to delete this document?')) return

        try {
            await api.deleteDocument(docId)
            setDocuments(docs => docs.filter(d => d.id !== docId))
        } catch (error) {
            console.error('Failed to delete document:', error)
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

    return (
        <AuthGuard>
            <div className="max-w-6xl mx-auto px-4 py-8">
                {/* Header */}
                <header className="flex items-center justify-between mb-8">
                    <div>
                        <h1 className="text-3xl font-bold text-slate-900 dark:text-white">
                            Collaborative Docs
                        </h1>
                        <p className="text-slate-500 dark:text-slate-400 mt-1">
                            Welcome back, {currentUser?.name || 'User'}
                        </p>
                    </div>

                    <div className="flex items-center gap-3">
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

                {/* Document List */}
                <div className="space-y-4">
                    <h2 className="text-lg font-semibold text-slate-700 dark:text-slate-300">
                        Your Documents
                    </h2>

                    {documents.length === 0 ? (
                        <div className="text-center py-16 bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700">
                            <FileText className="w-16 h-16 mx-auto text-slate-300 dark:text-slate-600 mb-4" />
                            <h3 className="text-lg font-medium text-slate-600 dark:text-slate-400">
                                No documents yet
                            </h3>
                            <p className="text-slate-500 dark:text-slate-500 mt-1">
                                Create your first document to get started
                            </p>
                        </div>
                    ) : (
                        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
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
                                                    onClick={(e) => deleteDocument(e, doc.id)}
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
        </AuthGuard>
    )
}
