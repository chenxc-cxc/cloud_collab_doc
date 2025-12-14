/**
 * Comment Position Resolver
 * 
 * Architecture:
 * 1. Comments store RelativePosition (from Yjs) as their anchor
 * 2. This resolver converts RelativePosition to current document positions
 * 3. Both Decoration and CommentsPanel use this resolver
 * 
 * RelativePosition is stable across document edits because it references
 * the Yjs item ID rather than an absolute position.
 */

import * as Y from 'yjs'
import type { Editor } from '@tiptap/react'
import type { Comment } from '@/types'

// Resolved position for a comment
export interface ResolvedCommentPosition {
    commentId: string
    from: number
    to: number
    isValid: boolean
}

// Cache for relative positions (created from absolute positions)
const relativePositionCache = new Map<string, {
    fromRelPos: Y.RelativePosition;
    toRelPos: Y.RelativePosition
}>()

/**
 * Create relative positions from absolute positions
 * This should be called when a comment is first created
 */
export function createRelativePositions(
    ydoc: Y.Doc,
    ytext: Y.XmlFragment,
    from: number,
    to: number
): { fromRelPos: Y.RelativePosition; toRelPos: Y.RelativePosition } | null {
    try {
        const fromRelPos = Y.createRelativePositionFromTypeIndex(ytext, from)
        const toRelPos = Y.createRelativePositionFromTypeIndex(ytext, to)
        return { fromRelPos, toRelPos }
    } catch (e) {
        console.warn('Failed to create relative positions:', e)
        return null
    }
}

/**
 * Store relative positions for a comment
 */
export function cacheRelativePositions(
    commentId: string,
    ydoc: Y.Doc,
    ytext: Y.XmlFragment,
    from: number,
    to: number
): void {
    const relPos = createRelativePositions(ydoc, ytext, from, to)
    if (relPos) {
        relativePositionCache.set(commentId, relPos)
    }
}

/**
 * Resolve a comment's position from its cached relative positions
 */
export function resolveCommentPosition(
    commentId: string,
    ydoc: Y.Doc,
    ytext: Y.XmlFragment
): { from: number; to: number } | null {
    const cached = relativePositionCache.get(commentId)
    if (!cached) return null

    try {
        const fromAbs = Y.createAbsolutePositionFromRelativePosition(cached.fromRelPos, ydoc)
        const toAbs = Y.createAbsolutePositionFromRelativePosition(cached.toRelPos, ydoc)

        if (fromAbs && toAbs) {
            return { from: fromAbs.index, to: toAbs.index }
        }
    } catch (e) {
        console.warn('Failed to resolve comment position:', commentId, e)
    }
    return null
}

/**
 * Initialize relative positions for all comments
 * This should be called when comments are first loaded
 */
export function initializeCommentPositions(
    comments: Comment[],
    ydoc: Y.Doc,
    ytext: Y.XmlFragment
): void {
    comments.forEach(comment => {
        if (comment.selection && !relativePositionCache.has(comment.id)) {
            const { anchor, head } = comment.selection
            const from = Math.min(anchor, head)
            const to = Math.max(anchor, head)
            cacheRelativePositions(comment.id, ydoc, ytext, from, to)
        }
    })
}

/**
 * Get the Y.XmlFragment from editor (the document content)
 */
export function getYTextFromEditor(editor: Editor, ydoc: Y.Doc): Y.XmlFragment | null {
    try {
        // TipTap Collaboration extension uses 'default' as the fragment name
        return ydoc.getXmlFragment('default')
    } catch (e) {
        console.warn('Failed to get YText from editor:', e)
        return null
    }
}

/**
 * Resolve all comments' positions
 */
export function resolveAllCommentPositions(
    comments: Comment[],
    ydoc: Y.Doc,
    ytext: Y.XmlFragment
): Map<string, { from: number; to: number }> {
    const positions = new Map<string, { from: number; to: number }>()

    comments.forEach(comment => {
        if (comment.selection) {
            const resolved = resolveCommentPosition(comment.id, ydoc, ytext)
            if (resolved) {
                positions.set(comment.id, resolved)
            } else {
                // Fallback to original selection
                const { anchor, head } = comment.selection
                const from = Math.min(anchor, head)
                const to = Math.max(anchor, head)
                positions.set(comment.id, { from, to })
            }
        }
    })

    return positions
}

/**
 * Clear cached positions (for cleanup or reset)
 */
export function clearPositionCache(): void {
    relativePositionCache.clear()
}

/**
 * Clear a single comment's cached position
 */
export function removePositionFromCache(commentId: string): void {
    relativePositionCache.delete(commentId)
}

// Export the cache for debugging
export function getPositionCacheSize(): number {
    return relativePositionCache.size
}
