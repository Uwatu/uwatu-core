# uwatu-core – Uwatu Unified Livestock Protection Platform (API Gateway)

---

## Project Status

- [x] **Core framework** – Fiber v2 server, MQTT client, configuration via Viper
- [x] **Nokia API Client** – RapidAPI gateway with rate limiting & shared transport
- [x] **Location Retrieval** – endpoint working with sandbox (test numbers `+99999991000`, etc.)
- [x] **SIM Swap Detection** – endpoint integrated
- [x] **MQTT Ingestion** – live telemetry from simulator, JSON payload parsing
- [x] **Enricher Engine** – thread‑safe caching, parallel Nokia calls, SignalMatrix assembly
- [x] **Professional Logger** – ANSI‑colored, bold key values, zero clutter
- [ ] **Database layer** – TimescaleDB/PostgreSQL with migrations (assigned to Elvis)
- [ ] **Fiber server wiring** – full setup: routes, middleware, DB pool (assigned to Elvis)
- [ ] **Remaining Nokia APIs** – Device Status, Connectivity, Congestion, Roaming, QoD, Slicing, Number Verification (assigned to Elvis)
- [ ] **Models & shared structs** – TagTelemetry, SignalMatrix, ScoredEvent, AlertPayload (assigned to Mphele)
- [ ] **Farm registry, animal CRUD, geofence manager** – core domain logic (assigned to Mphele)
- [ ] **Alert router** – tier‑based channel selection, goroutines per notification (assigned to Mphele)
- [ ] **Africa's Talking integration** – SMS, WhatsApp, USSD senders (assigned to Mphele)
- [ ] **Firebase FCM** – push notifications for Tier‑3 devices (assigned to Mphele)
- [ ] **Decision engine handoff** – POST /score to uwatu‑intelligence with 500ms timeout (assigned to Elvis)
- [ ] **Tests** – unit tests for Nokia client, table‑driven tests for alert routing
- [ ] **Documentation** – README, API contracts, architecture diagrams

**Legend:** `x` = completed, ` ` = pending

---

## How to Run (development)

1. **Prerequisites**
    - Go 1.22+
    - Git
    - Access to the Uwatu simulator (or real tags)
    - RapidAPI key for Nokia Network as Code (free Basic plan)

2. **Set environment variables**
   ```bash
   export NOKIA_RAPIDAPI_KEY="your-rapidapi-key"
   ```

3. **Start the simulator**  
   (in a separate terminal) – ensure it publishes MQTT to `uwatu/farm/+/tag/+` with Nokia magic test numbers.

4. **Run the core**
   ```bash
   go run ./cmd/server
   ```
   You will see live telemetry lines with enriched location data.

---

## Architecture Overview

```text
Simulator / Real Tags
       │  MQTT (CBOR/JSON)
       ▼
┌─────────────────────┐
│  Ingestion Handler  │
│ • parses nested JSON│
│ • extracts sensor   │
│   values            │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│     Enricher        │
│ • 2‑minute cache    │
│ • parallel Nokia    │
│   API calls         │
│ • builds SignalMatrix│
└────────┬────────────┘
         │
         ▼
   (to uwatu‑intelligence)
   POST /score → classification
         │
         ▼
┌─────────────────────┐
│   Alert Router      │
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
- Unmarshalls the simulator's nested JSON: extracts `device_id`, `msisdn`, and the `firmware_payload` fields (`body_temp_c`, `accel_magnitude`, `battery_pct`).
- Converts all JSON numbers (which Go treats as `float64`) to the correct Go types.

### Nokia API Client (`internal/nokia/`)

- Shared client (`client.go`) attaches RapidAPI headers and enforces a 100 req/min rate limit.
- Each API call gets a 3‑second context deadline.
- **Location Retrieval:** `POST /location-retrieval/v0/retrieve` – returns lat, lon, accuracy.
- **SIM Swap Detection:** `POST /sim-swap/v0/check` – returns `swapped` boolean.
- All responses are decoded into proper structs, and errors are handled without blocking the pipeline.

### Enricher Engine (`internal/ingestion/enricher.go`)

- Caches the last location/SIM status per device (TTL = 2 minutes) using `sync.RWMutex`.
- On each MQTT message:
    - Telemetry values are updated immediately.
    - If the cache is stale, it fires parallel goroutines to refresh Nokia data.
    - A 200ms stagger between location calls prevents RapidAPI burst limits.
    - If a Nokia API fails, the system continues with cached/default values (fail‑safe).

### Logging (`internal/config/logger.go`)

- Clean, colour‑coded output: `[INFO]` in cyan, `[OK]` in green, `[ERR]` in red.
- A dedicated `LogEnrich` function prints a scannable dashboard line with bold vital values and a red `ALERT` when a SIM swap is detected.

---

## Upcoming Work (linked issues)

- `#10` – Database wiring (Elvis)
- `#7`, `#9` – Remaining Nokia APIs (Elvis)
- `#11` – Decision engine handoff to uwatu‑intelligence (Elvis)
- `#15` – Shared data structures (Mphele)
- `#16` – Farm registry, animal CRUD, geofence manager (Mphele)
- `#12` – Alert router (Mphele)
- `#13` – Africa's Talking integration (Mphele)
- `#14` – Firebase FCM (Mphele)
- `#1` – LITS‑compliant digital evidence export (feature)

All issues are tracked in the repository's issue tracker.

---

## Integration Contract (with uwatu‑intelligence)

When the decision engine call is implemented, `uwatu-core` will `POST /score` to the intelligence service with a JSON payload containing:

- Telemetry snapshot (temp, accel, battery, RSSI, cell ID, etc.)
- All available Nokia signals (location, SIM swap, connectivity, etc.)
- Context (time, season, herd state)

Timeout: 500ms. On timeout or error, the last cached classification is used.

---

## Lessons Learned

- Go's `json.Unmarshal` treats all numbers as `float64` – always cast explicitly.
- The public HiveMQ broker rejects duplicate client IDs – use a timestamped value.
- The Nokia sandbox is reachable only through RapidAPI; the old `networkascode.nokia.io` URL is dead.
- A 200ms delay between concurrent location calls avoids 429 errors from RapidAPI's per‑second burst limit.
- The 2‑minute cache keeps us safely within the free 500 req/month quota.
- The sandbox must use Nokia's test phone numbers (like `+99999991000`); any other number returns a 401.