# uwatu-core – Uwatu Unified Livestock Protection Platform (API Gateway)

Repository: `github.com/uwatu/uwatu-core`  
Team: Elvis Chege (@elviscgn) & Mphele Moswane (@Mphele)  
Hackathon: Africa Ignite 2026 (GSMA & Nokia)

---

## Project Status

- [x] **Core framework** – Fiber v2 server, MQTT client, Viper config, JWT auth middleware
- [x] **Nokia API Client** – RapidAPI gateway with shared transport & rate limiting
- [x] **All 9 Nokia APIs** – Location, SIM Swap, Device Reachability, Roaming, Device Swap, QoD, Slicing, Congestion (stub), Number Verification (client)
- [x] **MQTT Ingestion** – JSON payload parsing with type‑safe conversion
- [x] **Enricher Engine** – thread‑safe caching, parallel staggered Nokia calls, SignalMatrix assembly
- [x] **Models** – TagTelemetry, SignalMatrix, ScoredEvent, AlertPayload, NokiaSignals
- [x] **Environment Config** – Viper‑based, JWT secret, RapidAPI key, DB credentials
- [x] **Auth Middleware** – JWT generation/validation, RBAC middleware
- [x] **Notification Dispatchers** – Africa's Talking SMS, WhatsApp, USSD, Firebase FCM
- [x] **Professional ANSI Logger** – colour‑coded, bold vital signs, zero emoji
- [ ] **Database layer** – TimescaleDB/PostgreSQL with migrations (Elvis)
- [ ] **Alert router** – tier‑based channel selection, notification_log writes (Mphele)
- [ ] **Farm registry, animal CRUD, geofence manager** – core domain logic (Mphele)
- [ ] **Decision engine handoff** – POST /score to uwatu‑intelligence with 500ms timeout (Elvis)
- [ ] **LITS‑compliant digital evidence export** – forensic report generation (Elvis)

**Legend:** `x` = completed, ` ` = pending

---

## How to Run (development)

1. **Prerequisites**
    - Go 1.22+
    - Git
    - Access to the Uwatu simulator (or real tags)
    - RapidAPI key for Nokia Network as Code (free Basic plan)
    - Africa's Talking sandbox credentials (optional, for notify)

2. **Set environment variables**
```bash
export NOKIA_RAPIDAPI_KEY="your-rapidapi-key"
export AT_API_KEY="your-africastalking-key"       # optional
export FIREBASE_CREDENTIALS="path/to/serviceAccountKey.json"  # optional
export JWT_SECRET="your-jwt-secret"
export DB_DSN="postgres://user:pass@localhost:5432/uwatu?sslmode=disable"
```

3. **Start the simulator**
   (in a separate terminal) – ensure it publishes MQTT to `uwatu/farm/+/tag/+` with Nokia magic test numbers.

4. **Run the core**
```bash
go run ./cmd/server
```
You will see live telemetry lines with enriched Nokia data.

---

## Architecture Overview

```text
Simulator / Real Tags
       │  MQTT (CBOR/JSON)
       ▼
┌─────────────────────┐
│  Ingestion Handler  │
│ • parses nested JSON│
│ • builds TagTelemetry│
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│     Enricher         │
│ • 2‑minute cache    │
│ • parallel Nokia    │
│   API calls (7 real)│
│ • assembles SignalMatrix│
└────────┬────────────┘
         │
         ▼
   (to uwatu‑intelligence)
   POST /score → classification
         │
         ▼
┌─────────────────────┐
│   Alert Router       │
│ • tier‑based channel│
│   selection         │
│ • SMS/WhatsApp/USSD │
│   /Push             │
└─────────────────────┘
```

---

## Implemented Features – Detail

### MQTT Ingestion (`internal/ingestion/handler.go`)
- Subscribes to `uwatu/farm/+/tag/+` on the HiveMQ public broker.
- Unique client ID prevents auth conflicts.
- Unmarshalls the simulator's nested JSON: extracts `device_id`, `msisdn`, and all fields of the `firmware_payload`.
- Converts JSON numbers (which Go treats as `float64`) to the correct Go types defined in `TagTelemetry`.

### Nokia API Client (`internal/nokia/`)
- Shared client (`client.go`) attaches RapidAPI headers (`x-rapidapi-key`, `x-rapidapi-host`) and enforces a **100 req/min** rate limit with a burst of 8 to allow parallel calls.
- Each API call is given a **3‑second context deadline** (10 s for slicing).
- The following endpoints are fully integrated:

