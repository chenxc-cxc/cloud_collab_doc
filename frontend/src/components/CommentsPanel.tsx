'use client'

import { useState, useEffect, useRef, useCallback } from 'react'
import { X, Send, Quote } from 'lucide-react'
import type { Comment } from '@/types'
import type { Editor } from '@tiptap/react'
import type { CommentPositionResolver } from '@/lib/useCommentResolver'

interface CommentsPanelProps {
    comments: Comment[]
    onAddComment: (content: string, selection?: { anchor: number; head: number }) => void
    onClose: () => void
    canComment: boolean
    editor?: Editor | null
    activeCommentId?: string | null
    commentResolver?: CommentPositionResolver
}

interface PositionedItem {
    id: string
    type: 'comment' | 'input'
    targetTop: number  // Desired position (from anchor)
    actualTop: number  // Adjusted position after avoiding overlaps
    height: number
}

export default function CommentsPanel({
    comments,
    onAddComment,
    onClose,
    canComment,
    editor,
    activeCommentId,
    commentResolver
}: CommentsPanelProps) {
    const [newComment, setNewComment] = useState('')
    const [selectedText, setSelectedText] = useState<string>('')
    const [selectionRange, setSelectionRange] = useState<{ anchor: number; head: number } | null>(null)
    const [positions, setPositions] = useState<Map<string, number>>(new Map())
    const sidebarRef = useRef<HTMLDivElement>(null)
    const isSelectingRef = useRef<boolean>(false)

    // Only get comments with selections
    const alignedComments = comments.filter(c => c.selection)

    // Sort by document position
    const sortedComments = [...alignedComments].sort((a, b) => {
        const aPos = a.selection ? Math.min(a.selection.anchor, a.selection.head) : 0
        const bPos = b.selection ? Math.min(b.selection.anchor, b.selection.head) : 0
        return aPos - bPos
    })

    // Calculate positions for all items (comments + input if active)
    const calculatePositions = useCallback(() => {
        if (!sidebarRef.current || !editor) return

        const sidebarRect = sidebarRef.current.getBoundingClientRect()
        const items: PositionedItem[] = []

        // Get all comment anchor positions
        const anchors = document.querySelectorAll('.comment-anchor[data-comment-id]')
        anchors.forEach(anchor => {
            const commentId = anchor.getAttribute('data-comment-id')
            if (commentId) {
                const rect = anchor.getBoundingClientRect()
                const targetTop = rect.top - sidebarRect.top
                items.push({
                    id: commentId,
                    type: 'comment',
                    targetTop,
                    actualTop: targetTop,
                    height: 100 // Estimated, will be refined
                })
            }
        })

        // Add input form if selection is active
        if (selectionRange) {
            try {
                const from = Math.min(selectionRange.anchor, selectionRange.head)
                const coords = editor.view.coordsAtPos(from)
                if (coords) {
                    const targetTop = coords.top - sidebarRect.top
                    items.push({
                        id: '__input__',
                        type: 'input',
                        targetTop,
                        actualTop: targetTop,
                        height: 140 // Approximate input form height
                    })
                }
            } catch {
                // ignore
            }
        }

        // Sort all items by target position
        items.sort((a, b) => a.targetTop - b.targetTop)

        // Adjust positions to avoid overlap
        const MIN_GAP = 8
        let lastBottom = 40 // Start below header

        // Get actual heights from DOM
        items.forEach(item => {
            if (item.type === 'comment') {
                const card = sidebarRef.current?.querySelector(`[data-comment-item="${item.id}"]`)
                if (card) {
                    item.height = card.getBoundingClientRect().height
                }
            }
        })

        // Calculate adjusted positions
        items.forEach(item => {
            item.actualTop = Math.max(item.targetTop, lastBottom)
            lastBottom = item.actualTop + item.height + MIN_GAP
        })

        // Update state
        const newPositions = new Map<string, number>()
        items.forEach(item => {
            newPositions.set(item.id, item.actualTop)
        })
        setPositions(newPositions)
    }, [editor, selectionRange])

    // Sync positions on scroll and changes
    useEffect(() => {
        const timer = setTimeout(calculatePositions, 100)

        const scrollContainer = sidebarRef.current?.parentElement
        const handleEvent = () => {
            calculatePositions()
        }

        scrollContainer?.addEventListener('scroll', handleEvent)
        window.addEventListener('resize', handleEvent)

        return () => {
            clearTimeout(timer)
            scrollContainer?.removeEventListener('scroll', handleEvent)
            window.removeEventListener('resize', handleEvent)
        }
    }, [calculatePositions])

    // Re-calculate when comments or selection changes
    useEffect(() => {
        const timer = setTimeout(calculatePositions, 50)
        return () => clearTimeout(timer)
    }, [comments, selectionRange, calculatePositions])

    // Track editor selection - only update after mouseup from editor
    useEffect(() => {
        if (!editor) return

        const handleMouseDown = () => {
            isSelectingRef.current = true
        }

        const handleMouseUp = (e: MouseEvent) => {
            // Only process if we started selecting from the editor
            if (!isSelectingRef.current) return
            isSelectingRef.current = false

            // Don't update selection if clicking inside the sidebar (e.g., submit button)
            if (sidebarRef.current?.contains(e.target as Node)) {
                return
            }

            setTimeout(() => {
                const { from, to, empty } = editor.state.selection
                if (!empty && from !== to) {
                    const text = editor.state.doc.textBetween(from, to, ' ')
                    setSelectedText(text.length > 100 ? text.substring(0, 100) + '...' : text)
                    setSelectionRange({ anchor: from, head: to })
                } else {
                    setSelectedText('')
                    setSelectionRange(null)
                }
            }, 10)
        }

        const editorDom = editor.view.dom
        editorDom.addEventListener('mousedown', handleMouseDown)
        document.addEventListener('mouseup', handleMouseUp)

        return () => {
            editorDom.removeEventListener('mousedown', handleMouseDown)
            document.removeEventListener('mouseup', handleMouseUp)
        }
    }, [editor])

    // Scroll to active comment
    useEffect(() => {
        if (!activeCommentId || !sidebarRef.current) return

        const commentElement = sidebarRef.current.querySelector(
            `[data-comment-item="${activeCommentId}"]`
        )
        if (commentElement) {
            commentElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
        }
    }, [activeCommentId])

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        if (!newComment.trim() || !selectionRange) return

        onAddComment(newComment.trim(), selectionRange)
        setNewComment('')
        setSelectedText('')
        setSelectionRange(null)
    }

    const getQuotedText = useCallback((comment: Comment): string => {
        if (!editor || !comment.selection) return ''

        // Try commentResolver first
        if (commentResolver) {
            const text = commentResolver.getTextBetween(comment.id)
            if (text && text.trim()) {
                return text.length > 50 ? text.substring(0, 50) + '...' : text
            }
        }

        // Fallback: use comment.selection directly
        try {
            const { anchor, head } = comment.selection
            const from = Math.min(anchor, head)
            const to = Math.max(anchor, head)
            const docSize = editor.state.doc.content.size

            // Validate positions
            if (from >= 0 && to <= docSize && from < to) {
                const text = editor.state.doc.textBetween(from, to, ' ')
                return text.length > 50 ? text.substring(0, 50) + '...' : text
            }
        } catch {
            // Position out of bounds
        }
        return ''
    }, [editor, commentResolver])

    const formatDate = (date: string) => {
        const d = new Date(date)
        const now = new Date()
        const diff = now.getTime() - d.getTime()
        const minutes = Math.floor(diff / 60000)

        if (minutes < 1) return 'Just now'
        if (minutes < 60) return `${minutes} mins ago`

        const hours = Math.floor(minutes / 60)
        if (hours < 24) return `${hours} hours ago`

        const days = Math.floor(hours / 24)
        if (days < 7) return `${days} days ago`

        return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
    }

    const renderCommentCard = (comment: Comment) => {
        const top = positions.get(comment.id)

        return (
            <div
                key={comment.id}
                data-comment-item={comment.id}
                className={`
                    absolute left-2 right-2
                    bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700
                    p-3 transition-all duration-200 hover:shadow-md
                    ${comment.resolved ? 'opacity-60' : ''}
                    ${activeCommentId === comment.id
                        ? 'ring-2 ring-amber-400 bg-amber-50 dark:bg-amber-900/20 comment-active'
                        : ''
                    }
                `}
                style={{ top: top !== undefined ? `${top}px` : '0px' }}
            >
                <div className="flex items-start gap-2">
                    <div
                        className="w-6 h-6 rounded-full flex items-center justify-center text-white text-xs font-medium flex-shrink-0"
                        style={{ backgroundColor: '#' + comment.user_id.substring(0, 6) }}
                    >
                        {(comment.user?.name || 'U').charAt(0).toUpperCase()}
                    </div>
                    <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-1.5 mb-0.5">
                            <span className="font-medium text-xs text-slate-900 dark:text-white truncate">
                                {comment.user?.name || 'Unknown'}
                            </span>
                            <span className="text-xs text-slate-400">
                                {formatDate(comment.created_at)}
                            </span>
                        </div>
                        {comment.selection && editor && (
                            <div className="mb-1.5 p-1 bg-amber-50 dark:bg-amber-900/20 border-l-2 border-amber-400 rounded-r">
                                <p className="text-xs text-amber-700 dark:text-amber-400 italic line-clamp-1">
                                    {getQuotedText(comment)}
                                </p>
                            </div>
                        )}
                        <p className="text-sm text-slate-700 dark:text-slate-300">
                            {comment.content}
                        </p>
                    </div>
                </div>
            </div>
        )
    }

    const inputTop = positions.get('__input__') ?? 40

    return (
        <div
            ref={sidebarRef}
            className="w-80 flex-shrink-0 bg-slate-50 dark:bg-slate-900 border-l border-slate-200 dark:border-slate-700"
        >
            {/* Header - fixed at top of viewport */}
            <div
                className="flex items-center justify-between px-4 py-2 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700"
                style={{ position: 'sticky', top: 0, zIndex: 20 }}
            >
                <h3 className="font-semibold text-sm text-slate-900 dark:text-white">
                    Comments ({alignedComments.length})
                </h3>
                <button
                    onClick={onClose}
                    className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors"
                >
                    <X className="w-4 h-4 text-slate-500" />
                </button>
            </div>

            {/* Comments area - extends with content */}
            <div className="relative p-2" style={{ minHeight: '200px' }}>
                {alignedComments.length === 0 && !selectedText ? (
                    <div className="text-center py-12 text-slate-500 dark:text-slate-400">
                        <p className="text-sm">No comment</p>
                        <p className="text-xs mt-1">Select text to add a comment</p>
                    </div>
                ) : (
                    <>
                        {/* All comment cards - absolutely positioned */}
                        {sortedComments.map(comment => renderCommentCard(comment))}

                        {/* Input form - also absolutely positioned */}
                        {canComment && selectedText && (
                            <form
                                onSubmit={handleSubmit}
                                className="absolute left-2 right-2 p-3 bg-white dark:bg-slate-800 rounded-lg shadow-lg border-2 border-amber-400 dark:border-amber-500 z-20 transition-all duration-200"
                                style={{ top: `${inputTop}px` }}
                            >
                                <div className="mb-2 p-1.5 bg-amber-50 dark:bg-amber-900/20 border-l-2 border-amber-400 rounded-r">
                                    <div className="flex items-start gap-1.5">
                                        <Quote className="w-3 h-3 text-amber-500 mt-0.5 flex-shrink-0" />
                                        <p className="text-xs text-amber-700 dark:text-amber-300 italic line-clamp-2">
                                            "{selectedText}"
                                        </p>
                                    </div>
                                </div>
                                <textarea
                                    value={newComment}
                                    onChange={(e) => setNewComment(e.target.value)}
                                    placeholder="Adding comment..."
                                    rows={2}
                                    className="w-full px-2 py-1.5 text-sm bg-slate-50 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded focus:ring-1 focus:ring-primary-500 focus:border-transparent resize-none"
                                />
                                <div className="flex justify-between items-center mt-1.5">
                                    <button
                                        type="button"
                                        onClick={() => {
                                            setSelectedText('')
                                            setSelectionRange(null)
                                            setNewComment('')
                                        }}
                                        className="text-xs text-slate-500 hover:text-slate-700"
                                    >
                                        cancel
                                    </button>
                                    <button
                                        type="submit"
                                        disabled={!newComment.trim()}
                                        className="flex items-center gap-1 px-2.5 py-1 bg-primary-500 hover:bg-primary-600 text-white text-xs font-medium rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                    >
                                        <Send className="w-3 h-3" />
                                        send
                                    </button>
                                </div>
                            </form>
                        )}

                        {/* Spacer to enable scrolling */}
                        <div style={{ height: `${Math.max(...Array.from(positions.values()), 0) + 200}px` }} />
                    </>
                )}
            </div>
        </div>
    )
}
