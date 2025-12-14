/**
 * Comment Highlight Plugin for TipTap/ProseMirror
 * 
 * Architecture:
 * - ProseMirror Plugin: stores decoration state, handles click events
 * - DecorationSet: renders highlights, underlines, comment markers
 * - React: only handles UI (comment panel, buttons)
 * 
 * Usage:
 * 1. Register the extension in useEditor
 * 2. Call editor.commands.setCommentDecorations(comments, showSidebar)
 * 3. Set editor.storage.commentHighlight.onCommentClick callback
 */

import { Extension } from '@tiptap/core'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import type { EditorView } from '@tiptap/pm/view'
import type { Comment } from '@/types'

// Plugin state interface
interface CommentPluginState {
    comments: Comment[]
    showSidebar: boolean
    decorations: DecorationSet
    positions: Map<string, { from: number; to: number }>
}

// Plugin key for accessing state
export const commentHighlightPluginKey = new PluginKey<CommentPluginState>('commentHighlight')

// Storage interface for React communication
export interface CommentHighlightStorage {
    onCommentClick: ((commentId: string) => void) | null
    getMappedCommentPositions: (() => Map<string, { from: number; to: number }>) | null
}

// Meta actions for updating state
interface SetDecorationsAction {
    type: 'SET_DECORATIONS'
    comments: Comment[]
    showSidebar: boolean
}

type CommentPluginAction = SetDecorationsAction

// Declare TipTap commands
declare module '@tiptap/core' {
    interface Commands<ReturnType> {
        commentHighlight: {
            setCommentDecorations: (comments: Comment[], showSidebar: boolean) => ReturnType
        }
    }
}
// Helper: Create decorations from comments and return positions map
function createDecorations(
    doc: any,
    comments: Comment[],
    showSidebar: boolean,
    onCommentClick: ((commentId: string) => void) | null
): { decorations: DecorationSet; positions: Map<string, { from: number; to: number }> } {
    const decorations: Decoration[] = []
    const positions = new Map<string, { from: number; to: number }>()

    comments.forEach((comment) => {
        if (!comment.selection) return

        const { anchor, head } = comment.selection
        const from = Math.min(anchor, head)
        const to = Math.max(anchor, head)

        // Validate positions against document size
        if (from < 0 || to > doc.content.size || from >= to) {
            return
        }

        // Store the position for this comment
        positions.set(comment.id, { from, to })

        try {
            if (showSidebar) {
                // Sidebar open: show yellow underline highlight
                decorations.push(
                    Decoration.inline(from, to, {
                        class: 'comment-highlight',
                        'data-comment-id': comment.id,
                    })
                )

                // Add anchor widget at the START of the highlighted text
                // This is the key element for Mirror Anchors alignment
                decorations.push(
                    Decoration.widget(from, () => {
                        const anchor = document.createElement('span')
                        anchor.className = 'comment-anchor'
                        anchor.setAttribute('data-comment-id', comment.id)
                        // Anchor is invisible but participates in document flow
                        return anchor
                    }, { side: -1 }) // side: -1 means before the position
                )
            } else {
                // Sidebar closed: show comment icon widget at end of selection
                decorations.push(
                    Decoration.widget(to, (view: EditorView) => {
                        const icon = document.createElement('span')
                        icon.className = 'comment-marker'
                        icon.innerHTML = '💬'
                        icon.setAttribute('data-comment-id', comment.id)
                        icon.title = `${comment.user?.name || 'Unknown'}: ${comment.content.substring(0, 50)}${comment.content.length > 50 ? '...' : ''}`

                        icon.addEventListener('click', (e) => {
                            e.preventDefault()
                            e.stopPropagation()
                            if (onCommentClick) {
                                onCommentClick(comment.id)
                            }
                        })

                        return icon
                    }, { side: 1 })
                )
            }
        } catch (e) {
            console.warn('Failed to create decoration for comment:', comment.id, e)
        }
    })

    return { decorations: DecorationSet.create(doc, decorations), positions }
}

// TipTap Extension
export const CommentHighlight = Extension.create<Record<string, never>, CommentHighlightStorage>({
    name: 'commentHighlight',

    // Storage for React communication
    addStorage() {
        return {
            onCommentClick: null,
            getMappedCommentPositions: null,
        }
    },

    // Commands for updating decorations from React
    addCommands() {
        return {
            setCommentDecorations: (comments: Comment[], showSidebar: boolean) => ({ tr, dispatch }) => {
                if (dispatch) {
                    const action: CommentPluginAction = {
                        type: 'SET_DECORATIONS',
                        comments,
                        showSidebar,
                    }
                    tr.setMeta(commentHighlightPluginKey, action)
                    dispatch(tr)
                }
                return true
            },
        }
    },

    // ProseMirror Plugin
    addProseMirrorPlugins() {
        const extension = this

        return [
            new Plugin<CommentPluginState>({
                key: commentHighlightPluginKey,

                // Initial state
                state: {
                    init() {
                        return {
                            comments: [],
                            showSidebar: false,
                            decorations: DecorationSet.empty,
                            positions: new Map(),
                        }
                    },

                    // State reducer
                    apply(tr, state, oldEditorState, newEditorState) {
                        const action = tr.getMeta(commentHighlightPluginKey) as CommentPluginAction | undefined

                        if (action?.type === 'SET_DECORATIONS') {
                            // Rebuild decorations with new data
                            const result = createDecorations(
                                newEditorState.doc,
                                action.comments,
                                action.showSidebar,
                                extension.storage.onCommentClick
                            )
                            return {
                                comments: action.comments,
                                showSidebar: action.showSidebar,
                                decorations: result.decorations,
                                positions: result.positions,
                            }
                        }

                        // Map decorations and positions through document changes
                        if (tr.docChanged) {
                            // Map positions through the mapping
                            const newPositions = new Map<string, { from: number; to: number }>()
                            state.positions.forEach((pos, commentId) => {
                                const newFrom = tr.mapping.map(pos.from)
                                const newTo = tr.mapping.map(pos.to)
                                if (newFrom < newTo) {
                                    newPositions.set(commentId, { from: newFrom, to: newTo })
                                }
                            })
                            return {
                                ...state,
                                decorations: state.decorations.map(tr.mapping, tr.doc),
                                positions: newPositions,
                            }
                        }

                        return state
                    },
                },

                // Provide decorations to the view
                props: {
                    decorations(state) {
                        return this.getState(state)?.decorations ?? DecorationSet.empty
                    },

                    // Handle clicks on highlighted text
                    handleClick(view, pos, event) {
                        const target = event.target as HTMLElement

                        // Check if clicked on a highlight
                        const highlight = target.closest('.comment-highlight') as HTMLElement
                        if (highlight) {
                            const commentId = highlight.getAttribute('data-comment-id')
                            if (commentId && extension.storage.onCommentClick) {
                                extension.storage.onCommentClick(commentId)
                                return true
                            }
                        }

                        return false
                    },
                },
            }),
        ]
    },
})
