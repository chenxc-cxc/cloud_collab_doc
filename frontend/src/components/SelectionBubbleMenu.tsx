'use client'

import { BubbleMenu, Editor } from '@tiptap/react'
import {
    Bold,
    Italic,
    Underline,
    Strikethrough,
    Heading1,
    Heading2,
    Highlighter,
    MessageSquare
} from 'lucide-react'

interface SelectionBubbleMenuProps {
    editor: Editor
    onAddComment?: () => void
    canComment?: boolean
}

interface BubbleButtonProps {
    onClick: () => void
    isActive?: boolean
    title: string
    children: React.ReactNode
}

function BubbleButton({ onClick, isActive, title, children }: BubbleButtonProps) {
    return (
        <button
            onClick={onClick}
            title={title}
            className={`p-1.5 rounded transition-colors ${isActive
                    ? 'bg-white/20 text-white'
                    : 'hover:bg-white/10 text-white/80 hover:text-white'
                }`}
        >
            {children}
        </button>
    )
}

export default function SelectionBubbleMenu({
    editor,
    onAddComment,
    canComment = true
}: SelectionBubbleMenuProps) {
    if (!editor) return null

    return (
        <BubbleMenu
            editor={editor}
            tippyOptions={{
                duration: 100,
                placement: 'top',
            }}
            className="flex items-center gap-0.5 px-2 py-1.5 bg-slate-800 rounded-lg shadow-xl border border-slate-700"
        >
            {/* Headings */}
            <BubbleButton
                onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
                isActive={editor.isActive('heading', { level: 1 })}
                title="Heading 1"
            >
                <Heading1 className="w-4 h-4" />
            </BubbleButton>
            <BubbleButton
                onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
                isActive={editor.isActive('heading', { level: 2 })}
                title="Heading 2"
            >
                <Heading2 className="w-4 h-4" />
            </BubbleButton>

            <div className="w-px h-4 bg-slate-600 mx-1" />

            {/* Text formatting */}
            <BubbleButton
                onClick={() => editor.chain().focus().toggleBold().run()}
                isActive={editor.isActive('bold')}
                title="Bold"
            >
                <Bold className="w-4 h-4" />
            </BubbleButton>
            <BubbleButton
                onClick={() => editor.chain().focus().toggleItalic().run()}
                isActive={editor.isActive('italic')}
                title="Italic"
            >
                <Italic className="w-4 h-4" />
            </BubbleButton>
            <BubbleButton
                onClick={() => editor.chain().focus().toggleUnderline().run()}
                isActive={editor.isActive('underline')}
                title="Underline"
            >
                <Underline className="w-4 h-4" />
            </BubbleButton>
            <BubbleButton
                onClick={() => editor.chain().focus().toggleStrike().run()}
                isActive={editor.isActive('strike')}
                title="Strikethrough"
            >
                <Strikethrough className="w-4 h-4" />
            </BubbleButton>
            <BubbleButton
                onClick={() => editor.chain().focus().toggleHighlight().run()}
                isActive={editor.isActive('highlight')}
                title="Highlight"
            >
                <Highlighter className="w-4 h-4" />
            </BubbleButton>

            {/* Add Comment button */}
            {canComment && onAddComment && (
                <>
                    <div className="w-px h-4 bg-slate-600 mx-1" />
                    <BubbleButton
                        onClick={onAddComment}
                        title="Add Comment"
                    >
                        <MessageSquare className="w-4 h-4" />
                    </BubbleButton>
                </>
            )}
        </BubbleMenu>
    )
}
