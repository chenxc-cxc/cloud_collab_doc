#!/usr/bin/env node

/**
 * Custom y-websocket server with persistence to Go backend
 */

const http = require('http')
const WebSocket = require('ws')
const Y = require('yjs')
const { setupWSConnection, setPersistence } = require('y-websocket/bin/utils')

const PORT = process.env.PORT || 1234
const API_URL = process.env.API_URL || 'http://api-service:8080'

console.log(`y-websocket server starting...`)
console.log(`  Port: ${PORT}`)
console.log(`  API URL: ${API_URL}`)

// Persistence layer - saves/loads documents to/from Go backend
const persistence = {
    bindState: async (docName, ydoc) => {
        // docName is the document ID
        console.log(`Loading document: ${docName}`)

        try {
            const response = await fetch(`${API_URL}/api/yjs/${docName}/snapshot`)

            if (response.ok) {
                const data = await response.json()
                if (data.snapshot) {
                    // Decode base64 snapshot
                    const snapshotBuffer = Buffer.from(data.snapshot, 'base64')
                    Y.applyUpdate(ydoc, snapshotBuffer)
                    console.log(`Loaded snapshot for ${docName} (${snapshotBuffer.length} bytes)`)
                } else {
                    console.log(`No existing snapshot for ${docName}`)
                }
            } else if (response.status === 404) {
                console.log(`No snapshot found for ${docName}`)
            } else {
                console.error(`Failed to load snapshot for ${docName}: ${response.status}`)
            }
        } catch (error) {
            console.error(`Error loading document ${docName}:`, error.message)
        }
    },

    writeState: async (docName, ydoc) => {
        // Save document snapshot to backend
        const snapshot = Y.encodeStateAsUpdate(ydoc)

        // Skip saving empty documents (Yjs empty state is 2 bytes)
        // This prevents overwriting real content when room is destroyed
        if (snapshot.length <= 2) {
            console.log(`Skipping empty document save: ${docName}`)
            return
        }

        console.log(`Saving document: ${docName} (${snapshot.length} bytes)`)

        try {
            const snapshotBase64 = Buffer.from(snapshot).toString('base64')

            const response = await fetch(`${API_URL}/api/yjs/${docName}/snapshot`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    snapshot: snapshotBase64,
                }),
            })

            if (response.ok) {
                console.log(`Saved snapshot for ${docName}`)
            } else {
                console.error(`Failed to save snapshot for ${docName}: ${response.status}`)
            }
        } catch (error) {
            console.error(`Error saving document ${docName}:`, error.message)
        }
    },
}

// Set persistence
setPersistence(persistence)

// Create HTTP server
const server = http.createServer((request, response) => {
    if (request.url === '/health') {
        response.writeHead(200, { 'Content-Type': 'application/json' })
        response.end(JSON.stringify({ status: 'ok' }))
        return
    }

    response.writeHead(200, { 'Content-Type': 'text/plain' })
    response.end('y-websocket server')
})

// Create WebSocket server
const wss = new WebSocket.Server({ server })

wss.on('connection', (conn, req) => {
    // Extract room name from URL path
    // y-websocket client sends path as /<roomName>
    const url = new URL(req.url, `http://${req.headers.host}`)
    let roomName = url.pathname.slice(1) // Remove leading /

    // Get user info from query params (optional)
    const userId = url.searchParams.get('userId') || 'anonymous'

    console.log(`Client connected to room: ${roomName} (user: ${userId})`)

    setupWSConnection(conn, req, {
        docName: roomName,
        gc: true, // Enable garbage collection
    })
})

// Start server
server.listen(PORT, '0.0.0.0', () => {
    console.log(`y-websocket server running on port ${PORT}`)
})

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('Received SIGTERM, shutting down...')
    server.close(() => {
        console.log('Server closed')
        process.exit(0)
    })
})
