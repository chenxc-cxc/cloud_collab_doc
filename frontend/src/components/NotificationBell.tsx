'use client'

import { useState, useEffect, useRef } from 'react'
import { Bell, CheckCircle, XCircle, Clock, Loader2, FileText } from 'lucide-react'
import { api } from '@/lib/api'
import type { AccessRequest } from '@/types'

export default function NotificationBell() {
    const [isOpen, setIsOpen] = useState(false)
    const [requests, setRequests] = useState<AccessRequest[]>([])
    const [loading, setLoading] = useState(true)
    const [processingId, setProcessingId] = useState<string | null>(null)
    const menuRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        loadRequests()
    }, [])

    useEffect(() => {
        function handleClickOutside(event: MouseEvent) {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                setIsOpen(false)
            }
        }
        document.addEventListener('mousedown', handleClickOutside)
        return () => document.removeEventListener('mousedown', handleClickOutside)
    }, [])

    const loadRequests = async () => {
        try {
            const data = await api.listMyPendingAccessRequests()
            setRequests(data)
        } catch (error) {
            console.error('Failed to load access requests:', error)
        } finally {
            setLoading(false)
        }
    }

    const handleAction = async (requestId: string, action: 'approved' | 'rejected') => {
        setProcessingId(requestId)
        try {
            await api.updateAccessRequest(requestId, action)
            // Remove from list after action
            setRequests(prev => prev.filter(r => r.id !== requestId))
        } catch (error) {
            console.error('Failed to update request:', error)
        } finally {
            setProcessingId(null)
        }
    }

    const pendingCount = requests.length

    return (
        <div className="relative" ref={menuRef}>
            <button
                onClick={() => setIsOpen(!isOpen)}
                className="relative p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
            >
                <Bell className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                {pendingCount > 0 && (
                    <span className="absolute -top-1 -right-1 w-5 h-5 bg-red-500 text-white text-xs font-bold rounded-full flex items-center justify-center">
                        {pendingCount > 9 ? '9+' : pendingCount}
                    </span>
                )}
            </button>

            {isOpen && (
                <div className="absolute right-0 mt-2 w-80 bg-white dark:bg-slate-800 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700 z-50 overflow-hidden">
                    <div className="px-4 py-3 border-b border-slate-200 dark:border-slate-700">
                        <h3 className="font-semibold text-slate-900 dark:text-white">
                            Access Requests
                        </h3>
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                            {pendingCount === 0 ? 'No pending requests' : `${pendingCount} pending request${pendingCount > 1 ? 's' : ''}`}
                        </p>
                    </div>

                    <div className="max-h-96 overflow-y-auto">
                        {loading ? (
                            <div className="p-4 text-center">
                                <Loader2 className="w-6 h-6 animate-spin text-slate-400 mx-auto" />
                            </div>
                        ) : requests.length === 0 ? (
                            <div className="p-6 text-center">
                                <Bell className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-2" />
                                <p className="text-sm text-slate-500 dark:text-slate-400">
                                    No pending access requests
                                </p>
                            </div>
                        ) : (
                            <div className="divide-y divide-slate-100 dark:divide-slate-700">
                                {requests.map((request) => (
                                    <div key={request.id} className="p-3 hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors">
                                        <div className="flex items-start gap-3">
                                            <div className="w-9 h-9 rounded-full bg-gradient-to-br from-amber-400 to-orange-500 flex items-center justify-center text-white text-sm font-medium flex-shrink-0">
                                                {(request.requester?.name || 'U').charAt(0).toUpperCase()}
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <p className="text-sm font-medium text-slate-900 dark:text-white">
                                                    {request.requester?.name || 'Unknown User'}
                                                </p>
                                                <p className="text-xs text-slate-500 dark:text-slate-400 flex items-center gap-1">
                                                    <FileText className="w-3 h-3" />
                                                    <span className="truncate">{request.document?.title || 'Unknown Document'}</span>
                                                </p>
                                                <p className="text-xs text-slate-400 dark:text-slate-500 flex items-center gap-1 mt-0.5">
                                                    <Clock className="w-3 h-3" />
                                                    Wants {request.requested_role} access
                                                </p>
                                            </div>
                                            <div className="flex items-center gap-1">
                                                {processingId === request.id ? (
                                                    <Loader2 className="w-5 h-5 animate-spin text-slate-400" />
                                                ) : (
                                                    <>
                                                        <button
                                                            onClick={() => handleAction(request.id, 'approved')}
                                                            className="p-1.5 text-green-600 hover:bg-green-100 dark:hover:bg-green-900/30 rounded-lg transition-colors"
                                                            title="Approve"
                                                        >
                                                            <CheckCircle className="w-5 h-5" />
                                                        </button>
                                                        <button
                                                            onClick={() => handleAction(request.id, 'rejected')}
                                                            className="p-1.5 text-red-500 hover:bg-red-100 dark:hover:bg-red-900/30 rounded-lg transition-colors"
                                                            title="Reject"
                                                        >
                                                            <XCircle className="w-5 h-5" />
                                                        </button>
                                                    </>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    )
}
