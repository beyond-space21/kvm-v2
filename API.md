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
| GET | `/api/v1/staff` | admin | List staff |
| GET | `/api/v1/staff/{id}` | admin | Get staff by ID |
| PATCH | `/api/v1/staff/{id}` | admin | Update staff / activate / deactivate |
| DELETE | `/api/v1/staff/{id}` | admin | Delete staff |

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

**Query params (list):** `page`, `limit`, `status`, `search` (name or email)

### Students

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/students` | admin | Create student |
| GET | `/api/v1/students` | admin | List students |
| GET | `/api/v1/students/{id}` | admin | Get student by ID |
| PATCH | `/api/v1/students/{id}` | admin | Update student / activate / deactivate |
| DELETE | `/api/v1/students/{id}` | admin | Delete student |

**Query params (list):** `page`, `limit`, `status`, `class_id`, `search` (name or email)

**Create student**

```json
{
  "email": "student@example.com",
  "password": "123456",
  "name": "Rahul",
  "phone": "+91...",
  "class_id": "<uuid>"
}
```

`class_id` is required. Students can only enroll in offerings for their assigned class.

**Update student**

```json
{
  "name": "Updated Name",
  "phone": "+91...",
  "status": "active",
  "class_id": "<uuid>"
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

**Query params (list):** `page`, `limit`, `is_active` (`true` | `false`)

### Classes

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/classes` | admin | Create class |
| GET | `/api/v1/classes` | Any | List classes |
| GET | `/api/v1/classes/{id}` | Any | Get class |
| PATCH | `/api/v1/classes/{id}` | admin | Update class |
| DELETE | `/api/v1/classes/{id}` | admin | Delete class |
| GET | `/api/v1/classes/{id}/offerings` | Any | Offerings for a class |

**Create / update body**

```json
{
  "name": "Class 8",
  "grade": 8
}
```

**Query params (list):** `page`, `limit`, `grade`

**Query params (class offerings):** `page`, `limit`, `subject_id`

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

**Query params (list):** `page`, `limit`, `search` (matches name or code)

### Subject offerings

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/offerings` | admin | Create offering (class + subject + fee) |
| GET | `/api/v1/offerings` | admin | List offerings |
| GET | `/api/v1/offerings/{id}` | admin | Get offering |
| PATCH | `/api/v1/offerings/{id}` | admin | Update offering class/subject |
| PATCH | `/api/v1/offerings/{id}/fee` | admin | Update fee (saves history) |
| DELETE | `/api/v1/offerings/{id}` | admin | Delete offering |
| GET | `/api/v1/offerings/{id}/fee-history` | admin | Fee revision history |

**Query params (list):** `class_id`, `subject_id`, `page`, `limit`

**Create offering**

```json
{
  "class_id": "<uuid>",
  "subject_id": "<uuid>",
  "fee_amount": 5000,
  "fee_currency": "INR"
}
```

**Update offering**

```json
{
  "class_id": "<uuid>",
  "subject_id": "<uuid>"
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
| GET | `/api/v1/batches/{id}` | admin | Get batch |
| PATCH | `/api/v1/batches/{id}` | admin | Update batch |
| DELETE | `/api/v1/batches/{id}` | admin | Delete batch |
| GET | `/api/v1/batches/mine` | staff, admin | Assigned batches (staff sees own) |
| GET | `/api/v1/batches/{id}/students` | staff*, admin | Enrolled students in batch |

\* Staff only if they teach that batch.

**Query params (list):** `offering_id`, `teacher_id`, `status`, `page`, `limit`

**Query params (mine):** `teacher_id` (admin only), `page`, `limit`

**Query params (batch students):** `page`, `limit`

**Create batch**

```json
{
  "offering_id": "<uuid>",
  "name": "Batch A",
  "teacher_id": "<uuid>"
}
```

**Update batch**

```json
{
  "name": "Batch A",
  "teacher_id": "<uuid>",
  "status": "active"
}
```

`status`: `active` | `disabled`

---

## Enrollments

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/enrollments` | admin | Enroll student in batch |
| GET | `/api/v1/enrollments/{id}` | admin | Get enrollment |
| PATCH | `/api/v1/enrollments/{id}/transfer` | admin | Transfer student to another batch |
| DELETE | `/api/v1/enrollments/{id}` | admin | Remove enrollment |
| GET | `/api/v1/enrollments` | admin, student | List enrollments |
| GET | `/api/v1/students/{id}/enrollments/history` | admin | Enrollment history for student |

**Query params (list):** `student_id`, `year_id`, `offering_id`, `batch_id`, `status`, `page`, `limit`

**Query params (history):** `year_id`, `offering_id`, `status`, `page`, `limit`

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

**Transfer enrollment** — new batch must belong to the same offering.

```json
{
  "batch_id": "<new-batch-uuid>"
}
```

---

## Payments

| Method | Path | Role | Description |
|--------|------|------|-------------|
| GET | `/api/v1/enrollments/{id}/invoice` | admin | Fee invoice for enrollment |
| GET | `/api/v1/enrollments/{id}/payments` | admin | Payment history for enrollment |
| POST | `/api/v1/enrollments/{id}/payments` | admin | Record a payment |

An invoice is created automatically when an enrollment is created (amount snapshotted from the offering fee).

**Record payment**

```json
{
  "amount": 5000,
  "method": "cash",
  "reference": "receipt-001"
}
```

`method` and `reference` are optional.

**Invoice response**

```json
{
  "id": "<uuid>",
  "enrollment_id": "<uuid>",
  "amount": 5000,
  "currency": "INR",
  "status": "pending",
  "paid_amount": 0,
  "created_at": "...",
  "updated_at": "..."
}
```

`status`: `pending` | `partial` | `paid` | `waived`

**Query params (payments list):** `page`, `limit`

---

## Sessions

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/api/v1/session-templates` | admin | Create recurring session template |
| GET | `/api/v1/session-templates` | admin | List session templates |
| GET | `/api/v1/session-templates/{id}` | admin | Get session template |
| PATCH | `/api/v1/session-templates/{id}` | admin | Update session template |
| DELETE | `/api/v1/session-templates/{id}` | admin | Delete session template |
| POST | `/api/v1/sessions/generate` | admin | Generate session occurrences |
| GET | `/api/v1/sessions/today` | staff, admin | Today's sessions |
| GET | `/api/v1/sessions/{id}` | staff, admin | Get session |
| PATCH | `/api/v1/sessions/{id}/cancel` | admin | Cancel a session |

**Query params (templates):** `batch_id`, `teacher_id`, `page`, `limit`

**Query params (today):** `teacher_id` (admin only), `page`, `limit`

**Create template**

```json
{
  "batch_id": "<uuid>",
  "day_of_week": 1,
  "start_time": "17:00",
  "end_time": "18:00"
}
```

Teacher is taken from the batch's assigned teacher. The batch must have a teacher before creating a template.

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

**Query params (session attendance):** `status`, `page`, `limit`

**Query params (student attendance):** `batch_id`, `status`, `page`, `limit`

Student attendance response wraps records in the standard paginated `data` object plus `percentage`, `present`, and `total` fields.

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
| GET | `/api/v1/reports/fees` | admin | Fee summary (total, paid, due per student) |

\* Student can only view own ID.

\** Staff only for batches they teach.

\*** Staff can only view own teacher ID.

\**** Staff results are scoped to their assigned batches.

**Query params**

| Endpoint | Params |
|----------|--------|
| `/reports/daily` | `date` (required, `YYYY-MM-DD`) |
| `/reports/monthly` | `month` (required, `YYYY-MM`) |
| `/reports/enrollments` | `year_id`, `class_id`, `batch_id`, `page`, `limit` |
| `/reports/fees` | `year_id`, `class_id`, `page`, `limit` |

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

## Typical admin setup flow

1. `POST /auth/login` (or `/auth/bootstrap` for first admin)
2. `POST /classes`
3. `POST /subjects`
4. `POST /academic-years`
5. `POST /offerings`
6. `POST /staff`
7. `POST /students` (include `class_id`)
8. `POST /batches`
9. `POST /enrollments`
10. `POST /enrollments/{id}/payments` (optional)
11. `POST /session-templates`
12. `POST /sessions/generate`

## Typical teacher flow

1. `POST /auth/login`
2. `GET /sessions/today`
3. `GET /batches/{id}/students`
4. `POST /attendance/sessions/{sessionId}`
5. `GET /reports/batches/{id}`
