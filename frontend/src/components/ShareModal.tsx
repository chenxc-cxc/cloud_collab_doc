'use client'

import { useState, useEffect } from 'react'
import { X, UserPlus, Trash2, Copy, Check, CheckCircle, XCircle, Clock, Loader2 } from 'lucide-react'
import { api } from '@/lib/api'
import type { DocumentPermission, PermissionRole, AccessRequest } from '@/types'

interface ShareModalProps {
    docId: string
    onClose: () => void
}

export default function ShareModal({ docId, onClose }: ShareModalProps) {
    const [permissions, setPermissions] = useState<DocumentPermission[]>([])
    const [accessRequests, setAccessRequests] = useState<AccessRequest[]>([])
    const [loading, setLoading] = useState(true)
    const [email, setEmail] = useState('')
    const [role, setRole] = useState<PermissionRole>('edit')
    const [copied, setCopied] = useState(false)
    const [processingRequest, setProcessingRequest] = useState<string | null>(null)
    useEffect(() => {
        loadData()
    }, [docId])

    const loadData = async () => {
        try {
            const [perms, requests] = await Promise.all([
                api.listPermissions(docId),
                api.listAccessRequests(docId).catch(() => [])
            ])
            setPermissions(perms)
            setAccessRequests(requests)
        } catch (error) {
            console.error('Failed to load data:', error)
        } finally {
            setLoading(false)
        }
    }

    const addPermission = async (userId: string) => {
        try {
            await api.setPermission(docId, userId, role)
            loadData()
            setEmail('')
        } catch (error) {
            console.error('Failed to add permission:', error)
        }
    }

    const updatePermission = async (userId: string, newRole: PermissionRole) => {
        try {
            await api.setPermission(docId, userId, newRole)
            loadData()
        } catch (error) {
            console.error('Failed to update permission:', error)
        }
    }

    const removePermission = async (userId: string) => {
        if (!confirm('Remove this user\'s access?')) return

        try {
            await api.removePermission(docId, userId)
            loadData()
        } catch (error) {
            console.error('Failed to remove permission:', error)
        }
    }

    const handleAccessRequest = async (requestId: string, action: 'approved' | 'rejected') => {
        setProcessingRequest(requestId)
        try {
            await api.updateAccessRequest(requestId, action)
            loadData()
        } catch (error) {
            console.error('Failed to update access request:', error)
        } finally {
            setProcessingRequest(null)
        }
    }

    const copyLink = () => {
        const url = `${window.location.origin}/doc/${docId}`
        navigator.clipboard.writeText(url)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    const getRoleBadgeColor = (role: string) => {
        switch (role) {
            case 'owner': return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
            case 'edit': return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
            case 'comment': return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
            default: return 'bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400'
        }
    }

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 animate-fade-in">
            <div className="bg-white dark:bg-slate-800 rounded-xl shadow-xl w-full max-w-md mx-4 animate-slide-up">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700">
                    <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                        Share Document
                    </h2>
                    <button
                        onClick={onClose}
                        className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
                    >
                        <X className="w-5 h-5 text-slate-500" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 space-y-6">
                    {/* Copy link */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Share link
                        </label>
                        <div className="flex gap-2">
                            <input
                                type="text"
                                value={`${typeof window !== 'undefined' ? window.location.origin : ''}/doc/${docId}`}
                                readOnly
                                className="flex-1 px-3 py-2 text-sm bg-slate-50 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded-lg text-slate-600 dark:text-slate-400"
                            />
                            <button
                                onClick={copyLink}
                                className={`px-3 py-2 rounded-lg text-sm font-medium transition-colors ${copied
                                    ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                                    : 'bg-slate-100 hover:bg-slate-200 dark:bg-slate-700 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300'
                                    }`}
                            >
                                {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                            </button>
                        </div>
                    </div>

                    {/* Add user (development) */}
                    {/* Add people by email */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Add people
                        </label>
                        <div className="flex flex-col gap-3">
                            <input
                                type="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="Enter email address"
                                className="w-full px-3 py-2 text-sm bg-slate-50 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded-lg text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' && email) {
                                        // We need to implement lookup by email or invite
                                        // For now, let's just log it or show a temporary alert
                                        // api.addPermissionByEmail(docId, email, role)
                                        console.log('Invite email:', email, role)
                                    }
                                }}
                            />
                            <div className="flex gap-2 justify-end">
                                <select
                                    value={role}
                                    onChange={(e) => setRole(e.target.value as PermissionRole)}
                                    className="px-3 py-2 text-sm bg-slate-50 dark:bg-slate-700 border border-slate-200 dark:border-slate-600 rounded-lg text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                                >
                                    <option value="edit">Can edit</option>
                                    <option value="comment">Can comment</option>
                                    <option value="view">Can view</option>
                                </select>
                                <button
                                    onClick={() => {
                                        if (email) {
                                            // This functionality might need backend support for looking up user by email
                                            // For now, we'll keep the UI but maybe notify it's not fully implemented if backend is missing
                                            // Or if you want to keep using ID, we can't.

                                            alert('Email invitation is not yet fully implemented in the backend (needs lookup by email).')
                                        }
                                    }}
                                    disabled={!email}
                                    className="px-4 py-2 bg-primary-500 hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
                                >
                                    <UserPlus className="w-4 h-4" />
                                    <span>Invite</span>
                                </button>
                            </div>
                        </div>
                    </div>

                    {/* Current permissions */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            People with access
                        </label>
                        {loading ? (
                            <div className="text-center py-4 text-slate-500">Loading...</div>
                        ) : (
                            <div className="space-y-2">
                                {permissions.map((perm) => (
                                    <div
                                        key={perm.user_id}
                                        className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-700/50 rounded-lg"
                                    >
                                        <div className="flex items-center gap-3">
                                            <div className="w-8 h-8 rounded-full bg-primary-500 flex items-center justify-center text-white text-sm font-medium">
                                                {(perm.user?.name || 'U').charAt(0).toUpperCase()}
                                            </div>
                                            <div>
                                                <p className="text-sm font-medium text-slate-900 dark:text-white">
                                                    {perm.user?.name || 'Unknown'}
                                                </p>
                                                <p className="text-xs text-slate-500 dark:text-slate-400">
                                                    {perm.user?.email || perm.user_id}
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            {perm.role === 'owner' ? (
                                                <span className={`px-2 py-1 rounded-full text-xs font-medium ${getRoleBadgeColor(perm.role)}`}>
                                                    Owner
                                                </span>
                                            ) : (
                                                <>
                                                    <select
                                                        value={perm.role}
                                                        onChange={(e) => updatePermission(perm.user_id, e.target.value as PermissionRole)}
                                                        className="px-2 py-1 text-xs bg-white dark:bg-slate-600 border border-slate-200 dark:border-slate-500 rounded-lg"
                                                    >
                                                        <option value="edit">Can edit</option>
                                                        <option value="comment">Can comment</option>
                                                        <option value="view">Can view</option>
                                                    </select>
                                                    <button
                                                        onClick={() => removePermission(perm.user_id)}
                                                        className="p-1.5 text-slate-400 hover:text-red-500 transition-colors"
                                                    >
                                                        <Trash2 className="w-4 h-4" />
                                                    </button>
                                                </>
                                            )}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Pending access requests */}
                    {accessRequests.filter(r => r.status === 'pending').length > 0 && (
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Pending Access Requests
                            </label>
                            <div className="space-y-2">
                                {accessRequests.filter(r => r.status === 'pending').map((request) => (
                                    <div
                                        key={request.id}
                                        className="flex items-center justify-between p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg"
                                    >
                                        <div className="flex items-center gap-3">
                                            <div className="w-8 h-8 rounded-full bg-amber-500 flex items-center justify-center text-white text-sm font-medium">
                                                {(request.requester?.name || 'U').charAt(0).toUpperCase()}
                                            </div>
                                            <div>
                                                <p className="text-sm font-medium text-slate-900 dark:text-white">
                                                    {request.requester?.name || 'Unknown User'}
                                                </p>
                                                <p className="text-xs text-slate-500 dark:text-slate-400 flex items-center gap-1">
                                                    <Clock className="w-3 h-3" />
                                                    Requested {request.requested_role} access
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            {processingRequest === request.id ? (
                                                <Loader2 className="w-5 h-5 animate-spin text-slate-400" />
                                            ) : (
                                                <>
                                                    <button
                                                        onClick={() => handleAccessRequest(request.id, 'approved')}
                                                        className="p-1.5 text-green-600 hover:bg-green-100 dark:hover:bg-green-900/30 rounded-lg transition-colors"
                                                        title="Approve"
                                                    >
                                                        <CheckCircle className="w-5 h-5" />
                                                    </button>
                                                    <button
                                                        onClick={() => handleAccessRequest(request.id, 'rejected')}
                                                        className="p-1.5 text-red-500 hover:bg-red-100 dark:hover:bg-red-900/30 rounded-lg transition-colors"
                                                        title="Reject"
                                                    >
                                                        <XCircle className="w-5 h-5" />
                                                    </button>
                                                </>
                                            )}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex justify-end">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 bg-primary-500 hover:bg-primary-600 text-white rounded-lg text-sm font-medium transition-colors"
                    >
                        Done
                    </button>
                </div>
            </div>
        </div>
    )
}
