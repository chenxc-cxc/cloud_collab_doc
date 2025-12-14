'use client'

import { useEffect, useState, useCallback, useMemo } from 'react'
import { useEditor, Editor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCursor from '@tiptap/extension-collaboration-cursor'
import Highlight from '@tiptap/extension-highlight'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import Underline from '@tiptap/extension-underline'
import Placeholder from '@tiptap/extension-placeholder'
import { CommentHighlight } from '@/extensions/CommentHighlight'
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'
import { useStore, getRandomColor } from '@/lib/store'
import type { User, Collaborator } from '@/types'


interface UseCollaborationReturn {
    editor: Editor | null
    provider: WebsocketProvider | null
    ydoc: Y.Doc | null
    isConnected: boolean
    collaborators: Collaborator[]
}

export function useCollaboration(
    docId: string,
    user: User | null,
    permission: string = 'view'
): UseCollaborationReturn {
    const [provider, setProvider] = useState<WebsocketProvider | null>(null)
    const [ydoc, setYdoc] = useState<Y.Doc | null>(null)
    const [isConnected, setIsConnected] = useState(false)
    const [collaborators, setCollaborators] = useState<Collaborator[]>([])

    // Get user color (consistent per user)
    const userColor = useMemo(() => {
        if (!user) return getRandomColor()
        // Generate consistent color from user ID
        const hash = user.id.split('').reduce((acc, char) => {
            return char.charCodeAt(0) + ((acc << 5) - acc)
        }, 0)
        const colors = [
            '#F44336', '#E91E63', '#9C27B0', '#673AB7',
            '#3F51B5', '#2196F3', '#03A9F4', '#00BCD4',
            '#009688', '#4CAF50', '#8BC34A', '#FF9800'
        ]
        return colors[Math.abs(hash) % colors.length]
    }, [user])

    // Initialize Yjs document and WebSocket provider
    useEffect(() => {
        if (!docId) return

        const doc = new Y.Doc()
        setYdoc(doc)

        // Get WebSocket URL from environment
        const wsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081'
        const userId = user?.id || localStorage.getItem('userId') || '11111111-1111-1111-1111-111111111111'

        // Create WebSocket provider
        // Room name is the document ID, y-websocket server handles the routing
        const wsProvider = new WebsocketProvider(
            wsUrl,
            docId,  // Use docId directly as room name
            doc,
            { connect: true, params: { userId /*, token: apiToken */ } }
        )

        setProvider(wsProvider)

        // Connection status
        wsProvider.on('status', (event: { status: string }) => {
            setIsConnected(event.status === 'connected')
        })

        // Handle sync
        wsProvider.on('sync', (isSynced: boolean) => {
            console.log('Sync status:', isSynced)
        })

        // Awareness (presence)
        const awareness = wsProvider.awareness

        // Set local user state
        awareness.setLocalStateField('user', {
            name: user?.name || 'Anonymous',
            color: userColor,
            id: userId,
        })

        // Track awareness changes
        const handleAwarenessChange = () => {
            const states = awareness.getStates()
            const collabs: Collaborator[] = []

            states.forEach((state, clientId) => {
                if (clientId !== awareness.clientID && state.user) {
                    collabs.push({
                        userId: state.user.id || clientId.toString(),
                        name: state.user.name || 'Anonymous',
                        color: state.user.color || '#888888',
                        cursor: state.cursor,
                    })
                }
            })

            setCollaborators(collabs)
        }

        awareness.on('change', handleAwarenessChange)

        // Cleanup
        return () => {
            awareness.off('change', handleAwarenessChange)
            wsProvider.destroy()
            doc.destroy()
        }
    }, [docId, user, userColor])

    // Create TipTap editor
    const editor = useEditor({
        extensions: [
            StarterKit.configure({
                history: false, // Disable history, Yjs handles this
            }),
            Highlight,
            TaskList,
            TaskItem.configure({
                nested: true,
            }),
            Underline,
            Placeholder.configure({
                placeholder: 'Start writing your document...',
            }),
            CommentHighlight,
            ...(ydoc && provider ? [
                Collaboration.configure({
                    document: ydoc,
                }),
                CollaborationCursor.configure({
                    provider: provider,
                    user: {
                        name: user?.name || 'Anonymous',
                        color: userColor,
                    },
                }),
            ] : []),
        ],
        editable: permission !== 'view' && permission !== 'comment',
        editorProps: {
            attributes: {
                class: 'tiptap prose prose-slate dark:prose-invert max-w-none focus:outline-none',
            },
        },
    }, [ydoc, provider, user, userColor, permission])

    // Update editability when permission changes
    useEffect(() => {
        if (editor) {
            editor.setEditable(permission === 'owner' || permission === 'edit')
        }
    }, [editor, permission])

    return {
        editor,
        provider,
        ydoc,
        isConnected,
        collaborators,
    }
}
