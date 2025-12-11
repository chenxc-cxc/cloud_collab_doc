'use client'

import { useState, useEffect } from 'react'
import { X, Send, Check, CornerDownRight } from 'lucide-react'
import type { Comment } from '@/types'

interface CommentsPanelProps {
    comments: Comment[]
    onAddComment: (content: string, selection?: { anchor: number; head: number }) => void
    onClose: () => void
    canComment: boolean
}

export default function CommentsPanel({
    comments,
    onAddComment,
    onClose,
    canComment
}: CommentsPanelProps) {
    const [newComment, setNewComment] = useState('')
    const [headerHeight, setHeaderHeight] = useState(56)

    useEffect(() => {
        // Get header height dynamically
        const updateHeaderHeight = () => {
            const header = document.getElementById('doc-header')
            if (header) {
                setHeaderHeight(header.offsetHeight)
            }
        }

        updateHeaderHeight()
        window.addEventListener('resize', updateHeaderHeight)

        // Check again after a short delay for toolbar to render
        const timer = setTimeout(updateHeaderHeight, 100)

        return () => {
            window.removeEventListener('resize', updateHeaderHeight)
            clearTimeout(timer)
        }
    }, [])

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        if (!newComment.trim()) return

        onAddComment(newComment.trim())
        setNewComment('')
    }

    const formatDate = (date: string) => {
        const d = new Date(date)
        const now = new Date()
        const diff = now.getTime() - d.getTime()
        const minutes = Math.floor(diff / 60000)
        const hours = Math.floor(minutes / 60)
        const days = Math.floor(hours / 24)

        if (minutes < 1) return 'Just now'
        if (minutes < 60) return `${minutes}m ago`
        if (hours < 24) return `${hours}h ago`
        if (days < 7) return `${days}d ago`

        return d.toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
        })
    }

    return (
        <div
            className="fixed right-0 bottom-0 w-80 bg-white dark:bg-slate-800 border-l border-slate-200 dark:border-slate-700 flex flex-col animate-slide-up"
            style={{ top: `${headerHeight}px` }}
        >
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200 dark:border-slate-700">
                <h3 className="font-semibold text-slate-900 dark:text-white">
                    Comments ({comments.length})
                </h3>
                <button
                    onClick={onClose}
                    className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
                >
                    <X className="w-5 h-5 text-slate-500" />
                </button>
            </div>

            {/* Comment form */}
            {canComment && (
                <form onSubmit={handleSubmit} className="p-4 border-b border-slate-200 dark:border-slate-700">
                    <textarea
                        value={newComment}
                        onChange={(e) => setNewComment(e.target.value)}
                        placeholder="Add a comment..."
                        rows={3}
                        className="w-full px-3 py-2 text-sm bg-slate-50 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
                    />
                    <div className="flex justify-end mt-2">
                        <button
                            type="submit"
                            disabled={!newComment.trim()}
                            className="flex items-center gap-1.5 px-3 py-1.5 bg-primary-500 hover:bg-primary-600 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            <Send className="w-3.5 h-3.5" />
                            Send
                        </button>
                    </div>
                </form>
            )}

            {/* Comments list */}
            <div className="flex-1 overflow-y-auto">
                {comments.length === 0 ? (
                    <div className="text-center py-12 text-slate-500 dark:text-slate-400">
                        <p className="text-sm">No comments yet</p>
                        {canComment && (
                            <p className="text-xs mt-1">Be the first to comment!</p>
                        )}
                    </div>
                ) : (
                    <div className="divide-y divide-slate-100 dark:divide-slate-700">
                        {comments.map((comment) => (
                            <div
                                key={comment.id}
                                className={`p-4 hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors ${comment.resolved ? 'opacity-60' : ''
                                    }`}
                            >
                                <div className="flex items-start gap-3">
                                    <div
                                        className="w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium flex-shrink-0"
                                        style={{ backgroundColor: '#' + comment.user_id.substring(0, 6) }}
                                    >
                                        {(comment.user?.name || 'U').charAt(0).toUpperCase()}
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2 mb-1">
                                            <span className="font-medium text-sm text-slate-900 dark:text-white truncate">
                                                {comment.user?.name || 'Unknown'}
                                            </span>
                                            <span className="text-xs text-slate-500 dark:text-slate-400">
                                                {formatDate(comment.created_at)}
                                            </span>
                                        </div>
                                        <p className="text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap">
                                            {comment.content}
                                        </p>
                                        {comment.resolved && (
                                            <div className="flex items-center gap-1 mt-2 text-xs text-green-600 dark:text-green-400">
                                                <Check className="w-3.5 h-3.5" />
                                                Resolved
                                            </div>
                                        )}
                                        {comment.replies && comment.replies.length > 0 && (
                                            <div className="mt-3 space-y-2">
                                                {comment.replies.map((reply) => (
                                                    <div key={reply.id} className="flex items-start gap-2 pl-2 border-l-2 border-slate-200 dark:border-slate-600">
                                                        <CornerDownRight className="w-3 h-3 text-slate-400 mt-1 flex-shrink-0" />
                                                        <div>
                                                            <span className="text-xs font-medium text-slate-600 dark:text-slate-400">
                                                                {reply.user?.name || 'Unknown'}
                                                            </span>
                                                            <p className="text-xs text-slate-600 dark:text-slate-400">
                                                                {reply.content}
                                                            </p>
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    )
}
