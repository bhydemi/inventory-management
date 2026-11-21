# Inventory Management API

A backend inventory management service built with [Go](https://go.dev/),
the [Gin](https://github.com/gin-gonic/gin) web framework, and
[GORM](https://gorm.io/) over PostgreSQL. It exposes a RESTful API for
creating, reading, updating, and deleting inventory items, with
pagination, filtering, sorting, and per-client rate limiting built in.

## Project structure

```
.
├── main.go                      # entry point: wires up routes and starts the server
├── models/
│   └── item.go                  # Item struct + request-input struct + sort whitelist
├── database/
│   └── database.go              # PostgreSQL connection, migration, and seeding
├── handlers/
│   ├── item_handler.go          # one handler function per inventory endpoint
│   └── auth_handler.go          # POST /login handler
├── middleware/
│   ├── rate_limiter.go          # per-client token-bucket rate limiting
│   └── auth.go                  # JWT issuing + verification middleware
├── cache/
│   └── cache.go                 # in-memory TTL cache for GET responses
└── docs/                        # generated Swagger spec (see "API documentation" below)
```

## Prerequisites

- [Go](https://go.dev/doc/install) 1.23+
- A local PostgreSQL server

## 1. Set up PostgreSQL

Install PostgreSQL if you don't already have it:

```bash
brew install postgresql@15
brew services start postgresql@15
```

Create a database and a `postgres` role for the app to use:

```bash
createdb inventory
psql -d postgres -c "CREATE ROLE postgres WITH LOGIN SUPERUSER PASSWORD 'postgres';"
```

(If a `postgres` role already exists on your system, skip that second
command.)

## 2. Configure the connection string

The server connects using the DSN in `database/database.go`:

```
host=localhost user=postgres password=postgres dbname=inventory port=5432 sslmode=disable
```

This matches a default local PostgreSQL install. If your setup differs
(different port, user, password, or database name), override it with an
environment variable instead of editing the code:

```bash
export DATABASE_URL="host=localhost user=postgres password=postgres dbname=inventory port=5432 sslmode=disable"
```

## 3. Install dependencies and run

```bash
go mod tidy
go run .
```

On first run, the server connects to PostgreSQL, creates the `items`
table (via GORM's `AutoMigrate`), and seeds it with 10 starter items if
the table is empty. You should see:

```
seeded database with 10 item(s)
listening on :8080
```

The server listens on `:8080` by default; set `PORT` to use a different
port.

## API documentation (Swagger)

Interactive, browsable API docs are generated from annotations in the
handler code and served at:

```
http://localhost:8080/swagger/index.html
```

Use the **Authorize** button there to paste a bearer token (see
Authentication below) and try the protected endpoints directly from the
browser. The docs are pre-generated and committed under `docs/`; if you
change a handler's annotations, regenerate them with:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init --parseDependency --parseInternal
```

## Authentication

Reads (`GET`) are open to anyone. Writes (`POST`, `PUT`, `DELETE` on
`/inventory`) require a JWT, obtained by logging in:

```bash
curl -X POST "http://localhost:8080/login" \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'
# => {"token": "eyJhbGciOi..."}
```

Then pass it on any write request:

```bash
curl -X POST "http://localhost:8080/inventory" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"name": "Desk Lamp", "stock": 40, "price": 24.99}'
```

A missing, malformed, or expired token returns `401`. The demo
credentials (`admin` / `admin123`) and the JWT signing secret can be
overridden via the `AUTH_USERNAME`, `AUTH_PASSWORD`, and `JWT_SECRET`
environment variables.

## Caching

`GET /inventory` and `GET /inventory/:id` responses are cached in memory
for 10 seconds, keyed by the exact query string (or item ID), to avoid
re-querying the database for identical, frequently-repeated reads. Any
write (create, update, delete) immediately flushes the entire cache, so a
client can never read stale data back after making a change.

## API reference

All responses are JSON. The base path is `/inventory`.

| Method | Path             | Description                  |
|--------|------------------|-------------------------------|
| GET    | `/inventory`     | List items (paginated, filterable, sortable) |
| GET    | `/inventory/:id` | Get a single item by ID       |
| POST   | `/inventory`     | Create a new item             |
| PUT    | `/inventory/:id` | Update an existing item       |
| DELETE | `/inventory/:id` | Delete an existing item       |

### List items — `GET /inventory`

Query parameters (all optional):

| Param       | Meaning                                      | Default |
|-------------|-----------------------------------------------|---------|
| `limit`     | max items to return                          | `10`    |
| `offset`    | number of items to skip                      | `0`     |
| `sort_by`   | `name`, `stock`, or `price`                  | `name`  |
| `order`     | `asc` or `desc`                               | `asc`   |
| `name`      | case-insensitive substring filter on name    | —       |
| `min_stock` | only items with `stock >=` this value        | —       |

```bash
curl "http://localhost:8080/inventory?limit=5&offset=0&sort_by=price&order=desc"
curl "http://localhost:8080/inventory?name=phone"
curl "http://localhost:8080/inventory?min_stock=20"
```

Response:

```json
{
  "data": [ { "id": "...", "name": "Laptop", "stock": 10, "price": 999.99 } ],
  "total": 10,
  "limit": 5,
  "offset": 0
}
```

### Get a single item — `GET /inventory/:id`

```bash
curl "http://localhost:8080/inventory/<item-id>"
```

Returns `200` with the item, or `404` with `{"error": "item not found"}`
if the ID doesn't exist or isn't a valid UUID (`400` in the latter case).

### Create an item — `POST /inventory` (requires auth)

```bash
curl -X POST "http://localhost:8080/inventory" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"name": "Desk Lamp", "stock": 40, "price": 24.99}'
```

Returns `201` with the created item (server-assigned UUID included),
`400` if `name` is missing or `stock`/`price` is negative, or `401` if
the bearer token is missing/invalid (see Authentication above).

### Update an item — `PUT /inventory/:id` (requires auth)

```bash
curl -X PUT "http://localhost:8080/inventory/<item-id>" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"name": "Desk Lamp (LED)", "stock": 35, "price": 29.99}'
```

Returns `200` with the updated item, or `404` if the ID doesn't exist.

### Delete an item — `DELETE /inventory/:id` (requires auth)

```bash
curl -X DELETE "http://localhost:8080/inventory/<item-id>" \
  -H "Authorization: Bearer <token>"
```

Returns `200` with a confirmation message, or `404` if the ID doesn't
exist.

## Rate limiting

Each client (identified by IP) is limited to **1 request per second**,
with a **burst capacity of 5** — implemented as a token-bucket limiter
(`golang.org/x/time/rate`) applied as Gin middleware to every route.
Exceeding the limit returns `429 Too Many Requests`:

```json
{ "error": "rate limit exceeded: 1 request per second, burst of 5" }
```

Idle clients are cleaned up from memory after a few minutes of
inactivity, so the limiter doesn't grow unbounded over the life of the
server.

## Error handling

- Malformed or missing request bodies return `400` with a descriptive
  `error` message.
- A path `:id` that isn't a valid UUID returns `400` before any database
  query runs.
- Looking up, updating, or deleting an item that doesn't exist returns
  `404` with `{"error": "item not found"}`.
- Every item gets a server-generated UUID as its primary key, so no two
  items can ever collide on ID.
