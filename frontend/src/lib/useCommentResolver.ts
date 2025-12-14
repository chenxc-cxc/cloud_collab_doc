/**
 * useCommentResolver Hook
 * 
 * Provides a unified interface for resolving comment positions.
 * Uses Yjs RelativePosition for stable anchors that track document changes.
 */

'use client'

import { useCallback, useEffect, useRef, useMemo } from 'react'
import type { Editor } from '@tiptap/react'
import * as Y from 'yjs'
import type { Comment } from '@/types'
import {
    initializeCommentPositions,
    cacheRelativePositions,
    resolveAllCommentPositions,
    getYTextFromEditor,
    clearPositionCache
} from './commentPositionResolver'

export interface CommentPositionResolver {
    // Resolve a single comment's position
    resolvePosition: (commentId: string) => { from: number; to: number } | null
    // Resolve all comments' positions
    resolveAllPositions: () => Map<string, { from: number; to: number }>
    // Cache a new comment's position
    cacheNewComment: (commentId: string, from: number, to: number) => void
    // Get Y position for rendering
    getYPosition: (commentId: string) => number | null
    // Get text between positions
    getTextBetween: (commentId: string) => string
    // Get comments with invalid (deleted) positions
    getInvalidCommentIds: () => string[]
}

export function useCommentResolver(
    editor: Editor | null,
    ydoc: Y.Doc | null,
    comments: Comment[]
): CommentPositionResolver {
    const ytextRef = useRef<Y.XmlFragment | null>(null)
    const positionsRef = useRef<Map<string, { from: number; to: number }>>(new Map())
    const invalidCommentsRef = useRef<Set<string>>(new Set())

    // Create a lookup map for comment selections (as fallback)
    const commentSelectionsMap = useMemo(() => {
        const map = new Map<string, { from: number; to: number }>()
        comments.forEach(c => {
            if (c.selection) {
                const from = Math.min(c.selection.anchor, c.selection.head)
                const to = Math.max(c.selection.anchor, c.selection.head)
                map.set(c.id, { from, to })
            }
        })
        return map
    }, [comments])

    // Get YText fragment when editor and ydoc are ready
    useEffect(() => {
        if (editor && ydoc) {
            ytextRef.current = getYTextFromEditor(editor, ydoc)
        }
    }, [editor, ydoc])

    // Initialize positions when comments load
    useEffect(() => {
        if (ydoc && ytextRef.current && comments.length > 0) {
            initializeCommentPositions(comments, ydoc, ytextRef.current)
            // Resolve all positions
            positionsRef.current = resolveAllCommentPositions(comments, ydoc, ytextRef.current)
        }

        // Also populate from comment selections as fallback
        comments.forEach(c => {
            if (c.selection && !positionsRef.current.has(c.id)) {
                const from = Math.min(c.selection.anchor, c.selection.head)
                const to = Math.max(c.selection.anchor, c.selection.head)
                positionsRef.current.set(c.id, { from, to })
            }
        })
    }, [ydoc, comments])

    // Update positions when document changes
    // Note: We do NOT auto-delete comments here as it's too aggressive
    // and can cause issues with copy-paste operations
    useEffect(() => {
        if (!editor || !ydoc || !ytextRef.current) return

        const updatePositions = () => {
            if (ytextRef.current) {
                positionsRef.current = resolveAllCommentPositions(comments, ydoc, ytextRef.current)

                // Check for truly invalid comments (position completely out of bounds)
                // Only mark as invalid if position is definitely wrong, not just empty text
                const docSize = editor.state.doc.content.size
                invalidCommentsRef.current.clear()

                positionsRef.current.forEach((pos, commentId) => {
                    // Only mark as invalid if position is definitively out of bounds
                    // Don't mark as invalid just because text is empty (could be transient)
                    if (pos.from < 0 || pos.to > docSize || pos.from > pos.to) {
                        invalidCommentsRef.current.add(commentId)
                    }
                })
            }
        }

        // Listen to editor updates
        editor.on('update', updatePositions)

        return () => {
            editor.off('update', updatePositions)
        }
    }, [editor, ydoc, comments])

    // Cleanup on unmount
    useEffect(() => {
        return () => {
            clearPositionCache()
        }
    }, [])

    // Resolve a single position
    const resolvePosition = useCallback((commentId: string): { from: number; to: number } | null => {
        // Try from positionsRef first
        const pos = positionsRef.current.get(commentId)
        if (pos) return pos

        // Fallback to comment selections map
        return commentSelectionsMap.get(commentId) || null
    }, [commentSelectionsMap])

    // Resolve all positions
    const resolveAllPositions = useCallback((): Map<string, { from: number; to: number }> => {
        if (ydoc && ytextRef.current) {
            positionsRef.current = resolveAllCommentPositions(comments, ydoc, ytextRef.current)
        }

        // Merge with fallback positions
        commentSelectionsMap.forEach((pos, id) => {
            if (!positionsRef.current.has(id)) {
                positionsRef.current.set(id, pos)
            }
        })

        return positionsRef.current
    }, [ydoc, comments, commentSelectionsMap])

    // Cache a new comment's position
    const cacheNewComment = useCallback((commentId: string, from: number, to: number) => {
        if (ydoc && ytextRef.current) {
            cacheRelativePositions(commentId, ydoc, ytextRef.current, from, to)
        }
        positionsRef.current.set(commentId, { from, to })
    }, [ydoc])

    // Get Y position for UI rendering
    const getYPosition = useCallback((commentId: string): number | null => {
        if (!editor) return null

        // Try positionsRef first, then fallback
        let pos = positionsRef.current.get(commentId)
        if (!pos) {
            pos = commentSelectionsMap.get(commentId)
        }
        if (!pos) return null

        try {
            const coords = editor.view.coordsAtPos(pos.from)
            return coords?.top ?? null
        } catch {
            return null
        }
    }, [editor, commentSelectionsMap])

    // Get text between positions
    const getTextBetween = useCallback((commentId: string): string => {
        if (!editor) return ''

        // Try positionsRef first, then fallback
        let pos = positionsRef.current.get(commentId)
        if (!pos) {
            pos = commentSelectionsMap.get(commentId)
        }
        if (!pos) return ''

        try {
            const docSize = editor.state.doc.content.size
            if (pos.from >= 0 && pos.to <= docSize && pos.from < pos.to) {
                const text = editor.state.doc.textBetween(pos.from, pos.to, ' ')
                return text
            }
        } catch {
            // Position out of bounds
        }
        return ''
    }, [editor, commentSelectionsMap])

    // Get comments with invalid (deleted) positions
    const getInvalidCommentIds = useCallback((): string[] => {
        return Array.from(invalidCommentsRef.current)
    }, [])

    return {
        resolvePosition,
        resolveAllPositions,
        cacheNewComment,
        getYPosition,
        getTextBetween,
        getInvalidCommentIds
    }
}
