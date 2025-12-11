'use client'

import { useEffect, useState, useRef } from 'react'
import { Editor } from '@tiptap/react'
import { List, ChevronLeft } from 'lucide-react'

interface HeadingItem {
    id: string
    level: number
    text: string
    pos: number
}

interface OutlineSidebarProps {
    editor: Editor | null
    isOpen: boolean
    onToggle: () => void
}

export default function OutlineSidebar({ editor, isOpen, onToggle }: OutlineSidebarProps) {
    const [headings, setHeadings] = useState<HeadingItem[]>([])
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

    useEffect(() => {
        if (!editor) return

        const updateHeadings = () => {
            const items: HeadingItem[] = []
            const doc = editor.state.doc

            doc.descendants((node, pos) => {
                if (node.type.name === 'heading') {
                    const id = `heading-${pos}`
                    items.push({
                        id,
                        level: node.attrs.level,
                        text: node.textContent || '(无标题)',
                        pos,
                    })
                }
            })

            setHeadings(items)
        }

        // Initial update
        updateHeadings()

        // Listen for document changes
        editor.on('update', updateHeadings)

        return () => {
            editor.off('update', updateHeadings)
        }
    }, [editor])

    const scrollToHeading = (pos: number) => {
        if (!editor) return

        // Set cursor to the heading position
        editor.chain().focus().setTextSelection(pos + 1).run()
    }

    const getIndentClass = (level: number) => {
        switch (level) {
            case 1: return 'pl-3'
            case 2: return 'pl-6'
            case 3: return 'pl-9'
            case 4: return 'pl-12'
            case 5: return 'pl-14'
            case 6: return 'pl-16'
            default: return 'pl-3'
        }
    }

    const getLevelStyle = (level: number) => {
        switch (level) {
            case 1: return 'text-sm font-semibold text-slate-800 dark:text-slate-200'
            case 2: return 'text-sm font-medium text-slate-700 dark:text-slate-300'
            case 3: return 'text-xs font-medium text-slate-600 dark:text-slate-400'
            default: return 'text-xs text-slate-500 dark:text-slate-500'
        }
    }

    if (!isOpen) return null

    return (
        <aside
            className="fixed left-0 bottom-0 w-64 bg-white dark:bg-slate-800 border-r border-slate-200 dark:border-slate-700 z-30 shadow-lg flex flex-col"
            style={{ top: `${headerHeight}px` }}
        >
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200 dark:border-slate-700">
                <div className="flex items-center gap-2">
                    <List className="w-4 h-4 text-slate-500" />
                    <span className="font-medium text-slate-900 dark:text-white text-sm">目录</span>
                </div>
                <button
                    onClick={onToggle}
                    className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors"
                    title="隐藏目录"
                >
                    <ChevronLeft className="w-4 h-4 text-slate-500" />
                </button>
            </div>

            {/* Content */}
            <div className="overflow-y-auto flex-1 pb-4">
                {headings.length === 0 ? (
                    <div className="px-4 py-8 text-center">
                        <p className="text-slate-400 text-sm">暂无标题</p>
                        <p className="mt-2 text-xs text-slate-400">
                            添加 H1、H2、H3 等标题
                            <br />
                            来自动生成目录
                        </p>
                    </div>
                ) : (
                    <nav className="py-2">
                        {headings.map((heading) => (
                            <button
                                key={heading.id}
                                onClick={() => scrollToHeading(heading.pos)}
                                className={`w-full text-left py-2 pr-4 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors ${getIndentClass(heading.level)}`}
                            >
                                <span className={`block truncate ${getLevelStyle(heading.level)}`}>
                                    {heading.text}
                                </span>
                            </button>
                        ))}
                    </nav>
                )}
            </div>
        </aside>
    )
}
