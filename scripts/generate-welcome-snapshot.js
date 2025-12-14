#!/usr/bin/env node

/**
 * Welcome Document Snapshot Generator
 * 
 * 这个脚本生成预填充内容的 Yjs 文档快照，
 * 用于在新用户注册时创建欢迎文档。
 * 
 * 使用方法：
 *   cd scripts
 *   npm install yjs (如果尚未安装)
 *   node generate-welcome-snapshot.js
 * 
 * 然后将输出的 Go byte slice 复制到 backend/internal/db/db.go 中的
 * getWelcomeDocumentSnapshot() 函数。
 */

const Y = require('yjs');

// ============================================================================
// 自定义欢迎内容 - 修改这里来改变欢迎文档的内容
// ============================================================================
const welcomeContent = [
    {
        type: 'heading',
        attrs: { level: 1 },
        content: [{ type: 'text', text: '欢迎使用 CollabDocs! 🎉' }]
    },
    {
        type: 'paragraph',
        content: [{ type: 'text', text: '这是你的第一个文档。CollabDocs 是一个实时协作文档平台，让团队协作变得简单高效。' }]
    },
    {
        type: 'heading',
        attrs: { level: 2 },
        content: [{ type: 'text', text: '✨ 快速开始' }]
    },
    {
        type: 'paragraph',
        content: [{ type: 'text', text: '以下是一些帮助你上手的小技巧：' }]
    },
    {
        type: 'bulletList',
        content: [
            {
                type: 'listItem',
                content: [{
                    type: 'paragraph',
                    content: [
                        { type: 'text', marks: [{ type: 'bold' }], text: '实时协作' },
                        { type: 'text', text: ' - 邀请团队成员一起编辑，所有更改实时同步' }
                    ]
                }]
            },
            {
                type: 'listItem',
                content: [{
                    type: 'paragraph',
                    content: [
                        { type: 'text', marks: [{ type: 'bold' }], text: '评论功能' },
                        { type: 'text', text: ' - 选中文本添加评论，进行讨论和反馈' }
                    ]
                }]
            },
            {
                type: 'listItem',
                content: [{
                    type: 'paragraph',
                    content: [
                        { type: 'text', marks: [{ type: 'bold' }], text: '权限管理' },
                        { type: 'text', text: ' - 精细控制谁可以查看、评论或编辑文档' }
                    ]
                }]
            },
            {
                type: 'listItem',
                content: [{
                    type: 'paragraph',
                    content: [
                        { type: 'text', marks: [{ type: 'bold' }], text: '版本历史' },
                        { type: 'text', text: ' - 自动保存，随时查看历史版本' }
                    ]
                }]
            }
        ]
    },
    {
        type: 'heading',
        attrs: { level: 2 },
        content: [{ type: 'text', text: '📝 试试看' }]
    },
    {
        type: 'paragraph',
        content: [{ type: 'text', text: '直接开始编辑这个文档，或者点击左侧边栏的「新建文档」创建一个全新的文档。' }]
    },
    {
        type: 'paragraph',
        content: [{ type: 'text', text: '' }]
    },
    {
        type: 'paragraph',
        content: [
            { type: 'text', text: '祝你使用愉快！如有问题，请随时联系我们。' }
        ]
    }
];

// ============================================================================
// 生成 Yjs 快照
// ============================================================================

// 创建 Yjs 文档
const ydoc = new Y.Doc();

// TipTap with Collaboration uses prosemirror fragment
// The default shared type name is 'default' for Collaboration extension
const yXmlFragment = ydoc.getXmlFragment('default');

// 将 ProseMirror JSON 转换为 Yjs XML Fragment
function jsonToYXml(json, parent) {
    if (Array.isArray(json)) {
        json.forEach(node => jsonToYXml(node, parent));
        return;
    }

    if (json.type === 'text') {
        const text = new Y.XmlText();
        text.insert(0, json.text || '');

        // 应用标记 (bold, italic, etc.)
        if (json.marks && json.marks.length > 0) {
            const attrs = {};
            json.marks.forEach(mark => {
                attrs[mark.type] = mark.attrs || true;
            });
            text.format(0, (json.text || '').length, attrs);
        }

        parent.push([text]);
        return;
    }

    // 创建元素节点
    const element = new Y.XmlElement(json.type);

    // 设置属性
    if (json.attrs) {
        Object.entries(json.attrs).forEach(([key, value]) => {
            element.setAttribute(key, value);
        });
    }

    // 递归处理子节点
    if (json.content) {
        jsonToYXml(json.content, element);
    }

    parent.push([element]);
}

// 创建文档根节点
const docNode = new Y.XmlElement('doc');
jsonToYXml(welcomeContent, docNode);
yXmlFragment.push([docNode]);

// 导出状态
const state = Y.encodeStateAsUpdate(ydoc);

// ============================================================================
// 输出各种格式
// ============================================================================

console.log('\n========================================');
console.log('Welcome Document Snapshot Generator');
console.log('========================================\n');

// 输出字节数组
console.log('Bytes length:', state.length);
console.log('');

// Go 格式
console.log('📋 Go byte slice (复制到 getWelcomeDocumentSnapshot 函数):');
console.log('----------------------------------------');
console.log('return []byte{' + Array.from(state).join(', ') + '}');
console.log('----------------------------------------\n');

// Base64 格式 (备用)
console.log('📋 Base64 (备用格式):');
console.log('----------------------------------------');
console.log(Buffer.from(state).toString('base64'));
console.log('----------------------------------------\n');

// 验证信息
console.log('✅ 生成完成！');
console.log('');
console.log('下一步:');
console.log('1. 复制上面的 Go byte slice');
console.log('2. 打开 backend/internal/db/db.go');
console.log('3. 找到 getWelcomeDocumentSnapshot() 函数');
console.log('4. 将 return nil 替换为复制的内容');
console.log('5. 重新编译后端: cd backend && go build ./...');
console.log('');
