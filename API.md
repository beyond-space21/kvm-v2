# KVM LMS API Reference

Base URL: `http://localhost:8080/api/v1`

## Authentication

Protected endpoints require:

```
Authorization: Bearer <jwt_token>
```

### Roles

| Role | Description |
|------|-------------|
| `super_admin` | Full access |
| `staff` | Assigned batches only |
| `student` | Read own data only |

### Error format

```json
{
  "error": "message",
  "code": "NOT_FOUND | FORBIDDEN | UNAUTHORIZED | BAD_REQUEST | CONFLICT | INTERNAL"
}
```

---

## Health

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Service and database health check |

**Response**

```json
{ "status": "ok", "database": "ok" }
```

---

## Auth

| Method | Path | Auth | Role | Description |
|--------|------|------|------|-------------|
| POST | `/api/v1/auth/bootstrap` | No | — | Create first super admin (only when none exists) |
| POST | `/api/v1/auth/login` | No | — | Login and receive JWT |
| GET | `/api/v1/auth/me` | Yes | Any | Current user profile |

### POST `/auth/bootstrap`

```json
{
  "email": "admin",
  "password": "123456",
  "name": "Admin"
}
```

### POST `/auth/login`

```json
{
  "email": "admin",
  "password": "123456"
}
```

**Response**

```json
{
  "token": "<jwt>",
  "user": { "id": "...", "email": "admin", "name": "...", "role": "super_admin", "status": "active" }
}
```

---

## Users

### Staff

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/staff` | admin | Create staff (teacher) |
| GET | `/api/v1/staff/{id}` | admin | Get staff by ID |
| PATCH | `/api/v1/staff/{id}` | admin | Update staff / activate / deactivate |

**Create staff**

```json
{
  "email": "teacher@example.com",
  "password": "123456",
  "name": "Mr. Arun",
  "phone": "+91..."
}
```

**Update staff**

```json
{
  "name": "Updated Name",
  "phone": "+91...",
  "status": "active"
}
```

`status`: `active` | `inactive`

### Students

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/students` | admin | Create student |
| GET | `/api/v1/students` | admin | List students |
| GET | `/api/v1/students/{id}` | admin | Get student by ID |
| PATCH | `/api/v1/students/{id}` | admin | Update student / activate / deactivate |

**Query params (list):** `page`, `limit`

**Create student**

```json
{
  "email": "student@example.com",
  "password": "123456",
  "name": "Rahul",
  "phone": "+91..."
}
```

---

## Academic

### Academic years

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/academic-years` | admin | Create academic year |
| GET | `/api/v1/academic-years` | admin | List academic years |
| GET | `/api/v1/academic-years/{id}` | admin | Get academic year |
| PATCH | `/api/v1/academic-years/{id}` | admin | Update academic year |
| DELETE | `/api/v1/academic-years/{id}` | admin | Delete academic year |

**Create / update body**

```json
{
  "name": "2026-2027",
  "start_date": "2026-04-01",
  "end_date": "2027-03-31",
  "is_active": true
}
```

### Classes

| Method | Path | Role | Description |
|--------|------|------|-------------|
| GET | `/api/v1/classes` | Any | List classes (seeded 8–12) |
| GET | `/api/v1/classes/{id}` | Any | Get class |
| GET | `/api/v1/classes/{id}/offerings` | Any | Offerings for a class |

**Query params (offerings):** `year_id`

### Subjects

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/subjects` | admin | Create subject |
| GET | `/api/v1/subjects` | admin | List subjects |
| GET | `/api/v1/subjects/{id}` | admin | Get subject |
| PATCH | `/api/v1/subjects/{id}` | admin | Update subject |
| DELETE | `/api/v1/subjects/{id}` | admin | Delete subject |

**Create / update body**

```json
{
  "name": "Physics",
  "code": "PHY"
}
```

### Subject offerings

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/offerings` | admin | Create offering (class + subject + fee) |
| GET | `/api/v1/offerings` | admin | List offerings |
| GET | `/api/v1/offerings/{id}` | admin | Get offering |
| PATCH | `/api/v1/offerings/{id}/fee` | admin | Update fee (saves history) |
| DELETE | `/api/v1/offerings/{id}` | admin | Delete offering |
| GET | `/api/v1/offerings/{id}/fee-history` | admin | Fee revision history |

**Query params (list):** `year_id`, `class_id`

**Create offering**

```json
{
  "academic_year_id": "<uuid>",
  "class_id": "<uuid>",
  "subject_id": "<uuid>",
  "fee_amount": 5000,
  "fee_currency": "INR"
}
```

**Update fee**

```json
{
  "fee_amount": 5500,
  "effective_from": "2026-07-01"
}
```

---

## Batches

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/batches` | admin | Create batch |
| GET | `/api/v1/batches` | admin | List batches |
| PATCH | `/api/v1/batches/{id}` | admin | Update batch |
| GET | `/api/v1/batches/mine` | staff, admin | Assigned batches (staff sees own) |
| GET | `/api/v1/batches/{id}/students` | staff*, admin | Enrolled students in batch |

\* Staff only if they teach that batch.

**Query params (list):** `offering_id`, `status`, `page`, `limit`

**Query params (mine):** `teacher_id` (admin only), `page`, `limit`

**Create batch**

