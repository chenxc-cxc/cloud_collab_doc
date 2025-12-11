'use client'

import { useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { FolderTree, Folder, FolderOpen, ChevronRight, ChevronDown, Home, FileText, List, ChevronLeft } from 'lucide-react'
import { api } from '@/lib/api'
import type { FolderTreeNode } from '@/types'

interface FolderTreeSidebarProps {
    isOpen: boolean
    onToggle: () => void
    currentFolderId?: string
    onFolderSelect: (folderId?: string) => void
}

export default function FolderTreeSidebar({
    isOpen,
    onToggle,
    currentFolderId,
    onFolderSelect
}: FolderTreeSidebarProps) {
    const router = useRouter()
    const [tree, setTree] = useState<FolderTreeNode[]>([])
    const [loading, setLoading] = useState(true)
    const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set())

    useEffect(() => {
        loadTree()
    }, [])

    // Auto-expand parent folders when currentFolderId changes
    // Only expand parents, never collapse - this preserves user's expanded state
    useEffect(() => {
        if (currentFolderId && tree.length > 0) {
            // Find the path to the current folder and expand all parents
            const findPath = (nodes: FolderTreeNode[], id: string, path: string[]): string[] | null => {
                for (const node of nodes) {
                    if (node.id === id) {
                        return path
                    }
                    if (node.children && node.children.length > 0) {
                        const result = findPath(node.children, id, [...path, node.id])
                        if (result) return result
                    }
                }
                return null
            }

            const path = findPath(tree, currentFolderId, [])
            if (path && path.length > 0) {
                setExpandedFolders(prev => {
                    // Only add to the set, never remove - this preserves user's expanded state
                    const next = new Set(prev)
                    path.forEach(id => next.add(id))
                    return next
                })
            }
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [currentFolderId])

    const loadTree = async () => {
        try {
            setLoading(true)
            const data = await api.getFolderTree()
            setTree(data || [])
        } catch (error) {
            console.error('Failed to load folder tree:', error)
        } finally {
            setLoading(false)
        }
    }


    const toggleExpand = (folderId: string, e: React.MouseEvent) => {
        e.stopPropagation()
        setExpandedFolders(prev => {
            const next = new Set(prev)
            if (next.has(folderId)) {
                next.delete(folderId)
            } else {
                next.add(folderId)
            }
            return next
        })
    }

    const handleFolderClick = (folderId: string) => {
        onFolderSelect(folderId)
    }

    const handleRootClick = () => {
        onFolderSelect(undefined)
    }

    const renderNode = (node: FolderTreeNode) => {
        const isExpanded = expandedFolders.has(node.id)
        const isSelected = currentFolderId === node.id
        const hasChildren = (node.children && node.children.length > 0) || (node.documents && node.documents.length > 0)

        return (
            <div key={node.id}>
                <div
                    className={`flex items-center gap-1 py-1.5 px-2 rounded-lg cursor-pointer transition-colors ${isSelected
                        ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                        : 'hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300'
                        }`}
                    style={{ paddingLeft: `${(node.level + 1) * 12}px` }}
                    onClick={() => handleFolderClick(node.id)}
                >
                    {/* Expand/Collapse toggle */}
                    <button
                        onClick={(e) => toggleExpand(node.id, e)}
                        className={`p-0.5 rounded hover:bg-slate-200 dark:hover:bg-slate-600 ${!hasChildren ? 'invisible' : ''
                            }`}
                    >
                        {isExpanded ? (
                            <ChevronDown className="w-3.5 h-3.5" />
                        ) : (
                            <ChevronRight className="w-3.5 h-3.5" />
                        )}
                    </button>

                    {/* Folder icon */}
                    {isExpanded ? (
                        <FolderOpen className="w-4 h-4 text-amber-500 flex-shrink-0" />
                    ) : (
                        <Folder className="w-4 h-4 text-amber-500 flex-shrink-0" />
                    )}

                    {/* Folder name */}
                    <span className="text-sm truncate flex-1">{node.name}</span>

                    {/* Document count */}
                    {node.doc_count > 0 && (
                        <span className="text-xs text-slate-400 flex-shrink-0">{node.doc_count}</span>
                    )}
                </div>

                {/* Children and Documents */}
                {hasChildren && isExpanded && (
                    <div>
                        {/* Render subfolders first */}
                        {node.children && node.children.map(child => renderNode(child))}

                        {/* Render documents */}
                        {node.documents && node.documents.map(doc => renderDocument(doc, node.level + 1))}
                    </div>
                )}
            </div>
        )
    }

    const renderDocument = (doc: { id: string; title: string }, level: number) => {
        return (
            <div
                key={doc.id}
                className="flex items-center gap-1 py-1.5 px-2 rounded-lg cursor-pointer transition-colors hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-400"
                style={{ paddingLeft: `${(level + 2) * 12}px` }}
                onClick={() => router.push(`/doc/${doc.id}`)}
            >
                {/* Spacer for alignment with folders */}
                <div className="w-4 h-4 flex-shrink-0" />

                {/* Document icon */}
                <FileText className="w-4 h-4 text-primary-500 flex-shrink-0" />

                {/* Document title */}
                <span className="text-sm truncate flex-1">{doc.title}</span>
            </div>
        )
    }


    if (!isOpen) return null

    return (
        <aside className="fixed left-0 top-0 bottom-0 w-64 bg-white dark:bg-slate-800 border-r border-slate-200 dark:border-slate-700 z-30 shadow-lg flex flex-col">
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200 dark:border-slate-700 flex-shrink-0">
                <div className="flex items-center gap-2">
                    <FolderTree className="w-4 h-4 text-slate-500" />
                    <span className="font-medium text-slate-900 dark:text-white text-sm">文件夹</span>
                </div>
                <button
                    onClick={onToggle}
                    className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors"
                    title="隐藏文件夹树"
                >
                    <ChevronLeft className="w-4 h-4 text-slate-500" />
                </button>
            </div>

            {/* Content */}
            <div className="flex-1 overflow-y-auto py-2 px-2">
                {loading ? (
                    <div className="flex items-center justify-center py-8">
                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-500"></div>
                    </div>
                ) : (
                    <>
                        {/* Root folder */}
                        <div
                            className={`flex items-center gap-2 py-1.5 px-2 rounded-lg cursor-pointer transition-colors ${!currentFolderId
                                ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                                : 'hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300'
                                }`}
                            onClick={handleRootClick}
                        >
                            <Home className="w-4 h-4 text-slate-500 flex-shrink-0" />
                            <span className="text-sm font-medium">我的文档</span>
                        </div>

                        {/* Folder tree */}
                        {tree.length === 0 ? (
                            <div className="px-2 py-4 text-center text-slate-400 text-xs">
                                暂无文件夹
                            </div>
                        ) : (
                            <div className="mt-1">
                                {tree.map(node => renderNode(node))}
                            </div>
                        )}
                    </>
                )}
            </div>

            {/* Refresh button */}
            <div className="flex-shrink-0 px-3 py-2 border-t border-slate-200 dark:border-slate-700">
                <button
                    onClick={loadTree}
                    className="w-full py-1.5 text-xs text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors"
                >
                    刷新文件夹树
                </button>
            </div>
        </aside>
    )
}
