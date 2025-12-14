'use client'

import { useEffect, useState, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { ArrowLeft, Users, Share2, MessageSquare, History, Settings, Lock, FileQuestion, Send, Check, Loader2, List } from 'lucide-react'
import { useStore } from '@/lib/store'
import { api } from '@/lib/api'
import { useCollaboration } from '@/hooks/useCollaboration'
import { useCommentResolver } from '@/lib/useCommentResolver'
import Editor from '@/components/Editor'
import Toolbar from '@/components/Toolbar'
import CollaboratorsList from '@/components/CollaboratorsList'
import CommentsPanel from '@/components/CommentsPanel'
import ShareModal from '@/components/ShareModal'
import OutlineSidebar from '@/components/OutlineSidebar'
import SelectionBubbleMenu from '@/components/SelectionBubbleMenu'
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
    const [showOutline, setShowOutline] = useState(true)
    const [activeCommentId, setActiveCommentId] = useState<string | null>(null)

    const [permission, setPermission] = useState<string>('view')
    const [requestStatus, setRequestStatus] = useState<'idle' | 'loading' | 'sent' | 'error'>('idle')
    const [selectedRole, setSelectedRole] = useState<'view' | 'edit'>('view')
    const [upgradeRequestStatus, setUpgradeRequestStatus] = useState<'idle' | 'loading' | 'sent' | 'error'>('idle')

    const {
        editor,
        provider,
        ydoc,
        isConnected,
        collaborators,
    } = useCollaboration(docId, currentUser, permission)

    // Comment position resolver (uses Yjs RelativePosition for stable anchors)
    const commentResolver = useCommentResolver(editor, ydoc, comments)

    const [originalTitle, setOriginalTitle] = useState<string>('')

    // Sync comments and sidebar state with ProseMirror plugin
    useEffect(() => {
        if (!editor) return

        // Set click callback in plugin storage
        editor.storage.commentHighlight.onCommentClick = (commentId: string) => {
            // Open sidebar and set active comment
            setShowComments(true)
            setActiveCommentId(commentId)

            // Clear the active highlight after a delay
            setTimeout(() => {
                setActiveCommentId(null)
            }, 3000)
        }

        // Set the resolver function for getting mapped positions
        // This is stable because commentResolver uses refs internally
        editor.storage.commentHighlight.getMappedCommentPositions = commentResolver.resolveAllPositions
    }, [editor, commentResolver.resolveAllPositions])

    // Update decorations when comments or sidebar state changes
    useEffect(() => {
        if (!editor) return
        // This ensures decorations are applied whenever any dependency changes
        // including when editor first becomes ready
        editor.commands.setCommentDecorations(comments, showComments)
    }, [editor, comments, showComments])

    // Also trigger decoration update when editor becomes ready
    // (in case comments were loaded before editor was initialized)
    useEffect(() => {
        if (!editor) return
        // Small delay to ensure editor is fully ready
        const timer = setTimeout(() => {
            editor.commands.setCommentDecorations(comments, showComments)
        }, 100)
        return () => clearTimeout(timer)
    }, [editor]) // Only depends on editor - run when editor becomes available

    // Note: We previously had auto-deletion of comments when text was deleted,
    // but this was too aggressive and caused issues with copy-paste operations.
    // Users should manually delete comments if needed.

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
            // Add current user info to the comment since backend doesn't return it
            const commentWithUser: Comment = {
                ...comment,
                user: currentUser ? {
                    id: currentUser.id,
                    email: currentUser.email,
                    name: currentUser.name,
                    avatar_url: currentUser.avatar_url,
                    created_at: currentUser.created_at,
                    updated_at: currentUser.updated_at
                } : undefined
            }

            // Cache the new comment's relative position
            if (selection) {
                const from = Math.min(selection.anchor, selection.head)
                const to = Math.max(selection.anchor, selection.head)
                commentResolver.cacheNewComment(comment.id, from, to)
            }

            setComments(prev => [commentWithUser, ...prev])
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
                await api.requestAccess(docId, selectedRole)
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
                                <div className="space-y-4">
                                    {/* Permission Selection */}
                                    <div className="flex items-center justify-center gap-4 mb-4">
                                        <label className="text-sm text-slate-600 dark:text-slate-400">请求权限:</label>
                                        <select
                                            value={selectedRole}
                                            onChange={(e) => setSelectedRole(e.target.value as 'view' | 'edit')}
                                            className="px-4 py-2 bg-white dark:bg-slate-700 border border-slate-300 dark:border-slate-600 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                                        >
                                            <option value="view">仅阅读</option>
                                            <option value="edit">可编辑</option>
                                        </select>
                                    </div>
                                    <button
                                        onClick={handleRequestAccess}
                                        disabled={requestStatus === 'loading'}
                                        className="flex items-center justify-center gap-2 px-6 py-3 bg-primary-500 hover:bg-primary-600 disabled:bg-primary-400 text-white rounded-lg font-medium transition-colors"
                                    >
                                        {requestStatus === 'loading' ? (
                                            <Loader2 className="w-5 h-5 animate-spin" />
                                        ) : (
                                            <Send className="w-5 h-5" />
                                        )}
                                        Request Access
                                    </button>
                                </div>
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

    // Handle upgrade request for users with view-only permission
    const handleUpgradeRequest = async () => {
        setUpgradeRequestStatus('loading')
        try {
            await api.requestAccess(docId, 'edit')
            setUpgradeRequestStatus('sent')
        } catch (err) {
            console.error('Failed to request edit access:', err)
            setUpgradeRequestStatus('error')
        }
    }

    return (
        <div className="min-h-screen flex flex-col">
            {/* View-only Banner */}
            {permission === 'view' && (
                <div className="bg-amber-50 dark:bg-amber-900/30 border-b border-amber-200 dark:border-amber-800 px-4 py-2">
                    <div className="flex items-center justify-between max-w-6xl mx-auto">
                        <div className="flex items-center gap-2 text-amber-700 dark:text-amber-400">
                            <Lock className="w-4 h-4" />
                            <span className="text-sm font-medium">您当前只有阅读权限，无法编辑此文档</span>
                        </div>
                        {upgradeRequestStatus === 'sent' ? (
                            <div className="flex items-center gap-1 text-green-600 dark:text-green-400 text-sm">
                                <Check className="w-4 h-4" />
                                <span>申请已发送，等待审批</span>
                            </div>
                        ) : upgradeRequestStatus === 'error' ? (
                            <span className="text-red-500 text-sm">申请失败，您可能已申请过</span>
                        ) : (
                            <button
                                onClick={handleUpgradeRequest}
                                disabled={upgradeRequestStatus === 'loading'}
                                className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-500 hover:bg-amber-600 disabled:bg-amber-400 text-white text-sm font-medium rounded-lg transition-colors"
                            >
                                {upgradeRequestStatus === 'loading' ? (
                                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                ) : (
                                    <Send className="w-3.5 h-3.5" />
                                )}
                                申请编辑权限
                            </button>
                        )}
                    </div>
                </div>
            )}

            {/* Header */}
            <header className="sticky top-0 z-50 bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm border-b border-slate-200 dark:border-slate-700" id="doc-header">
                <div className="w-full px-4">
                    <div className="flex items-center justify-between h-14 gap-4">
                        {/* Left side */}
                        <div className="flex items-center gap-2 min-w-0 flex-1">
                            {/* Outline toggle button */}
                            <button
                                onClick={() => setShowOutline(!showOutline)}
                                className={`p-2 rounded-lg transition-colors flex-shrink-0 ${showOutline
                                    ? 'bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400'
                                    : 'hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-400'
                                    }`}
                                title={showOutline ? '隐藏目录' : '显示目录'}
                            >
                                <List className="w-5 h-5" />
                            </button>

                            <button
                                onClick={() => router.push(getBackUrl())}
                                className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors flex-shrink-0"
                            >
                                <ArrowLeft className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                            </button>

                            <input
                                type="text"
                                value={document.title}
                                onChange={(e) => setDocument(prev => prev ? { ...prev, title: e.target.value } : null)}
                                onBlur={(e) => updateTitle(e.target.value)}
                                disabled={!canEdit}
                                className="text-lg font-semibold bg-transparent border-none outline-none focus:ring-2 focus:ring-primary-500 rounded px-2 py-1 text-slate-900 dark:text-white disabled:opacity-75 min-w-0 flex-1 truncate"
                            />
                        </div>

                        {/* Right side */}
                        <div className="flex items-center gap-2 flex-shrink-0">
                            {/* Connection status */}
                            <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full text-sm whitespace-nowrap ${isConnected
                                ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                                : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
                                }`}>
                                <span className={`w-2 h-2 rounded-full flex-shrink-0 ${isConnected ? 'bg-green-500' : 'bg-yellow-500'}`} />
                                <span className="hidden sm:inline">{isConnected ? 'Connected' : 'Connecting...'}</span>
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
                                    className="flex items-center gap-2 px-3 py-2 bg-primary-500 hover:bg-primary-600 text-white rounded-lg text-sm font-medium transition-colors whitespace-nowrap"
                                >
                                    <Share2 className="w-4 h-4" />
                                    <span className="hidden sm:inline">Share</span>
                                </button>
                            )}
                        </div>
                    </div>

                    {/* Toolbar */}
                    {editor && canEdit && <Toolbar editor={editor} />}
                </div>
            </header>

            {/* Outline Sidebar */}
            <OutlineSidebar
                editor={editor}
                isOpen={showOutline}
                onToggle={() => setShowOutline(!showOutline)}
            />

            {/* Main content - Shared scroll container for Editor and Comments */}
            <main
                className={`flex-1 flex transition-all duration-300 overflow-y-auto ${showOutline ? 'ml-64' : 'ml-0'}`}
                style={{ height: 'calc(100vh - var(--header-height, 120px))' }}
            >
                {/* Editor area - expands to fill */}
                <div className={`flex-1 transition-all duration-300`}>
                    <div className="px-8 py-6">
                        <div className="bg-white dark:bg-slate-800/50 min-h-[calc(100vh-10rem)]">
                            {editor ? (
                                <>
                                    <SelectionBubbleMenu
                                        editor={editor}
                                        onAddComment={() => setShowComments(true)}
                                        canComment={permission !== 'view'}
                                    />
                                    <Editor editor={editor} />
                                </>
                            ) : (
                                <div className="flex items-center justify-center h-96">
                                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
                                </div>
                            )}
                        </div>
                    </div>
                </div>

                {/* Comments sidebar - shares scroll with editor */}
                {showComments && (
                    <CommentsPanel
                        comments={comments}
                        onAddComment={addComment}
                        onClose={() => setShowComments(false)}
                        canComment={permission !== 'view'}
                        editor={editor}
                        activeCommentId={activeCommentId}
                        commentResolver={commentResolver}
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
