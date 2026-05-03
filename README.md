# uwatu-core · Uwatu Unified Livestock Protection Platform (API Gateway)

## Project Status

- [x] **Core framework** – Fiber v2 server, MQTT client, Viper config, JWT auth middleware
- [x] **Nokia API Client** – RapidAPI gateway with shared transport & rate limiting
- [x] **All 9 Nokia APIs** – Location, SIM Swap, Reachability, Roaming, Device Swap, QoD, Slicing, Congestion (stub), Number Verification (client)
- [x] **MQTT Ingestion** – JSON payload parsing with type‑safe conversion
- [x] **Enricher Engine** – thread‑safe caching, parallel staggered Nokia calls, SignalMatrix assembly
- [x] **Models** – TagTelemetry, SignalMatrix, ScoredEvent, AlertPayload, NokiaSignals
- [x] **Environment Config** – Viper‑based, JWT secret, RapidAPI key, DB credentials
- [x] **Auth Middleware** – JWT generation/validation, RBAC middleware
- [x] **Notification Dispatchers** – Africa's Talking SMS, WhatsApp, USSD, Firebase FCM
- [x] **Decision Engine** – POST /score to uwatu‑intelligence with 500ms timeout + cached fallback
- [x] **Database Layer** – TimescaleDB/PostgreSQL connection pool, hypertable, persistence
- [x] **WebSocket Hub** – live telemetry broadcast to dashboard
- [ ] **Alert Router** – tier‑based channel selection, notification_log writes (Mphele)
- [ ] **Farm Registry, Animal CRUD, Geofence Manager** – core domain logic (Mphele)
- [ ] **Intelligence Service** – Python/FastAPI classification endpoint (Mphele)
- [ ] **Coverage Map** – network coverage visualization (Elvis, stretch)
- [ ] **LITS‑Compliant Digital Evidence Export** – forensic report generation (Elvis, stretch)

---

## Project Overview

`uwatu-core` is the central nervous system of the Uwatu platform. It receives real-time telemetry from livestock ear tags (or our Go simulator), enriches that data with nine Nokia Network as Code CAMARA APIs, persists everything into a TimescaleDB hypertable, and passes a fully assembled **SignalMatrix** to the Python intelligence service for behavioural classification. Alerts are then routed through Africa’s Talking and Firebase to reach smallholder farmers on any phone.

The core is built in Go (Fiber v2) with an emphasis on resilience, low latency, and minimal external API consumption – critical for rural Sub‑Saharan Africa where both connectivity and budget are scarce.

---

## Architecture

```text
Simulator / Real Tags
       │  MQTT (JSON)
       ▼
┌─────────────────────┐
│  Ingestion Handler  │ handler.go
│ • subscribes to      │
│   uwatu/farm/+/tag/+ │
│ • parses nested JSON │
│ • builds TagTelemetry│
└────────┬────────────┘
         │ device_id, msisdn, telemetry
         ▼
┌─────────────────────┐
│     Enricher         │ enricher.go
│ • 2‑minute cache    │
│ • 7 parallel Nokia  │
│   API calls (staggered)
│ • assembles SignalMatrix
│ • persists to DB    │
│ • calls Intelligence │
└────────┬────────────┘
         │ SignalMatrix
         ▼
┌─────────────────────┐
│   Decision Engine    │ decision/engine.go
│ • POST /score to     │
│   uwatu-intelligence │
│ • 500ms timeout      │
│ • safe fallback      │
└────────┬────────────┘
         │ ScoredEvent
         ▼
┌─────────────────────┐
│   Alert Router       │ alerts/router.go (Mphele)
│ • tier‑based channel│
│   selection         │
│ • SMS/WhatsApp/USSD │
│   /Push via AT & FCM│
└─────────────────────┘
         │
         ▼
┌─────────────────────┐
│   WebSocket Hub      │ ws/hub.go
│ • broadcasts live    │
│   telemetry to dashboard
└─────────────────────┘
```

---

## Implemented Features

### MQTT Ingestion (`internal/ingestion/handler.go`)
- Connects to HiveMQ public broker (`tcp://broker.hivemq.com:1883`).
- Uses a unique, timestamped Client ID to avoid conflicts.
- Unmarshals nested JSON (the simulator’s `firmware_payload` object).
- Handles Go’s `float64` JSON numbers by explicit casting to `int` where needed.

### Nokia API Client (`internal/nokia/`)
A shared HTTP client (`client.go`) handles:
- RapidAPI headers (`x-rapidapi-key`, `x-rapidapi-host`)
- Rate limiter: 100 requests/minute with burst of 8 (allows parallel calls)
- 3‑second context deadline per call (10 s for slicing)

The following **nine** endpoints are integrated:

| # | API | Endpoint | Status |
|---|-----|----------|--------|
| 1 | Location Retrieval | `/location-retrieval/v0/retrieve` | real |
| 2 | SIM Swap | `/passthrough/camara/v1/sim-swap/sim-swap/v0/check` | real |
| 3 | Device Reachability | `/device-status/device-reachability-status/v1/retrieve` | real |
| 4 | Device Roaming | `/device-status/device-roaming-status/v1/retrieve` | real |
| 5 | Device Swap | `/passthrough/camara/v1/device-swap/device-swap/v1/check` | real |
| 6 | Quality on Demand | `/quality-on-demand/v1/sessions` | real |
| 7 | Network Slicing | `/slice/v1/slices` | real |
| 8 | Congestion Insights | (stub – always returns `"Low"`) | stub |
| 9 | Number Verification | `/passthrough/camara/v1/number-verification/.../verify` | client exists |

All responses are decoded into typed structs. Errors are logged but never block the pipeline (fail‑safe).

