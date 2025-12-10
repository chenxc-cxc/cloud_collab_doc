#!/bin/bash

# Local testing script for the Collaborative Document System

set -e

echo "🚀 Starting Collaborative Document System (Local Testing)"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Start infrastructure
echo "📦 Starting PostgreSQL and Redis..."
docker-compose up -d postgres redis

# Wait for services to be healthy
echo "⏳ Waiting for services to be ready..."
sleep 5

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first."
    exit 1
fi

# Check if Node is installed
if ! command -v npm &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18+ first."
    exit 1
fi

echo ""
echo "✅ Infrastructure ready!"
echo ""
echo "📋 Next steps to run the application:"
echo ""
echo "1. Start the API service (Terminal 1):"
echo "   cd backend && go run ./cmd/api"
echo ""
echo "2. Start the Collaboration service (Terminal 2):"
echo "   cd backend && go run ./cmd/collab"
echo ""
echo "3. Start the Frontend (Terminal 3):"
echo "   cd frontend && npm install && npm run dev"
echo ""
echo "4. Open in browser:"
echo "   http://localhost:3000"
echo ""
echo "📊 Service URLs:"
echo "   - Frontend:    http://localhost:3000"
echo "   - API:         http://localhost:8080"
echo "   - WebSocket:   ws://localhost:8081"
echo "   - PostgreSQL:  localhost:5432"
echo "   - Redis:       localhost:6379"
echo ""
echo "👥 Test Users:"
echo "   - Alice (alice@example.com) - Owner"
echo "   - Bob (bob@example.com) - Editor"
echo "   - Charlie (charlie@example.com) - Viewer"
echo ""
