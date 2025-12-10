'use client'

import { useEffect, useState, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { ArrowLeft, Users, Share2, MessageSquare, History, Settings, Lock, FileQuestion, Send, Check, Loader2 } from 'lucide-react'
import { useStore } from '@/lib/store'
import { api } from '@/lib/api'
import { useCollaboration } from '@/hooks/useCollaboration'
import Editor from '@/components/Editor'
import Toolbar from '@/components/Toolbar'
import CollaboratorsList from '@/components/CollaboratorsList'
import CommentsPanel from '@/components/CommentsPanel'
import ShareModal from '@/components/ShareModal'
import type { Document, Comment } from '@/types'

export default function DocumentPage() {
    const params = useParams()
    const router = useRouter()
    const docId = params.id as string

    // Compute back URL based on document's folder
    const getBackUrl = () => {
        if (document?.folder_id) {
            return `/?folder=${document.folder_id}`
        }
        return '/'
    }

    const { currentUser, setCurrentUser } = useStore()
    const [document, setDocument] = useState<Document | null>(null)
    const [comments, setComments] = useState<Comment[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [showComments, setShowComments] = useState(false)
    const [showShare, setShowShare] = useState(false)

    const [permission, setPermission] = useState<string>('view')
    const [requestStatus, setRequestStatus] = useState<'idle' | 'loading' | 'sent' | 'error'>('idle')

    const {
        editor,
        provider,
        isConnected,
        collaborators,
    } = useCollaboration(docId, currentUser, permission)

    const [originalTitle, setOriginalTitle] = useState<string>('')

    useEffect(() => {
        loadDocument()
    }, [docId])

    const loadDocument = async () => {
        try {
            const [user, doc, docComments, perm] = await Promise.all([
                api.getCurrentUser(),
                api.getDocument(docId),
                api.listComments(docId),
                api.getMyPermission(docId)
            ])
            setCurrentUser(user)
            setDocument(doc)
            setOriginalTitle(doc.title) // Track original title from server
            setComments(docComments)
            setPermission(perm)
        } catch (err) {
            console.error('Failed to load document:', err)
            if (err instanceof Error) {
                setError(err.message)
            } else {
                setError('Failed to load document')
            }
        } finally {
            setLoading(false)
        }
    }

    const updateTitle = useCallback(async (newTitle: string) => {
        if (!document || originalTitle === newTitle) return

        try {
            await api.updateDocument(docId, { title: newTitle })
            setOriginalTitle(newTitle) // Update original title after successful save
        } catch (error) {
            console.error('Failed to update title:', error)
            // Revert to original title on error
            setDocument(prev => prev ? { ...prev, title: originalTitle } : null)
        }
    }, [docId, document, originalTitle])

    const addComment = async (content: string, selection?: { anchor: number; head: number }) => {
        try {
            const comment = await api.createComment(docId, { content, selection })
            setComments(prev => [comment, ...prev])
        } catch (error) {
            console.error('Failed to add comment:', error)
        }
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500"></div>
            </div>
        )
    }

    if (error || !document) {
        const isAccessDenied = error?.includes('No access') || error?.includes('Forbidden') || error?.includes('access')

        const handleRequestAccess = async () => {
            setRequestStatus('loading')
            try {
                await api.requestAccess(docId)
                setRequestStatus('sent')
            } catch (err) {
                console.error('Failed to request access:', err)
                setRequestStatus('error')
            }
        }

        return (
            <div className="flex flex-col items-center justify-center min-h-screen bg-gradient-to-b from-slate-50 to-white dark:from-slate-900 dark:to-slate-800">
                <div className="text-center p-8">
                    {isAccessDenied ? (
                        <>
                            <div className="inline-flex items-center justify-center w-20 h-20 bg-orange-100 dark:bg-orange-900/30 rounded-full mb-6">
                                <Lock className="w-10 h-10 text-orange-500" />
                            </div>
                            <h1 className="text-2xl font-bold text-slate-900 dark:text-white mb-3">
                                No Permission
                            </h1>
                            <p className="text-slate-600 dark:text-slate-400 mb-6 max-w-md">
                                You don't have access to this document.
                            </p>

                            {requestStatus === 'sent' ? (
                                <div className="flex items-center justify-center gap-2 text-green-600 dark:text-green-400 mb-6">
                                    <Check className="w-5 h-5" />
                                    <span>Request sent! The document owner will review your request.</span>
                                </div>
                            ) : (
                                <button
                                    onClick={handleRequestAccess}
                                    disabled={requestStatus === 'loading'}
                                    className="flex items-center justify-center gap-2 px-6 py-3 bg-primary-500 hover:bg-primary-600 disabled:bg-primary-400 text-white rounded-lg font-medium transition-colors mb-4"
                                >
                                    {requestStatus === 'loading' ? (
                                        <Loader2 className="w-5 h-5 animate-spin" />
                                    ) : (
                                        <Send className="w-5 h-5" />
                                    )}
                                    Request Access
                                </button>
                            )}

                            {requestStatus === 'error' && (
                                <p className="text-red-500 mb-4">Failed to send request. You may have already requested access.</p>
                            )}
                        </>
                    ) : (
                        <>
                            <div className="inline-flex items-center justify-center w-20 h-20 bg-slate-100 dark:bg-slate-800 rounded-full mb-6">
                                <FileQuestion className="w-10 h-10 text-slate-400" />
                            </div>
                            <h1 className="text-2xl font-bold text-slate-900 dark:text-white mb-3">
                                Document Not Found
                            </h1>
                            <p className="text-slate-600 dark:text-slate-400 mb-6">
                                This document doesn't exist or has been deleted.
                            </p>
                        </>
                    )}
                    <button
                        onClick={() => router.push('/')}
                        className="px-6 py-3 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg font-medium transition-colors"
                    >
                        Go to My Documents
                    </button>
                </div>
            </div>
        )
    }

    const canEdit = permission === 'owner' || permission === 'edit'

    return (
        <div className="min-h-screen flex flex-col">
            {/* Header */}
            <header className="sticky top-0 z-50 bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm border-b border-slate-200 dark:border-slate-700">
                <div className="max-w-7xl mx-auto px-4">
                    <div className="flex items-center justify-between h-14">
                        {/* Left side */}
                        <div className="flex items-center gap-4">
                            <button
                                onClick={() => router.push(getBackUrl())}
                                className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
                            >
                                <ArrowLeft className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                            </button>

                            <input
                                type="text"
                                value={document.title}
                                onChange={(e) => setDocument(prev => prev ? { ...prev, title: e.target.value } : null)}
                                onBlur={(e) => updateTitle(e.target.value)}
                                disabled={!canEdit}
                                className="text-lg font-semibold bg-transparent border-none outline-none focus:ring-2 focus:ring-primary-500 rounded px-2 py-1 text-slate-900 dark:text-white disabled:opacity-75"
                            />
                        </div>

                        {/* Right side */}
                        <div className="flex items-center gap-2">
                            {/* Connection status */}
                            <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full text-sm ${isConnected
                                ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                                : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
                                }`}>
                                <span className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-yellow-500'}`} />
                                {isConnected ? 'Connected' : 'Connecting...'}
                            </div>

                            {/* Collaborators */}
                            <CollaboratorsList collaborators={collaborators} />

                            {/* Comments button */}
                            <button
                                onClick={() => setShowComments(!showComments)}
                                className={`p-2 rounded-lg transition-colors ${showComments
                                    ? 'bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400'
                                    : 'hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-400'
                                    }`}
                            >
                                <MessageSquare className="w-5 h-5" />
                            </button>

                            {/* Share button */}
                            {permission === 'owner' && (
                                <button
                                    onClick={() => setShowShare(true)}
                                    className="flex items-center gap-2 px-3 py-2 bg-primary-500 hover:bg-primary-600 text-white rounded-lg text-sm font-medium transition-colors"
                                >
                                    <Share2 className="w-4 h-4" />
                                    Share
                                </button>
                            )}
                        </div>
                    </div>

                    {/* Toolbar */}
                    {editor && canEdit && <Toolbar editor={editor} />}
                </div>
            </header>

            {/* Main content */}
            <main className="flex-1 flex">
                {/* Editor */}
                <div className={`flex-1 transition-all ${showComments ? 'mr-80' : ''}`}>
                    <div className="max-w-4xl mx-auto py-8 px-4">
                        <div className="bg-white dark:bg-slate-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700 min-h-[600px]">
                            {editor ? (
                                <Editor editor={editor} />
                            ) : (
                                <div className="flex items-center justify-center h-96">
                                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
                                </div>
                            )}
                        </div>
                    </div>
                </div>

                {/* Comments panel */}
                {showComments && (
                    <CommentsPanel
                        comments={comments}
                        onAddComment={addComment}
                        onClose={() => setShowComments(false)}
                        canComment={permission !== 'view'}
                    />
                )}
            </main>

            {/* Share modal */}
            {showShare && (
                <ShareModal
                    docId={docId}
                    onClose={() => setShowShare(false)}
                />
            )}
        </div>
    )
}
