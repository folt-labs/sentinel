# ServerGuard Testing Guide

## Prerequisites

- Docker Desktop installed and running
- Go 1.22+ installed
- Node.js 20+ installed
- PowerShell or Command Prompt

## Step 1: Start Infrastructure

Open PowerShell in the `securityguard` directory:

```powershell
cd C:\projects\foltlabs-new\securityguard
docker compose -f deploy/docker-compose.dev.yml up -d
```

**Expected output:**
```
[+] Running 3/3
 ✔ Container serverguard-redis    Started
 ✔ Container serverguard-postgres Started
 ✔ Container serverguard-mailhog  Started
```

**Verify containers are running:**
```powershell
docker ps
```

You should see 3 containers: postgres, redis, mailhog.

## Step 2: Run Database Migrations

Connect to PostgreSQL and run the migration:

```powershell
docker exec -i serverguard-postgres psql -U serverguard -d serverguard < api/internal/database/migrations/001_initial.sql
```

**Expected output:**
```
CREATE EXTENSION
CREATE TABLE
CREATE INDEX
... (multiple CREATE statements)
```

**Verify tables exist:**
```powershell
docker exec -it serverguard-postgres psql -U serverguard -d serverguard -c "\dt"
```

**Expected output:**
```
               List of relations
 Schema |         Name          | Type  |   Owner
--------+-----------------------+-------+------------
 public | alert_rules           | table | serverguard
 public | alerts                | table | serverguard
 public | notification_channels | table | serverguard
 public | organizations         | table | serverguard
 public | security_events       | table | serverguard
 public | servers               | table | serverguard
 public | users                 | table | serverguard
```

## Step 3: Start the API

Open a NEW PowerShell window:

```powershell
cd C:\projects\foltlabs-new\securityguard\api
go run cmd/serverguard-api/main.go
```

**Expected output:**
```
2024/XX/XX 12:00:00 ServerGuard API dev listening on :8080
```

**Test the health endpoint (in another terminal):**
```powershell
curl http://localhost:8080/health
```

**Expected output:**
```json
{"status":"ok","time":"now"}
```

## Step 4: Install Dashboard Dependencies

Open a NEW PowerShell window:

```powershell
cd C:\projects\foltlabs-new\securityguard\dashboard
npm install
```

**Expected output:**
```
added XXX packages in Xs
```

## Step 5: Start the Dashboard

In the same terminal:

```powershell
npm run dev
```

**Expected output:**
```
  ▲ Next.js 15.x.x
  - Local:        http://localhost:3001
  - Environments: .env.local

 ✓ Starting...
 ✓ Ready in Xs
```

## Step 6: Test the Application

### 6.1 Open the Dashboard

Open your browser to: **http://localhost:3001**

**Expected:** You should be redirected to the login page.

### 6.2 Register a New Account

1. Click "Sign up" link
2. Fill in the form:
   - Name: `Test User`
   - Email: `test@example.com`
   - Password: `password123`
   - Organization: `Test Org`
3. Click "Create account"

**Expected:** You should be redirected to the dashboard.

### 6.3 View the Dashboard

**Expected:** Dashboard showing:
- Total Servers: 0
- Online Servers: 0
- Open Alerts: 0
- Critical Alerts: 0
- Events Today: 0

### 6.4 Add a Server

1. Click "Servers" in the sidebar
2. Click "Add Server" button
3. Fill in:
   - Hostname: `test-server`
   - IP Address: `192.168.1.100` (optional)
4. Click "Create"

**Expected:**
- Modal shows an API key starting with `sg_...`
- Copy this key - you'll need it for the agent!

### 6.5 View Server Details

1. Click on the server in the list

**Expected:** Server detail page showing:
- Status: offline (agent not connected yet)
- No events yet

## Step 7: Test API Directly

### 7.1 Register via API

```powershell
curl -X POST http://localhost:8080/api/v1/auth/register `
  -H "Content-Type: application/json" `
  -d '{"email":"api@test.com","password":"password123","name":"API User","organization_name":"API Org"}'
```

**Expected output:**
```json
{
  "token": "eyJhbG...",
  "user": {"id":"...","email":"api@test.com",...},
  "organization": {"id":"...","name":"API Org",...}
}
```

### 7.2 Login via API

```powershell
curl -X POST http://localhost:8080/api/v1/auth/login `
  -H "Content-Type: application/json" `
  -d '{"email":"test@example.com","password":"password123"}'
```

**Expected:** Returns token and user info.

### 7.3 Create Server via API

Save the token from login, then:

```powershell
$token = "YOUR_JWT_TOKEN_HERE"
curl -X POST http://localhost:8080/api/v1/servers `
  -H "Content-Type: application/json" `
  -H "Authorization: Bearer $token" `
  -d '{"hostname":"api-server","ip_address":"10.0.0.1"}'
```

**Expected:**
```json
{
  "server": {"id":"...","hostname":"api-server",...},
  "api_key": "sg_..."
}
```

### 7.4 Simulate Agent Sending Events

Use the API key from server creation:

```powershell
$apiKey = "sg_YOUR_API_KEY_HERE"
curl -X POST http://localhost:8080/api/v1/agent/events `
  -H "Content-Type: application/json" `
  -H "X-API-Key: $apiKey" `
  -d '{"events":[{"type":"ssh_login_failed","severity":"warning","timestamp":"2024-01-15T12:00:00Z","data":{"username":"root","source_ip":"192.168.1.50"}}]}'
```

**Expected:**
```json
{"status":"ok","received":1}
```

### 7.5 Check Dashboard

Refresh the dashboard in your browser.

**Expected:**
- Events Today: 1
- Open Alerts: 1 (created from the warning event)
- Server status may show "warning"

## Step 8: View Alerts

1. Click "Alerts" in the sidebar

**Expected:**
- One alert: "Failed SSH Login Attempt"
- Severity: warning
- Status: open

### 8.1 Acknowledge Alert

1. Click "Acknowledge" button on the alert

**Expected:** Alert status changes to "acknowledged"

### 8.2 Resolve Alert

1. Click "Resolve" button

**Expected:** Alert status changes to "resolved"

## Step 9: Check Mailhog (Email Testing)

Open browser to: **http://localhost:8025**

**Expected:** Mailhog web UI (emails would appear here when notification channels are configured)

## Step 10: Cleanup

When done testing:

```powershell
# Stop the API (Ctrl+C in its terminal)
# Stop the Dashboard (Ctrl+C in its terminal)

# Stop and remove containers
docker compose -f deploy/docker-compose.dev.yml down

# To also remove data volumes:
docker compose -f deploy/docker-compose.dev.yml down -v
```

## Troubleshooting

### API won't start - database connection error
```
Failed to connect to database
```
**Solution:** Make sure Docker containers are running: `docker ps`

### Dashboard shows "Failed to fetch"
**Solution:** Make sure API is running on port 8080

### Can't access localhost:3001
**Solution:** Make sure no other app is using port 3001, or change port in package.json

### Migrations fail
```
relation already exists
```
**Solution:** Reset the database:
```powershell
docker exec -it serverguard-postgres psql -U serverguard -c "DROP DATABASE serverguard; CREATE DATABASE serverguard;"
```
Then run migrations again.
