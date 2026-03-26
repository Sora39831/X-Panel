# User Registration Feature Design

## Overview

Add a registration feature to X-Panel that allows users to self-register via the login page. Registration automatically creates proxy clients across all inbounds, simulating the manual "add client" workflow. Registered users can log in to view their own traffic and expiry information.

## Core Concept

- **Registration = auto-creating proxy clients**: When a user registers, the system generates a UUID-based client ID, creates a `Client` object in every inbound's `Settings.clients` array, and creates corresponding `ClientTraffic` records.
- **Single identity**: The user's email serves as both the panel login username and the proxy client email identifier across all inbounds.
- **Role-based access**: `User` model gains a `Role` field (`"admin"` / `"user"`). Admins retain full panel access; regular users see only their own traffic info.

## Data Model Changes

### User Model (`database/model/model.go`)

Add one field:

```go
type User struct {
    Id       int    `json:"id" gorm:"primaryKey;autoIncrement" bson:"_id,omitempty"`
    Username string `json:"username" bson:"username"`
    Password string `json:"password" bson:"password"`
    Role     string `json:"role" bson:"role"` // "admin" or "user"
}
```

- Default admin user: `Role = "admin"`
- Registered users: `Role = "user"`
- Migration: existing users without Role get `"admin"` automatically

### No new tables

All data fits in existing tables: `users`, `inbounds` (Settings JSON), `client_traffics`.

## Registration Flow

1. User visits login page, clicks "Register" tab
2. Enters email (serves as username) and password (panel login only)
3. Frontend validates input, sends POST to `/register`
4. Backend:
   - Validate email format, length (4-64 chars), allowed characters: letters, digits, `_`, `@`, `.`
   - Validate password: min 6 chars, must contain letter + digit
   - Check `User.Username` uniqueness
   - Check `ClientTraffic.Email` uniqueness (no existing client with same email)
   - Generate UUID via xray's UUID generation for client ID
   - For each existing inbound:
     - Parse `Inbound.Settings` JSON
     - Append new `Client` object to `clients` array (`email=用户名, TotalGB=0, ExpiryTime=0, Enable=true`)
     - Save updated inbound
     - Create `ClientTraffic` record (`Email=用户名, Up=0, Down=0, Total=0, ExpiryTime=0, Enable=true`)
   - Create `User` record with `Role="user"`, bcrypt-hashed password
5. Return success, frontend redirects to login page

## Security

### Registration endpoint

| Threat | Mitigation |
|--------|-----------|
| Brute-force registration | IP rate limit: max 5 requests/min per IP |
| SQL/NoSQL injection | Parameterized queries, input sanitization |
| Weak passwords | Min 6 chars, must contain letter + digit |
| Duplicate registration | Check both `User.Username` and `ClientTraffic.Email` |
| XSS / path traversal | Restrict email to `[a-zA-Z0-9_@.]`, length 4-64 |
| CAPTCHA bypass | Simple math CAPTCHA generated server-side, validated on submit |

### Login endpoint (existing, hardened)

| Threat | Mitigation |
|--------|-----------|
| Brute-force login | IP lockout: 5 consecutive failures → 15 min block |
| User enumeration | Unified error message: "username or password incorrect" |

### Session / access control

- Session stores `Role` alongside user object
- `checkLogin` middleware enforces role-based routing:
  - Admin → full panel access
  - User → only `/panel/userinfo` and `/panel/api/user/info`
- All admin APIs reject requests from `role="user"`

## Frontend Changes

### login.html

- Add "Register" tab alongside existing login form
- Register form fields: email, password, confirm password, CAPTCHA
- Form validation: email format, password match, password strength
- On success: show message, auto-switch to login tab

### userinfo.html (new page)

- Simple card-based layout, no sidebar navigation
- Displays:
  - Email (username)
  - For each inbound containing this client:
    - Inbound remark + protocol
    - Upload / Download (human-readable bytes)
    - Total traffic limit (0 = unlimited)
    - Expiry time (0 = never)
    - Enable status

## Backend Changes

### database/model/model.go

- Add `Role` field to `User` struct

### database/db.go

- Migration: set `Role="admin"` for existing users where Role is empty

### web/service/user.go

- `Register(email, password)` — full registration logic
- `GetClientTrafficByEmail(email)` — returns all ClientTraffic records matching email
- `CheckEmailExists(email)` — checks User + ClientTraffic tables

### web/controller/index.go

- New route: `POST /register`
- Handler: validate input → rate limit check → CAPTCHA verify → call Register service

### web/controller/xui.go (or new controller)

- New route: `GET /panel/userinfo` — render userinfo.html (with role check)
- New route: `GET /panel/api/user/info` — return current user's ClientTraffic data

### web/controller/base.go

- `checkLogin` middleware: add role-based path filtering for `role="user"`

## Files to Create/Modify

| File | Action |
|------|--------|
| `database/model/model.go` | Modify: add Role to User |
| `database/db.go` | Modify: migration for existing users |
| `web/service/user.go` | Modify: add Register, GetClientTrafficByEmail |
| `web/controller/index.go` | Modify: add /register route |
| `web/controller/base.go` | Modify: role-based access control |
| `web/controller/xui.go` | Modify: add userinfo route |
| `web/html/login.html` | Modify: add register tab |
| `web/html/userinfo.html` | Create: user info page |
