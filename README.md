# Multi-User Collaborative Document System

A real-time collaborative document editing system inspired by Notion and Feishu, featuring:

- 📝 Rich text editing with TipTap
- 🔄 Real-time collaboration with CRDT (Yjs)
- 👥 Multi-user cursor presence
- 🔐 Document permissions (owner/edit/comment/view)
- 💬 Block-level comments
- 📸 Version snapshots & rollback
- ☁️ Cloud sync with offline support

## Tech Stack

- **Frontend**: Next.js 14 + React 18 + TipTap + Yjs
- **Backend**: Go (REST API + WebSocket)
- **Database**: PostgreSQL (Supabase compatible)
- **Cache/PubSub**: Redis
- **Deployment**: Vercel (frontend) + Railway (backend)

## Local Development

### Prerequisites

- Docker & Docker Compose
- Node.js 18+
- Go 1.21+

### Quick Start

1. **Start infrastructure services:**

```bash
docker-compose up -d postgres redis
```

2. **Start backend services:**

```bash
# Terminal 1: API Service
cd backend
go run ./cmd/api

# Terminal 2: Collaboration Service
cd backend
go run ./cmd/collab
```

3. **Start frontend:**

```bash
cd frontend
npm install
npm run dev
```

4. **Open in browser:**

- Frontend: http://localhost:3000
- API Service: http://localhost:8080
- WebSocket: ws://localhost:8081

### Full Docker Stack

Run everything in Docker:

```bash
docker-compose up --build
```

### Test Users

For local development, the following test users are available:

| Email | Name | User ID |
|-------|------|---------|
| alice@example.com | Alice | 11111111-1111-1111-1111-111111111111 |
| bob@example.com | Bob | 22222222-2222-2222-2222-222222222222 |
| charlie@example.com | Charlie | 33333333-3333-3333-3333-333333333333 |

## Project Structure

```
.
├── frontend/                 # Next.js application
│   ├── src/
│   │   ├── app/             # App Router pages
│   │   ├── components/      # React components
│   │   ├── lib/             # Utilities
│   │   └── hooks/           # Custom hooks
│   └── package.json
│
├── backend/
│   ├── cmd/
│   │   ├── api/             # REST API entrypoint
│   │   └── collab/          # WebSocket service
│   ├── internal/
│   │   ├── api/             # API handlers
│   │   ├── collab/          # Collaboration logic
│   │   ├── auth/            # Authentication
│   │   ├── db/              # Database operations
│   │   └── redis/           # Redis pub/sub
│   └── go.mod
│
├── db/
│   └── schema.sql           # Database schema
│
└── docker-compose.yml       # Local dev stack
```

## API Endpoints

### Documents

- `GET /api/docs` - List documents
- `POST /api/docs` - Create document
- `GET /api/docs/:id` - Get document
- `PUT /api/docs/:id` - Update document
- `DELETE /api/docs/:id` - Delete document

### Permissions

- `GET /api/docs/:id/permissions` - Get permissions
- `PUT /api/docs/:id/permissions` - Update permissions

### Comments

- `GET /api/docs/:id/comments` - Get comments
- `POST /api/docs/:id/comments` - Create comment
- `PUT /api/comments/:id` - Update comment
- `DELETE /api/comments/:id` - Delete comment

### WebSocket

- `WS /ws/collab/:docId` - Real-time collaboration

## Environment Variables

### Frontend (.env.local)

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8081
```

### Backend

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/collab_docs?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key
PORT=8080
```

## License

MIT
