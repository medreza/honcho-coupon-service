# Honcho Coupon Service

> By the way, I also made an alternative service implementation using MongoDB. The Mongo based implementation can be found in the `use-mongo` branch: https://github.com/medreza/honcho-coupon-service/tree/use-mongo

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
The service uses PostgreSQL with a clear separation of concerns between coupon definitions and claim history. 

PostgreSQL is chosen for this service (over MongoDB or other NoSQL databases) because relational database offers better consistency and data integrity than non-relational database, and PostgreSQL is ACID-compliant out of the box and has native row-level locking, which is perfect for this service use case.

#### Coupons Table
| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PRIMARY KEY |
| name | VARCHAR(255) | UNIQUE, NOT NULL |
| amount | INT | NOT NULL, CHECK (>= 0) |
| remaining_amount | INT | NOT NULL, CHECK (>= 0) |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

#### Claims Table
| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PRIMARY KEY |
| user_id | VARCHAR(255) | NOT NULL |
| coupon_name | VARCHAR(255) | NOT NULL, REFERENCES coupons(name) |
| claimed_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

##### Claims Table Constraints
| Type | Columns | Description |
|--------|------|-------------|
| UNIQUE | (user_id, coupon_name) | Prevents double claims so that each user can only claim a coupon once |

### Strategy for Handling Concurrency
To prevent race conditions (which might cause claiming more coupons than available in stock), the service implemented these strategies:

1.  **Row Locking (`SELECT FOR UPDATE`)**: Inside the database transaction, the service locks the specific coupon row when it's being claimed. This will make sure that when a request is checking and updating the stock, other request can't touch that same row at the same time.
2.  **Database Constraint**: A unique constraint on the `(user_id, coupon_name)` pair in the `claims` table will prevent any user from claiming the same coupon twice, even if the endpoint calls are made at the same time.
3.  **Atomic Transactions**: All coupon operation (checking stock, inserting the claim record, and decrementing the remaining count) are wrapped in a database transaction. If any step fails in the middle of the operation (e.g. duplicate claim or empty stock), the whole operation will be rolled back.
