'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { useStore } from '@/lib/store'

interface AuthGuardProps {
    children: React.ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
    const router = useRouter()
    const { setCurrentUser } = useStore()
    const [isLoading, setIsLoading] = useState(true)
    const [isAuthenticated, setIsAuthenticated] = useState(false)

    useEffect(() => {
        checkAuth()
    }, [])

    const checkAuth = async () => {
        if (!api.isAuthenticated()) {
            router.push('/login')
            return
        }

        try {
            const user = await api.getCurrentUser()
            setCurrentUser(user) // 设置用户到 store
            setIsAuthenticated(true)
        } catch {
            // Token is invalid or expired
            api.clearAuth()
            setCurrentUser(null)
            router.push('/login')
        } finally {
            setIsLoading(false)
        }
    }

    if (isLoading) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500"></div>
            </div>
        )
    }

    if (!isAuthenticated) {
        return null
    }

    return <>{children}</>
}
