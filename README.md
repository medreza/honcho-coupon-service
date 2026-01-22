# Honcho Coupon Service

A high concurrency 'Flash Sale' Coupon service with strict consistency and data integrity.

## Prerequisites

- **Docker** (it can be Docker Desktop or Docker Engine, with Docker Compose)
- **Make** (should already be available on Linux/MacOs/WSL on Windows)
- **Go 1.24+** (**optional**, only if you want to run tests or build locally without Docker)

## How to Run

### Running the Service
Build and run the service and its dependencies

```bash
make build run
```


### Stopping the Service
Press Ctrl+C in the terminal where 'make run' is running, or run this in another terminal:

```bash
docker compose down
```

## How to Run Test


### 1. Concurrency Tests
This test will run the concurrency tests (FlashSaleAttack and DoubleDipAttack scenarios).
```bash
make test
```

### 2. Manual Testing
Otherwise, you can test the API endpoints manually using tool such as `curl`, for example:

**Create a Coupon:**
```bash
curl -X POST http://localhost:8080/api/coupons \
  -H "Content-Type: application/json" \
  -d '{"name": "PROMO_123", "amount": 100}'
```

**Claim a Coupon:**
```bash
curl -X POST http://localhost:8080/api/coupons/claim \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user_123", "coupon_name": "PROMO_123"}'
```

**Get a Coupon details:**
```bash
curl http://localhost:8080/api/coupons/PROMO_123
```

---

## Architecture Notes

### Database Design
The service uses MongoDB as its primary data store. MongoDB is great for its flexibility and high performance read/write capabilities. To ensure strict consistency and data integrity during high concurrency events, the service leverages Mongo's native multi-document transaction which is supported in replica set configuration. For local development, the `docker-compose.yml` is configured to automatically initialize a single node replica set (`rs0`) which is required for MongoDB to be able to use transaction.

#### Collections

**`coupons`**
| Field | Type | Description |
|-------|------|-------------|
| name | String | Unique name of the coupon |
| amount | Int32 | Total amount of coupons available |
| remaining_amount | Int32 | Current remaining stock (must be >= 0) |
| created_at | DateTime | Timestamp when the coupon was created |

**`claims`**
| Field | Type | Description |
|-------|------|-------------|
| user_id | String | ID of the user who claimed the coupon |
| coupon_name | String | Name of the claimed coupon |
| claimed_at | DateTime | Timestamp when the claim was recorded |

#### Indexes
- **`coupons`**: Unique index on `name`.
- **`claims`**: Unique index on the pair `(user_id, coupon_name)` to prevent double claims.

### Strategy for Handling Concurrency
To prevent race conditions (which might cause claiming more coupons than available in stock), the service implemented these strategies:

1.  **Atomic Transactions**: By utilizing MongoDB's native multi-document transactions, all coupon operation (checking stock, inserting the claim record, and decrementing the remaining count) are wrapped in a transaction. If any step fails in the middle of the operation (e.g. duplicate claim or empty stock), the whole operation will be rolled back.
2.  **Atomic Update (inside the claiming transaction)**: Coupon stock is decremented using the `$inc` operator with a filter that ensures `remaining_amount > 0`. This atomic operation will prevent the stock from ever dropping below zero.
3.  **Database Constraint**: A unique constraint on the `(user_id, coupon_name)` pair in the `claims` collection will prevent any user from claiming the same coupon twice, even if the endpoint calls are made at the same time.
