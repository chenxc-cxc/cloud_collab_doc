'use client'

import { EditorContent, Editor } from '@tiptap/react'

interface EditorProps {
    editor: Editor
}

export default function EditorComponent({ editor }: EditorProps) {
    return (
        <div className="h-full min-h-[calc(100vh-12rem)]">
            <EditorContent
                editor={editor}
                className="h-full focus:outline-none"
            />
        </div>
    )
}
