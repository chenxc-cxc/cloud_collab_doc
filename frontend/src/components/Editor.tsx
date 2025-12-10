'use client'

import { EditorContent, Editor } from '@tiptap/react'

interface EditorProps {
    editor: Editor
}

export default function EditorComponent({ editor }: EditorProps) {
    return (
        <div className="min-h-[500px]">
            <EditorContent editor={editor} />
        </div>
    )
}
