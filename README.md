# Multi-User Collaborative Document System

A real-time collaborative document editing system inspired by Notion and Feishu, featuring modern UI/UX and comprehensive document management capabilities.



## Cloud Deployment

The application is deployed on cloud infrastructure:

### Production URLs

| Service      | URL                                 |
| ------------ | ----------------------------------- |
| **Frontend** | https://cswuncollabdocs.vercel.app/ |

### Deployment Architecture

- **Frontend**: Deployed on [Vercel](https://vercel.com/) with automatic CI/CD from GitHub
- **Backend API & Y-WebSocket & Redis:** Deployed on cloud server (Railway)
- **Database**: PostgreSQL hosted on Supabase



## Demo

![homepage](./assets/homepage.jpg)

![collab_demo](./assets/collab_demo.jpg)

![comment_demo](./assets/comment_test.jpg)

![sharing_demo](./assets/sharing_demo.jpg)


## Features

### Rich Text Editing
- Full WYSIWYG editor powered by TipTap
- Formatting: Bold, Italic, Underline, Strikethrough
- Headings, Lists (ordered/unordered), Task Lists

### Real-Time Collaboration
- CRDT-based synchronization with Yjs
- Multi-user cursor presence with user colors
- Conflict-free concurrent editing
- Offline support with automatic synchronization

### User Authentication & Management
- User registration and login
- Password change and forgot password flow

### Folder Management
- Hierarchical folder organization
- Create, rename and delete folders
- Folder tree sidebar navigation
- Breadcrumb navigation

### Document Permissions
- Role-based access control (Owner/Edit/View)
- Share documents via email invitation
- Access request system with approval workflow
- Notification bell for pending requests

### Comments & Annotations
- Text selection-based comments
- Threaded comment replies
- Comment resolution
- Synchronized comment sidebar with editor scroll
- Click-to-highlight commented text

### Version Control
- Automatic document snapshots
- Snapshot-based persistence



## Getting Started

### Full Docker Stack

Run all services with Docker:

```bash
docker-compose up --build
```

Services will be available at:

- API: http://localhost:18080
- WebSocket: ws://localhost:1234
- Frontend: Run `cd frontend && npm run dev` separately

### Local Development (Alternatively)

#### 1. Start Infrastructure Services

Start PostgreSQL and Redis using Docker Compose:

```bash
docker-compose up -d postgres redis
```

This will:
- Start PostgreSQL on port `5432` with database `collab_docs`
- Start Redis on port `6379`
- Auto-initialize the database schema from `db/schema.sql`
- Create test users (alice, bob, charlie)

#### 2. Start Backend API Service

```bash
cd backend
go run ./cmd/api
```

API server will be available at: http://localhost:8080

#### 3. Start Y-WebSocket Server

```bash
cd y-websocket-server
npm install
npm start
```

WebSocket server will be available at: ws://localhost:1234

#### 4. Start Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend will be available at: http://localhost:3000



### Test Users

For local development, these test users are pre-seeded (password: `password123`):

| Email | Name | User ID |
|-------|------|---------|
| alice@example.com | Alice | 11111111-1111-1111-1111-111111111111 |
| bob@example.com | Bob | 22222222-2222-2222-2222-222222222222 |
| charlie@example.com | Charlie | 33333333-3333-3333-3333-333333333333 |



## Project Structure

```
.
├── frontend/                    # Next.js application
│   ├── src/
│   │   ├── app/                # App Router pages
│   │   │   ├── doc/[id]/       # Document editor page
│   │   │   ├── login/          # Login page
│   │   │   ├── register/       # Registration page
│   │   │   ├── forgot-password/# Forgot password page
│   │   │   └── reset-password/ # Reset password page
│   │   ├── components/         # React components
│   │   │   ├── AuthGuard.tsx   # Authentication wrapper
│   │   │   ├── CommentsPanel.tsx # Comments sidebar
│   │   │   ├── FolderItem.tsx  # Folder card component
│   │   │   ├── FolderTreeSidebar.tsx # Folder tree navigation
│   │   │   ├── NotificationBell.tsx # Access request notifications
│   │   │   ├── OutlineSidebar.tsx # Document outline
│   │   │   ├── SelectionBubbleMenu.tsx # Text selection toolbar
│   │   │   ├── ShareModal.tsx  # Document sharing modal
│   │   │   ├── Toolbar.tsx     # Editor toolbar
│   │   │   └── UserMenu.tsx    # User dropdown menu
│   │   ├── extensions/         # TipTap custom extensions
│   │   ├── hooks/              # Custom React hooks
│   │   ├── lib/                # Utilities (API client, store)
│   │   └── types/              # TypeScript type definitions
│   └── package.json
│
├── backend/
│   ├── cmd/
│   │   └── api/                # API server entrypoint
│   ├── internal/
│   │   ├── api/                # HTTP handlers & routes
│   │   ├── auth/               # JWT authentication & middleware
│   │   ├── db/                 # Database operations
│   │   ├── logger/             # Logging utilities
│   │   └── models/             # Data models
│   ├── Dockerfile
│   └── go.mod
│
├── y-websocket-server/         # Real-time collaboration server
│   ├── server.js               # Custom y-websocket with persistence
│   ├── Dockerfile
│   └── package.json
│
├── db/
│   ├── schema.sql              # Database schema (development)
│   └── schema.production.sql   # Production schema
│
├── scripts/                    # Utility scripts
├── docker-compose.yml          # Local development stack
├── docker-compose.prod.yml     # Production-like stack
└── .env.example                # Environment variables template
```



## API Reference

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Register new user |
| POST | `/api/auth/login` | Login with email/password |
| POST | `/api/auth/logout` | Logout (protected) |
| GET | `/api/auth/me` | Get current user (protected) |
| PUT | `/api/auth/password` | Change password (protected) |
| POST | `/api/auth/forgot-password` | Request password reset |
| POST | `/api/auth/reset-password` | Reset password with token |

### Documents

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/docs` | List accessible documents |
| POST | `/api/docs` | Create new document |
| GET | `/api/docs/:id` | Get document (requires view) |
| PUT | `/api/docs/:id` | Update document (requires edit) |
| DELETE | `/api/docs/:id` | Delete document (requires owner) |
| PUT | `/api/docs/:id/move` | Move document to folder |

### Permissions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/docs/:id/permissions` | List permissions (owner) |
| PUT | `/api/docs/:id/permissions` | Set permission (owner) |
| DELETE | `/api/docs/:id/permissions/:userId` | Remove permission (owner) |
| GET | `/api/docs/:id/my-permission` | Get own permission |

### Access Requests

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/docs/:id/access-request` | Request document access |
| GET | `/api/docs/:id/access-requests` | List requests (owner) |
| GET | `/api/access-requests/pending` | List pending requests for owner |
| PUT | `/api/access-requests/:id` | Approve/reject request |

### Comments

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/docs/:id/comments` | List comments (requires view) |
| POST | `/api/docs/:id/comments` | Create comment (requires comment+) |
| PUT | `/api/comments/:id` | Update own comment |
| DELETE | `/api/comments/:id` | Delete own comment |

### Snapshots

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/docs/:id/snapshots` | List snapshots (requires view) |

### Folders

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/folders` | Create folder |
| GET | `/api/folders` | Get folder contents |
| GET | `/api/folders/tree` | Get complete folder tree |
| GET | `/api/folders/:id` | Get folder by ID |
| GET | `/api/folders/:id/path` | Get folder path (breadcrumbs) |
| PUT | `/api/folders/:id` | Update folder |
| DELETE | `/api/folders/:id` | Delete folder |
| PUT | `/api/folders/:id/move` | Move folder |

### Yjs Persistence (Internal)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/yjs/:docId/snapshot` | Get Yjs snapshot |
| POST | `/api/yjs/:docId/snapshot` | Save Yjs snapshot |



## Environment Variables

### Frontend (.env.local)

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:1234
```

### Backend (.env)

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/collab_docs?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key-change-in-production
PORT=8080
ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```

### Y-WebSocket Server

```env
PORT=1234
API_URL=http://localhost:8080
```



## Tech Stack

### Frontend

- **Framework**: Next.js 14 (App Router)
- **UI**: React 18 + TypeScript
- **Editor**: TipTap 2.x
- **Real-time**: Yjs + y-websocket
- **State**: Zustand
- **Styling**: TailwindCSS
- **Icons**: Lucide React

### Backend

- **API Server**: Go + Gin Framework
- **Authentication**: JWT (golang-jwt)
- **Database**: PostgreSQL 15
- **Real-time**: y-websocket server (Node.js)

### Infrastructure

- **Containerization**: Docker & Docker Compose
- **Database Driver**: pgx/v5



## Database Schema

### Core Tables

- **users**: User accounts (id, email, password_hash, name, avatar_url)
- **folders**: Hierarchical folder structure (id, name, owner_id, parent_id)
- **documents**: Document metadata (id, title, owner_id, folder_id)
- **document_permissions**: Access control (doc_id, user_id, role)
- **doc_snapshots**: Yjs document state (doc_id, version, snapshot)
- **comments**: Document comments with selection (id, doc_id, user_id, content, selection)
- **access_requests**: Permission request workflow (id, doc_id, requester_id, status, requested_role)

### Permission Roles

| Role | Capabilities |
|------|-------------|
| `owner` | Full access, manage permissions, delete document |
| `edit` | Read/write document content |
| `comment` | Read document, add/edit own comments |
| `view` | Read-only access |



## Acknowledgments

- [TipTap](https://tiptap.dev/) - Headless rich-text editor
- [Yjs](https://yjs.dev/) - CRDT framework
- [y-websocket](https://github.com/yjs/y-websocket) - WebSocket provider for Yjs
- [Gin](https://gin-gonic.com/) - Go web framework
- [Next.js](https://nextjs.org/) - React framework
- [TailwindCSS](https://tailwindcss.com/) - Utility-first CSS