### Enricher Engine (`internal/ingestion/enricher.go`)
- Thread‑safe in‑memory cache (TTL = 2 minutes) using `sync.RWMutex`.
- On each MQTT message, telemetry is updated instantly.
- Network refresh fires up to **7 parallel goroutines** with staggered delays (0–900 ms) to avoid RapidAPI burst limits.
- QoD, Slicing, and Verification calls are made only once; subsequent cycles reuse stored IDs.
- After enrichment, the `SignalMatrix` is persisted to TimescaleDB and optionally sent to the intelligence service.

### Database Layer (`internal/db/`)
- PostgreSQL 17 + TimescaleDB connection pool (`pgxpool`).
- Automatic migration runner (stubbed for hackathon).
- Telemetry hypertable: `telemetry_events` partitioned by `recorded_at`.
- Insertion via `maybePersist()` in the enrichment loop.

### Decision Engine (`internal/decision/engine.go`)
- `POST /score` to `uwatu-intelligence` with 500 ms timeout.
- Bearer token authentication.
- On timeout or error, returns a safe `NORMAL` fallback so the pipeline never stalls.

### Professional ANSI Logger (`internal/config/logger.go`)
- Colour‑coded, zero‑emoji output.
- `LogEnrich` prints a compact dashboard line with bold vital values, colour‑coded by thresholds (fever, high movement, low battery, tamper, roaming, network type).

### WebSocket Hub (`internal/ws/`)
- Maintains a set of active dashboard connections.
- Broadcasts every enriched telemetry update as JSON.
- Role‑filtered: farmers see only their farm’s data; insurers/vets see alert events.

### Dashboard (uwatu-dashboard repo)
- React + TypeScript + Vite + Tailwind + Leaflet.
- Live herd map, animal detail, alert history, insurance dashboard, notifications, settings, onboarding, and login.
- Connects to uwatu-core via WebSocket for real‑time updates.

---

## How to Run

### Prerequisites
- Go 1.22+
- PostgreSQL 17 + TimescaleDB (optional – core runs without DB)
- RapidAPI key for Nokia Network as Code (free Basic plan)
- Africa’s Talking sandbox credentials (optional, for alerts)
- Firebase service account (optional, for push notifications)

### Environment Variables
```bash
export NOKIA_RAPIDAPI_KEY="your-rapidapi-key"
export DATABASE_URL="postgres://user:pass@127.0.0.1:5432/uwatu?sslmode=disable"
export AT_API_KEY="your-africastalking-key"
export FIREBASE_CREDENTIALS="path/to/serviceAccountKey.json"
export JWT_SECRET="your-jwt-secret"
```

### Start the Simulator
(In a separate terminal) – ensure it publishes to `uwatu/farm/+/tag/+` with Nokia magic test numbers (`+99999991000`, etc.).

### Launch uwatu-core
```bash
go run ./cmd/server
```

You will see live telemetry lines with enriched data. If the database is connected, rows appear in the `telemetry_events` hypertable.

### Connect the Dashboard
The dashboard WebSocket connects to `ws://localhost:8080/ws/farm/{farm_id}`. Configure the farm ID in the dashboard’s environment.

---

## Key Design Decisions

- **Staggered API calls** – RapidAPI’s free tier has a per‑second burst limit. Spreading 7 calls over 900 ms avoids 429 errors while still completing well within the 2‑minute cache window.
- **2‑minute cache** – Without caching, the free 500 requests/month quota would be exhausted in minutes. The cache keeps us safely under the limit even with 2 devices.
- **Fail‑safe enrichment** – If any Nokia API fails, the system continues with cached or zero values. The pipeline never blocks on external dependency.
- **No emoji logging** – Professional ANSI colours only. Bold vitals for quick scanning.
- **JSON over CBOR (for hackathon)** – While the blueprint targets CBOR for bandwidth savings, JSON from the simulator is sufficient for the demo. CBOR support is a production roadmap item.
- **TimescaleDB hypertable** – Chosen over pure PostgreSQL or InfluxDB because it gives us both relational data (farmers, animals, geofences) and time‑series performance in a single database.

---

## Remaining Work

- **Alert Router integration** – Mphele’s `alerts/router.go` is partially built; it needs to be connected to the enricher so that scored events trigger SMS/WhatsApp/USSD/Push.
- **Farm Registry & Geofence Manager** – CRUD endpoints and point‑in‑polygon checking for farm boundaries.
- **Intelligence Service** – Mphele’s `uwatu-intelligence` Python service must be running for real classifications.
- **Coverage Map** – A visual map of known network coverage areas.
- **LITS‑Compliant Digital Evidence Export** – Forensic report generation for theft events.

---

## Lessons Learned

- Go’s `json.Unmarshal` treats all numbers as `float64` – always cast explicitly.
- The public HiveMQ broker rejects duplicate client IDs – use a timestamped value.
- The Nokia sandbox is reachable **only** through RapidAPI; the old `networkascode.nokia.io` URL is dead.
- RapidAPI has a per‑second burst limit – staggered goroutines with 150 ms offsets prevent 429 errors.
- Sandbox magic MSISDNs (e.g. `+99999991000`) must be used; any other number returns a 401.
- QoD returns `201 Created`, Slicing returns `202 Accepted` – status checks must accommodate these.
- Number Verification is a one‑time operation (registration) and should not be part of the periodic refresh.
- Multiple PostgreSQL versions on macOS can conflict; uninstalling old versions saved hours of debugging.

---

## Team

- **Elvis Chege** (`github.com/elviscgn`) – Backend (Go), firmware (Zig), simulator, dashboard.
- **Mphele Moswane** (`github.com/Mphele`) – Intelligence layer (Python), alert router, models, config, auth.

Built for the **Africa Ignite Hackathon 2026** by GSMA & Nokia.
