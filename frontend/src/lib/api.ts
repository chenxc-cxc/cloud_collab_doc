import type { User, Document, Comment, DocumentPermission, LoginResponse } from '@/types'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

class ApiClient {
    private token: string | null = null
    private userId: string | null = null

    constructor() {
        if (typeof window !== 'undefined') {
            this.token = localStorage.getItem('token')
            this.userId = localStorage.getItem('userId') || '11111111-1111-1111-1111-111111111111'
        }
    }

    private async fetch<T>(path: string, options: RequestInit = {}): Promise<T> {
        const headers: Record<string, string> = {
            'Content-Type': 'application/json',
            ...options.headers as Record<string, string>,
        }

        if (this.token) {
            headers['Authorization'] = `Bearer ${this.token}`
        }

        if (this.userId) {
            headers['X-User-ID'] = this.userId
        }

        const response = await fetch(`${API_URL}${path}`, {
            ...options,
            headers,
        })

        if (!response.ok) {
            const error = await response.json().catch(() => ({ error: 'Unknown error' }))
            throw new Error(error.error || `HTTP ${response.status}`)
        }

        return response.json()
    }

    setToken(token: string) {
        this.token = token
        if (typeof window !== 'undefined') {
            localStorage.setItem('token', token)
        }
    }

    setUserId(userId: string) {
        this.userId = userId
        if (typeof window !== 'undefined') {
            localStorage.setItem('userId', userId)
        }
    }

    // Auth
    async login(email: string): Promise<LoginResponse> {
        const response = await this.fetch<LoginResponse>('/api/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email }),
        })
        this.setToken(response.token)
        this.setUserId(response.user.id)
        return response
    }

    async getCurrentUser(): Promise<User> {
        return this.fetch<User>('/api/auth/me')
    }

    // Documents
    async listDocuments(): Promise<Document[]> {
        return this.fetch<Document[]>('/api/docs')
    }

    async getDocument(id: string): Promise<Document> {
        return this.fetch<Document>(`/api/docs/${id}`)
    }

    async createDocument(title: string): Promise<Document> {
        return this.fetch<Document>('/api/docs', {
            method: 'POST',
            body: JSON.stringify({ title }),
        })
    }

    async updateDocument(id: string, data: { title: string }): Promise<Document> {
        return this.fetch<Document>(`/api/docs/${id}`, {
            method: 'PUT',
            body: JSON.stringify(data),
        })
    }

    async deleteDocument(id: string): Promise<void> {
        await this.fetch(`/api/docs/${id}`, { method: 'DELETE' })
    }

    // Get current user's permission for a document
    async getMyPermission(docId: string): Promise<string> {
        try {
            const result = await this.fetch<{ role: string }>(`/api/docs/${docId}/my-permission`)
            return result.role || 'view'
        } catch {
            return 'view'
        }
    }

    // Permissions
    async listPermissions(docId: string): Promise<DocumentPermission[]> {
        return this.fetch<DocumentPermission[]>(`/api/docs/${docId}/permissions`)
    }

    async setPermission(docId: string, userId: string, role: string): Promise<void> {
        await this.fetch(`/api/docs/${docId}/permissions`, {
            method: 'PUT',
            body: JSON.stringify({ user_id: userId, role }),
        })
    }

    async removePermission(docId: string, userId: string): Promise<void> {
        await this.fetch(`/api/docs/${docId}/permissions/${userId}`, {
            method: 'DELETE',
        })
    }

    // Comments
    async listComments(docId: string): Promise<Comment[]> {
        return this.fetch<Comment[]>(`/api/docs/${docId}/comments`)
    }

    async createComment(docId: string, data: { content: string; selection?: { anchor: number; head: number } }): Promise<Comment> {
        return this.fetch<Comment>(`/api/docs/${docId}/comments`, {
            method: 'POST',
            body: JSON.stringify(data),
        })
    }

    async updateComment(id: string, data: { content?: string; resolved?: boolean }): Promise<Comment> {
        return this.fetch<Comment>(`/api/comments/${id}`, {
            method: 'PUT',
            body: JSON.stringify(data),
        })
    }

    async deleteComment(id: string): Promise<void> {
        await this.fetch(`/api/comments/${id}`, { method: 'DELETE' })
    }

    // Get WebSocket URL
    getWebSocketUrl(docId: string): string {
        const wsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081'
        const params = new URLSearchParams()
        if (this.token) params.set('token', this.token)
        if (this.userId) params.set('userId', this.userId)
        return `${wsUrl}/ws/collab/${docId}?${params.toString()}`
    }
}

export const api = new ApiClient()