| # | API | Endpoint | Status |
|---|-----|----------|--------|
| 1 | Location Retrieval | `/location-retrieval/v0/retrieve` | real |
| 2 | SIM Swap | `/passthrough/camara/v1/sim-swap/sim-swap/v0/check` | real |
| 3 | Device Reachability | `/device-status/device-reachability-status/v1/retrieve` | real |
| 4 | Device Roaming | `/device-status/device-roaming-status/v1/retrieve` | real |
| 5 | Device Swap | `/passthrough/camara/v1/device-swap/device-swap/v1/check` | real |
| 6 | QoD | `/quality-on-demand/v1/sessions` | real |
| 7 | Slicing | `/slice/v1/slices` | real |
| 8 | Congestion Insights | (stub – returns `"Low"`) | stub |
| 9 | Number Verification | `/passthrough/camara/v1/number-verification/number-verification/v0/verify` | client exists |

- Responses are decoded into strongly‑typed structs; errors are logged but never block the pipeline (fail‑safe).

### Enricher Engine (`internal/ingestion/enricher.go`)
- Caches the full Nokia signal set per device (TTL = 2 minutes) using `sync.RWMutex`.
- On each MQTT message:
    - Telemetry values are updated immediately.
    - If the cache is stale, up to **7 parallel goroutines** fire with staggered delays (0‑900 ms) to respect RapidAPI burst limits.
    - Calls to QoD, Slicing, and Number Verification are only made once; subsequent refreshes reuse the stored session/slice/verification IDs.
    - If any Nokia API fails, the system continues with cached/default values (fail‑safe).

### Models (`internal/models/`)
- `TagTelemetry` – maps the firmware payload.
- `NokiaSignals` – all nine network signals (location, swaps, reachability, roaming, congestion, QoD/slice/verify IDs).
- `SignalMatrix` – top‑level envelope combining telemetry, Nokia signals, baseline, and context.
- `ScoredEvent` – classification response from intelligence layer.
- `AlertPayload` – final alert data handed to the dispatchers.

### Configuration & Auth (`internal/config/`)
- Viper‑based environment config (`config.go`) with structured validation.
- JWT token generation and validation middleware (`auth.go`).
- RBAC middleware for role‑based endpoint protection (`rbac.go`).

### Notification Dispatchers (`internal/notify/`)
- **Africa's Talking** senders for SMS, WhatsApp (template‑based), and USSD push.
- **Firebase FCM** sender for tier‑3 push notifications.
- Each dispatcher implements fallback: if one channel fails, the next is tried automatically.

### Logging (`internal/config/logger.go`)
- Colour‑coded output: `[INFO]` in cyan, `[OK]` in green, `[ERR]` in red.
- `LogEnrich` prints a one‑line dashboard with bold vital signs:
    - Temperature, accelerometer, battery (colour‑coded by thresholds)
    - Location (dimmed if zero)
    - TAMPER (red `ALERT` if SIM or device swapped)
    - ROAM (red `ROAM` if roaming)
    - NET (green `DAT`, yellow `SMS`, red `OFF`)
    - CGEST (congestion level stub)

---

## Upcoming Work (open issues)

- **#10** – Database wiring & TimescaleDB migrations (Elvis)
- **#11** – Decision engine handoff to uwatu‑intelligence (Elvis)
- **#12** – Alert router: tier‑switch + per‑channel goroutines + notification_log writes (Mphele)
- **#16** – Farm registry, animal CRUD, geofence manager (Mphele)
- **#1** – LITS‑compliant digital evidence export (Elvis)

---

## Integration Contract (with uwatu‑intelligence)

Once `POST /score` is implemented, `uwatu-core` will send the full `SignalMatrix` JSON to the intelligence service.  
**Timeout:** 500ms. **Fallback:** last cached classification.  
The payload includes:

- `telemetry` – all TagTelemetry fields
- `nokia_signals` – all NokiaSignals fields
- `baseline` – rolling averages (once DB is available)
- `context` – time of day, season, etc.

---

## Lessons Learned

- Go's `json.Unmarshal` treats all numbers as `float64` – always cast explicitly.
- The public HiveMQ broker rejects duplicate client IDs – use a timestamped value.
- The Nokia sandbox is reachable **only** through RapidAPI; the old `networkascode.nokia.io` URL is dead.
- RapidAPI has a per‑second burst limit – staggered goroutines with 150 ms offsets avoid 429 errors.
- The 2‑minute cache keeps Nokia API consumption well within free tier limits.
- Sandbox magic MSISDNs (e.g. `+99999991000`) must be used; any other number returns a 401.
- QoD returns `201 Created`, Slicing returns `202 Accepted` – status checks must accommodate these.
- Number Verification is a one‑time operation (registration) and should not be part of the periodic refresh.