```json
{
  "offering_id": "<uuid>",
  "name": "Batch A",
  "teacher_id": "<uuid>",
  "capacity": 30
}
```

**Update batch**

```json
{
  "name": "Batch A",
  "teacher_id": "<uuid>",
  "capacity": 25,
  "status": "active"
}
```

`status`: `active` | `disabled`

---

## Enrollments

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/enrollments` | admin | Enroll student in batch |
| PATCH | `/api/v1/enrollments/{id}/transfer` | admin | Transfer student to another batch |
| DELETE | `/api/v1/enrollments/{id}` | admin | Remove enrollment |
| GET | `/api/v1/enrollments` | admin, student | List enrollments |
| GET | `/api/v1/students/{id}/enrollments/history` | admin | Enrollment history for student |

**Query params (list):** `student_id`, `year_id`, `batch_id`, `status`, `page`, `limit`

Students automatically see only their own active enrollments.

**Create enrollment**

```json
{
  "student_id": "<uuid>",
  "academic_year_id": "<uuid>",
  "offering_id": "<uuid>",
  "batch_id": "<uuid>"
}
```

**Transfer enrollment**

```json
{
  "batch_id": "<new-batch-uuid>"
}
```

---

## Sessions

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/session-templates` | admin | Create recurring session template |
| GET | `/api/v1/session-templates` | admin | List session templates |
| PATCH | `/api/v1/session-templates/{id}` | admin | Update session template |
| POST | `/api/v1/sessions/generate` | admin | Generate session occurrences |
| GET | `/api/v1/sessions/today` | staff, admin | Today's sessions |
| PATCH | `/api/v1/sessions/{id}/cancel` | admin | Cancel a session |

**Query params (templates):** `batch_id`

**Query params (today):** `teacher_id` (admin only)

**Create template**

```json
{
  "batch_id": "<uuid>",
  "teacher_id": "<uuid>",
  "day_of_week": 1,
  "start_time": "17:00",
  "end_time": "18:00"
}
```

`day_of_week`: 0 = Sunday … 6 = Saturday

**Generate sessions**

```json
{
  "start_date": "2026-06-01",
  "end_date": "2026-06-30",
  "batch_id": "<uuid>"
}
```

`batch_id` is optional; omit to generate for all active batches.

---

## Attendance

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/attendance/sessions/{sessionId}` | staff*, admin | Bulk mark attendance |
| GET | `/api/v1/attendance/sessions/{sessionId}` | staff*, admin | Attendance for a session |
| GET | `/api/v1/attendance/students/{id}` | admin, staff, student** | Student attendance + percentage |
| PATCH | `/api/v1/attendance/{id}` | admin | Edit attendance (audit logged) |

\* Staff only for batches they teach.

\** Students can only view their own record.

**Query params (student):** `batch_id`

**Bulk mark**

```json
{
  "records": [
    { "student_id": "<uuid>", "status": "present" },
    { "student_id": "<uuid>", "status": "absent" },
    { "student_id": "<uuid>", "status": "late" }
  ],
  "lock": true
}
```

`status`: `present` | `absent` | `late`

Set `"lock": true` to finalize attendance for the session.

**Admin edit**

```json
{
  "status": "present",
  "reason": "Corrected after review"
}
```

---

## Reports

| Method | Path | Role | Description |
|--------|------|------|-------------|
| GET | `/api/v1/reports/students/{id}/attendance` | admin, staff, student* | Student attendance by batch |
| GET | `/api/v1/reports/batches/{id}` | admin, staff** | Batch attendance report |
| GET | `/api/v1/reports/subjects/{id}` | admin | Subject attendance report |
| GET | `/api/v1/reports/teachers/{id}` | admin, staff*** | Teacher report |
| GET | `/api/v1/reports/daily` | admin, staff**** | Daily attendance report |
| GET | `/api/v1/reports/monthly` | admin, staff**** | Monthly attendance report |
| GET | `/api/v1/reports/enrollments` | admin | Enrollment report |
| GET | `/api/v1/reports/fees` | admin | Fee summary report |

\* Student can only view own ID.

\** Staff only for batches they teach.

\*** Staff can only view own teacher ID.

\**** Staff results are scoped to their assigned batches.

**Query params**

| Endpoint | Params |
|----------|--------|
| `/reports/daily` | `date` (required, `YYYY-MM-DD`) |
| `/reports/monthly` | `month` (required, `YYYY-MM`) |
| `/reports/enrollments` | `year_id` |
| `/reports/fees` | `year_id` |

---

## Pagination

List endpoints support:

| Param | Default | Max |
|-------|---------|-----|
| `page` | 1 | — |
| `limit` | 20 | 100 |

**Response format**

```json
{
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 42
  }
}
```

---

## Test UI

| Path | Description |
|------|-------------|
| `/test_ui/` | Simple HTML page for manual API testing |

---

## Typical admin setup flow

1. `POST /auth/login`
2. `POST /academic-years`
3. `POST /offerings`
4. `POST /staff`
5. `POST /students`
6. `POST /batches`
7. `POST /enrollments`
8. `POST /session-templates`
9. `POST /sessions/generate`

## Typical teacher flow

1. `POST /auth/login`
2. `GET /sessions/today`
3. `GET /batches/{id}/students`
4. `POST /attendance/sessions/{sessionId}`
5. `GET /reports/batches/{id}`
