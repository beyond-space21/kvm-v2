# KVM LMS Backend

Learning Management System REST API for coaching institutes (grades 8–12).

## Stack

- Go + Chi router
- PostgreSQL
- JWT authentication with role-based access control

## Roles

| Role | Access |
|------|--------|
| `super_admin` | Full system management |
| `staff` | Assigned batches, mark attendance, own reports |
| `student` | Read-only: enrollments, attendance, own reports |

## Quick Start

Requires a local PostgreSQL instance.

```bash
# Create database (once)
createdb lms

cp .env.example .env
# Edit .env with your DATABASE_URL and JWT_SECRET

go run .
```

On first run with `RUN_MIGRATIONS=true`, only the schema is applied. No admin user is created automatically.

Create the first admin explicitly with `POST /api/v1/auth/bootstrap`.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `JWT_SECRET` | — | HS256 signing secret |
| `RUN_MIGRATIONS` | `false` | Run migrations on startup |

## API Overview

Base path: `/api/v1`

### Auth
- `POST /auth/bootstrap` — Create first super admin (only when none exists)
- `POST /auth/login` — Login
- `GET /auth/me` — Current user (auth required)

### Users (admin)
- `POST/GET/PATCH/DELETE /staff/{id}`, `GET /staff`
- `POST/GET/PATCH/DELETE /students`, `GET /students`

### Academic (admin unless noted)
- `POST/GET/PATCH/DELETE /academic-years/{id}`
- `POST/GET/PATCH/DELETE /classes/{id}`, `GET /classes/{id}/offerings`
- `POST/GET/PATCH/DELETE /subjects/{id}`
- `POST/GET/PATCH/DELETE /offerings/{id}`, `PATCH /offerings/{id}/fee`

### Batches
- `POST/GET/PATCH/DELETE /batches` (admin)
- `GET /batches/mine` (staff)
- `GET /batches/{id}/students` (admin, staff own batch)

### Enrollments
- `POST/GET/DELETE /enrollments`, `GET /enrollments/{id}`, `PATCH /enrollments/{id}/transfer` (admin)
- `GET /enrollments` (admin all, student own)
- `GET /students/{id}/enrollments/history` (admin)

### Sessions
- `POST/GET/PATCH/DELETE /session-templates` (admin)
- `POST /sessions/generate` (admin)
- `GET /sessions/today`, `GET /sessions/{id}` (staff)
- `PATCH /sessions/{id}/cancel` (admin)

### Attendance
- `POST /attendance/sessions/{sessionId}` — Bulk mark + lock (admin, staff)
- `GET /attendance/sessions/{sessionId}` (admin, staff)
- `GET /attendance/students/{id}` (admin, staff, student self)
- `PATCH /attendance/{id}` — Admin edit with audit log

### Reports
- `GET /reports/students/{id}/attendance`
- `GET /reports/batches/{id}`
- `GET /reports/subjects/{id}`
- `GET /reports/teachers/{id}`
- `GET /reports/daily?date=YYYY-MM-DD`
- `GET /reports/monthly?month=YYYY-MM`
- `GET /reports/enrollments`
- `GET /reports/fees`

### Health
- `GET /health`

## Make Targets

```bash
make run          # Run server with migrations
make test         # Run tests
make migrate-up   # Run migrations manually
```

## Academic Structure

```
Academic Year → Enrollment ← Student
Class → Subject Offering (+ fee) → Batch → Session Template → Session → Attendance
```

Seeded data: classes 8–12, subjects (Science, Math, English, Physics, Chemistry, Biology).
