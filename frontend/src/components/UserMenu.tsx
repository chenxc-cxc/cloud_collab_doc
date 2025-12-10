'use client'

import { useState, useRef, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { User, LogOut, Key, ChevronDown } from 'lucide-react'
import { api } from '@/lib/api'
import { useStore } from '@/lib/store'

interface UserMenuProps {
    onChangePassword?: () => void
}

export function UserMenu({ onChangePassword }: UserMenuProps) {
    const router = useRouter()
    const { currentUser } = useStore()
    const [isOpen, setIsOpen] = useState(false)
    const menuRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        function handleClickOutside(event: MouseEvent) {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                setIsOpen(false)
            }
        }
        document.addEventListener('mousedown', handleClickOutside)
        return () => document.removeEventListener('mousedown', handleClickOutside)
    }, [])

    const handleLogout = async () => {
        try {
            await api.logout()
        } catch (error) {
            console.error('Logout failed:', error)
        }
        router.push('/login')
    }

    if (!currentUser) return null

    return (
        <div className="relative" ref={menuRef}>
            <button
                onClick={() => setIsOpen(!isOpen)}
                className="flex items-center gap-2 px-3 py-2 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 transition-colors"
            >
                <div className="w-8 h-8 rounded-full bg-gradient-to-r from-purple-500 to-pink-500 flex items-center justify-center text-white font-medium text-sm">
                    {currentUser.name?.charAt(0).toUpperCase() || 'U'}
                </div>
                <span className="text-slate-700 dark:text-slate-200 text-sm font-medium hidden sm:block">
                    {currentUser.name}
                </span>
                <ChevronDown className={`w-4 h-4 text-slate-400 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
            </button>

            {isOpen && (
                <div className="absolute right-0 mt-2 w-56 bg-slate-800 rounded-xl shadow-xl border border-slate-700 overflow-hidden z-50">
                    <div className="p-3 border-b border-slate-700">
                        <p className="text-white font-medium">{currentUser.name}</p>
                        <p className="text-slate-400 text-sm truncate">{currentUser.email}</p>
                    </div>
                    <div className="p-1">
                        {onChangePassword && (
                            <button
                                onClick={() => {
                                    setIsOpen(false)
                                    onChangePassword()
                                }}
                                className="flex items-center gap-3 w-full px-3 py-2 text-slate-300 hover:bg-slate-700 rounded-lg transition-colors"
                            >
                                <Key className="w-4 h-4" />
                                Change Password
                            </button>
                        )}
                        <button
                            onClick={handleLogout}
                            className="flex items-center gap-3 w-full px-3 py-2 text-red-400 hover:bg-slate-700 rounded-lg transition-colors"
                        >
                            <LogOut className="w-4 h-4" />
                            Sign Out
                        </button>
                    </div>
                </div>
            )}
        </div>
    )
}
