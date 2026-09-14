# Business Requirements Document & System Design (BRD)
## Lottery Search System — Production Architecture & Engineering Specification

เอกสารฉบับนี้เป็น **Business Requirement Document (BRD) & System Design Proposal** ฉบับสมบูรณ์สำหรับระบบ **Lottery Search System** โดยสรุปและสังเคราะห์จากข้อกำหนดใน [requirement.md](./requirement.md) และเอกสารการออกแบบเชิงสถาปัตยกรรมทั้งหมดในโฟลเดอร์ [`documents/design/`](../../design/) เพื่อรองรับการค้นหาและจัดสรรสลากกินแบ่งรัฐบาล (Lottery Tickets) ขนาดใหญ่ 10,000,000 ใบขึ้นไป พร้อมรองรับการค้นหาแบบ Wildcard (`*`) และการป้องกันการแจกสลากซ้ำซ้อนในระดับ High Concurrency

---

## สารบัญ

1. [ภาพรวมและขอบเขตความต้องการของระบบ (Executive Summary & Requirements)](#1-ภาพรวมและขอบเขตความต้องการของระบบ-executive-summary--requirements)
2. [ข้อสังเกตสำคัญเชิงสถาปัตยกรรม (Key Architectural Insights)](#2-ข้อสังเกตสำคัญเชิงสถาปัตยกรรม-key-architectural-insights)
3. [สถาปัตยกรรมระบบโดยรวม (High-Level System Architecture)](#3-สถาปัตยกรรมระบบโดยรวม-high-level-system-architecture)
4. [การออกแบบฐานข้อมูลและ Data Model (Database Schema & ER Diagram)](#4-การออกแบบฐานข้อมูลและ-data-model-database-schema--er-diagram)
5. [กลยุทธ์การทำ Index และอัลกอริทึมค้นหา Wildcard (Search Strategy & Query Routing)](#5-กลยุทธ์การทำ-index-และอัลกอริทึมค้นหา-wildcard-search-strategy--query-routing)
6. [กลไกจัดสรรสลากและป้องกันการชนกัน (Concurrency & Distribution Mechanism)](#6-กลไกจัดสรรสลากและป้องกันการชนกัน-concurrency--distribution-mechanism)
7. [ลำดับขั้นตอนการทำงาน (Sequence Diagram: Concurrent Allocation Flow)](#7-ลำดับขั้นตอนการทำงาน-sequence-diagram-concurrent-allocation-flow)
8. [ข้อกำหนด API Interface และวงจรการทำงาน (API Specification & Lifecycle)](#8-ข้อกำหนด-api-interface-และวงจรการทำงาน-api-specification--lifecycle)
9. [การเปรียบเทียบเทคโนโลยีจัดเก็บข้อมูล (Technology Evaluation & Selection)](#9-การเปรียบเทียบเทคโนโลยีจัดเก็บข้อมูล-technology-evaluation--selection)
10. [การวิเคราะห์ประสิทธิภาพและการรองรับโหลดสูง (Performance, Scaling & KPIs)](#10-การวิเคราะห์ประสิทธิภาพและการรองรับโหลดสูง-performance-scaling--kpis)
11. [ความน่าเชื่อถือและการรับมือความล้มเหลว (Failure Handling & Recovery)](#11-ความน่าเชื่อถือและการรับมือความล้มเหลว-failure-handling--recovery)
12. [สรุปผลการออกแบบตามเกณฑ์การประเมิน (Evaluation Criteria Alignment)](#12-สรุปผลการออกแบบตามเกณฑ์การประเมิน-evaluation-criteria-alignment)
13. [เอกสารและไฟล์สคริปต์ที่เกี่ยวข้อง (Cross-References & Design Artifacts)](#13-เอกสารและไฟล์สคริปต์ที่เกี่ยวข้อง-cross-references--design-artifacts)

---

## 1. ภาพรวมและขอบเขตความต้องการของระบบ (Executive Summary & Requirements)

### 1.1 วัตถุประสงค์
ออกแบบระบบระดับ Production สำหรับการค้นหาสลากกินแบ่งรัฐบาลที่มีชุดข้อมูล **10,000,000 รายการ (10 Million Tickets)** โดยผู้ใช้สามารถระบุรูปแบบตัวเลขผสมเครื่องหมาย Wildcard (`*`) ความยาว 6 ตัวอักษร พร้อมกลไกจัดสรรสลากที่การันตีว่า **ผู้ใช้หลายคนที่ค้นหา Pattern เดียวกันในเวลาพร้อมกัน จะต้องไม่ได้รับสลากใบเดียวกันอย่างเด็ดขาด (No Duplicate Simultaneous Selection)**

### 1.2 สรุป Requirement Matrix

| หัวข้อความต้องการ | รายละเอียดตาม Requirement | สถาปัตยกรรมและผลลัพธ์ที่ตอบสนอง |
|---|---|---|
| **1. Data Volume** | รองรับสลาก **10,000,000 ใบ** แต่ละใบเป็นเลข 6 หลัก (`000000` - `999999`) | ตาราง `tickets` ขนาด ~1.2 GB + Indexes ~800 MB รวม ~2 GB สามารถ **Cache อยู่ใน RAM (Shared Buffers) ได้ 100%** |
| **2. Search Pattern** | รูปแบบ 6 ตัวอักษร เช่น `****23`, `1****5`, `123***` | มี Internal Query Routing Planner เลือก B-Tree Range, Digit Index, หรือ Reverse Index ตามความเหมาะสม |
| **3. Result Distribution** | หากค้นหา Pattern เดียวกันพร้อมกัน ต้องไม่คืนสลากซ้ำให้ผู้ใช้หลายคน | ใช้กลไก **Row-level Locking (`FOR UPDATE SKIP LOCKED`)** ร่วมกับ **Redis Lock / Unique Constraint** |
| **4. Performance** | ค้นหาและจัดสรรได้รวดเร็วที่ระดับ 10M+ records | Read/Write Separation (ค้นหาผ่าน Read Replica / Redis Cache, จองผ่าน Primary DB) Response Time < 50ms |
| **5. Real-World Proposal** | เสนอ Database, Indexing Strategy, Concurrency Control (ไม่ต้องเขียนโค้ด) | เลือก **Go + PostgreSQL + Redis** มี DDL SQL Scripts, ER Diagram, และ Sequence Diagram ครบถ้วน |

---

## 2. ข้อสังเกตสำคัญเชิงสถาปัตยกรรม (Key Architectural Insights)

1. **Search Space มีขนาดจำกัด (Bounded Domain 1,000,000 รูปแบบ)**
   - แม้จำนวนสลากจริงจะมี 10,000,000 ใบ แต่ตัวเลขสลาก 6 หลักมีชุดตัวเลขที่เป็นไปได้จริงเพียง **1,000,000 รูปแบบ** (`000000` ถึง `999999`)
   - ในโลกจริง สลากแต่ละหมายเลขสามารถมีได้หลายใบ (เช่น สลากชุด, เล่มต่างกัน) ดังนั้น **หมายเลขสลากไม่ใช่ Unique Key** แต่ `ticket_id` คือ Unique Identity หลัก
2. **การแยกหน้าที่ระหว่าง Search กับ Allocation (Separation of Concerns)**
   - **Search (Browse)**: ผู้ใช้เปิดดูผลลัพธ์ว่ามีสลากที่ตรงกับเลขที่ต้องการหรือไม่ (Read-heavy, ใช้ Read Replica หรือ Redis Cache, ไม่มีการล็อคข้อมูล)
   - **Allocation (Reserve/Hold)**: ผู้ใช้กดเลือกซื้อ หรือระบบสุ่มสลากใบจริงให้ผู้ใช้ (Transactional Write, ต้องล็อคสลากแบบ Atomic บน Primary Database)
3. **การหลีกเลี่ยงปัญหา Data Explosion**:
   - การสร้างตาราง Index จับคู่ทุก Pattern ที่เป็นไปได้ ($11^6 - 1 = 1,771,560$ รูปแบบ) กับสลาก 10 ล้านใบ จะทำให้เกิดข้อมูลนับหมื่นล้านแถว ดังนั้นจึงเลือกใช้ **ตาราง `patterns` บันทึกเฉพาะ Pattern ที่ถูกค้นหาจริง** ควบคู่กับ **Functional Digit Columns (`p1..p6`)** เพื่อประสิทธิภาพสูงสุด

---

## 3. สถาปัตยกรรมระบบโดยรวม (High-Level System Architecture)

ระบบใช้แนวคิด **Command-Query Responsibility Segregation (CQRS-lite)** แยกเส้นทางระหว่างการสืบค้น (Search) และการทำธุรกรรมจัดสรร (Allocation):

```mermaid
flowchart TD
    Client([Client / Web / Mobile]) --> API_GW[API Gateway / Load Balancer]
    
    subgraph Service_Tier [Go Application Cluster]
        SearchSvc[Search Service Pods]
        AllocSvc[Allocation Service Pods]
        ReaperWorker[Reservation Expiry Reaper]
    end
    
    API_GW -->|GET /lottery/search| SearchSvc
    API_GW -->|POST /lottery/allocate| AllocSvc
    
    subgraph Data_Tier [Data & Cache Infrastructure]
        RedisCache[(Redis Cluster\n- Pattern Cache 2-3s TTL\n- Distributed Lock / Rate Limit)]
        PG_Primary[(PostgreSQL Primary\n- Source of Truth\n- Allocations & Ticket State\n- FOR UPDATE SKIP LOCKED)]
        PG_Replica[(PostgreSQL Read Replicas\n- Wildcard Search Engine\n- B-Tree & Digit Indexes)]
    end
    
    SearchSvc -->|1. Check Popular Pattern| RedisCache
    SearchSvc -->|2. Query Matching Tickets| PG_Replica
    AllocSvc -->|Acquire Lock if Pattern Queue| RedisCache
    AllocSvc -->|Atomic Transaction & Row Lock| PG_Primary
    PG_Primary -.->|Streaming Replication| PG_Replica
    ReaperWorker -->|Revert Expired Holds Every 5s| PG_Primary
```

---

## 4. การออกแบบฐานข้อมูลและ Data Model (Database Schema & ER Diagram)

อ้างอิงโครงสร้างมาตรฐานจาก [table_5.png](../../design/database/schema/table_5.png) และไฟล์สคริปต์ใน [`documents/design/database/sql_script/`](../../design/database/sql_script/)

### 4.1 ข้อกำหนดฐานข้อมูลและตารางหลัก 5 ตาราง (Database Specification & Schema)

> **Database Configuration**:
> - **Database Name**: `lottery_system` (Encoding: `UTF8`)
> - **Bootstrap Script**: [`00_init_db_lottery.sql`](../../design/database/sql_script/00_init_db_lottery.sql) (สคริปต์เดียวสร้างทั้ง DB `lottery_system` และตาราง ดัชนี ข้อจำกัดครบถ้วน)
> - **Connection String**: `postgresql://postgres:postgres@localhost:5432/lottery_system?sslmode=disable`



```text
┌──────────────────┐       ┌──────────────────────┐       ┌──────────────────┐
│      users       │       │     allocations      │       │     tickets      │
├──────────────────┤       ├──────────────────────┤       ├──────────────────┤
│ PK user_id       │◄──1:N─┤ PK allocation_id     │─N:1──►│ PK ticket_id     │
│    name          │       │ FK user_id           │       │    ticket_number │
│    email (UQ)    │       │ FK ticket_id         │       │    allocated     │
│    created_at    │       │    pattern           │       │    allocated_at  │
└──────────────────┘       │    allocated_at      │       └─────────┬────────┘
                           │    expires_at        │                 │
                           │    status            │                 │
                           └──────────────────────┘                 │
                                                                   1:N
┌──────────────────┐       ┌──────────────────────┐                 │
│     patterns     │       │    pattern_index     │                 │
├──────────────────┤       ├──────────────────────┤                 ▼
│ PK pattern_id    │◄──1:N─┤ PK index_id          │◄──N:1───────────┘
│    pattern_text  │       │ FK pattern_id        │
│    created_at    │       │    position (1-6)    │
└──────────────────┘       │    digit (0-9)       │
                           │ FK ticket_id         │
                           └──────────────────────┘
```

#### 1. ตาราง [`users`](../../design/database/sql_script/01_users.sql)
```sql
CREATE TABLE IF NOT EXISTS users (
    user_id     BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    email       VARCHAR(100) UNIQUE NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### 2. ตาราง [`tickets`](../../design/database/sql_script/02_tickets.sql)
```sql
CREATE TABLE IF NOT EXISTS tickets (
    ticket_id       BIGINT PRIMARY KEY,
    ticket_number   CHAR(6) NOT NULL,
    allocated       BOOLEAN NOT NULL DEFAULT FALSE,
    allocated_at    TIMESTAMP NULL,

    CONSTRAINT uq_tickets_ticket_number UNIQUE (ticket_number)
);

CREATE INDEX idx_tickets_allocated_number ON tickets (allocated, ticket_number);
```

#### 3. ตาราง [`patterns`](../../design/database/sql_script/03_patterns.sql)
```sql
CREATE TABLE IF NOT EXISTS patterns (
    pattern_id    BIGSERIAL PRIMARY KEY,
    pattern_text  VARCHAR(6) NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_patterns_pattern_text UNIQUE (pattern_text)
);

CREATE INDEX idx_patterns_pattern_text ON patterns (pattern_text);
```

#### 4. ตาราง [`allocations`](../../design/database/sql_script/04_allocations.sql)
```sql
CREATE TABLE IF NOT EXISTS allocations (
    allocation_id   BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    ticket_id       BIGINT NOT NULL REFERENCES tickets(ticket_id) ON DELETE RESTRICT,
    pattern         VARCHAR(6) NOT NULL,
    allocated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      TIMESTAMP NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
);

CREATE INDEX idx_allocations_user_allocated_at ON allocations (user_id, allocated_at);
CREATE INDEX idx_allocations_active_expires_at ON allocations (expires_at) WHERE status = 'ACTIVE';
```

#### 5. ตาราง [`pattern_index`](../../design/database/sql_script/05_pattern_index.sql)
```sql
CREATE TABLE IF NOT EXISTS pattern_index (
    index_id      BIGSERIAL PRIMARY KEY,
    pattern_id    BIGINT NOT NULL REFERENCES patterns(pattern_id) ON DELETE CASCADE,
    position      INT NOT NULL,
    digit         CHAR(1) NULL,
    ticket_id     BIGINT NOT NULL REFERENCES tickets(ticket_id) ON DELETE CASCADE,

    CONSTRAINT uq_pattern_index UNIQUE (pattern_id, position, digit, ticket_id)
);

CREATE INDEX idx_pattern_index_pattern_pos_digit ON pattern_index (pattern_id, position, digit);
```

### 4.2 Entity-Relationship Diagram (ERD)

```mermaid
erDiagram
    users ||--o{ allocations : "makes"
    tickets ||--o{ allocations : "allocated_to"
    patterns ||--o{ pattern_index : "indexes"
    tickets ||--o{ pattern_index : "indexed_in"

    users {
        bigint user_id PK
        varchar name
        varchar email UK
        timestamp created_at
    }

    tickets {
        bigint ticket_id PK
        char(6) ticket_number UK
        boolean allocated
        timestamp allocated_at
    }

    allocations {
        bigserial allocation_id PK
        bigint user_id FK
        bigint ticket_id FK
        varchar pattern
        timestamp allocated_at
        timestamp expires_at
        varchar status
    }

    patterns {
        bigserial pattern_id PK
        varchar pattern_text UK
        timestamp created_at
    }

    pattern_index {
        bigserial index_id PK
        bigint pattern_id FK
        int position
        char(1) digit
        bigint ticket_id FK
    }
```

---

## 5. กลยุทธ์การทำ Index และอัลกอริทึมค้นหา Wildcard (Search Strategy & Query Routing)

Search Service ภายใน Go มี **Query Routing Planner** คอยวิเคราะห์รูปแบบความแน่นอนของตัวเลขในแต่ละตำแหน่ง (Selectivity):

```mermaid
flowchart TD
    InPattern([Search Pattern: 6 Chars]) --> CheckType{Analyze Positions}
    
    CheckType -->|6 Fixed Digits: 123456| CaseExact[Exact Point Lookup]
    CheckType -->|Prefix Match: 123***| CasePrefix[Range Index Scan]
    CheckType -->|Suffix Match: ****23| CaseSuffix[Suffix Index / Candidate Enumerate]
    CheckType -->|Mixed Wildcard: 1****5, 1*3*5*| CaseMixed[Pattern Index / Digit Filter]
    CheckType -->|All Wildcards: ******| CaseAll[Available Sample Pool]
    
    CaseExact --> Q_Exact["SELECT ticket_id, ticket_number FROM tickets\nWHERE ticket_number = '123456' AND allocated = false\nLatency: < 1ms"]
    CasePrefix --> Q_Prefix["SELECT ticket_id, ticket_number FROM tickets\nWHERE ticket_number >= '123000' AND ticket_number <= '123999'\n  AND allocated = false\nLatency: 1 - 5ms"]
    CaseSuffix --> Q_Suffix["Approach 1: Reverse String Index\nApproach 2: Enumerate 10,000 numbers IN (...)\nLatency: 5 - 15ms"]
    CaseMixed --> Q_Mixed["Query ผ่านตาราง pattern_index หรือ\nWHERE p1 = 1 AND p6 = 5\nLatency: 5 - 20ms"]
    CaseAll --> Q_All["SELECT ticket_id, ticket_number FROM tickets\nWHERE allocated = false LIMIT 50\nLatency: 1 - 3ms"]
```

---

## 6. กลไกจัดสรรสลากและป้องกันการชนกัน (Concurrency & Distribution Mechanism)

โจทย์สำคัญ: **"The same search pattern should not return the same ticket to multiple users at the same time"**

### 6.1 กลยุทธ์การล็อค 2 ระดับ (Two-Level Locking Strategy)

| มิติ | วิธีที่ 1: Redis Distributed Lock (ตาม Sequence Diagram) | วิธีที่ 2: PostgreSQL `FOR UPDATE SKIP LOCKED` (Optimized Production) |
|---|---|---|
| **หลักการทำงาน** | สั่งล็อค Key ระดับ Pattern บน Redis ชั่วคราวด้วย TTL เช่น `SET lock:pattern:1****5 <token> NX PX 2000` | สั่งล็อคระดับแถวของสลากที่เข้าข่ายใน Database Transaction โดยตรง |
| **พฤติกรรม User B** | ต้องรอให้ User A จัดสรรเสร็จและปล่อย Lock ก่อน (Serialized Queue) | **ไม่ต้องรอคิว (Non-blocking)** ระบบจะข้ามสลากที่ User A ล็อคอยู่ไปหยิบสลากใบถัดไปทันที |
| **Throughput** | ปานกลาง เหมาะกับกรณีที่ต้องการจำกัดโควตาต่อ Pattern อย่างเข้มงวด | **สูงมาก** สามารถประมวลผลคำขอหลายพันรายการต่อวินาทีพร้อมกันได้ |
| **Fault Tolerance** | ต้องตั้ง TTL เผื่อกรณี Process ตายกลางคัน | ปลอดภัย 100% หาก Application Crash ระบบ Database จะทำ Auto-Rollback ทันที |

### 6.2 คำสั่ง Atomic Allocation ด้วย `FOR UPDATE SKIP LOCKED`
```sql
BEGIN;

WITH selected AS (
    SELECT ticket_id 
    FROM tickets
    WHERE allocated = false
      AND ticket_number >= '123000' AND ticket_number <= '123999'
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE tickets t
SET allocated = true,
    allocated_at = CURRENT_TIMESTAMP
FROM selected s
WHERE t.ticket_id = s.ticket_id
RETURNING t.ticket_id, t.ticket_number;

INSERT INTO allocations (user_id, ticket_id, pattern, status, expires_at)
VALUES ($user_id, $ticket_id, '123***', 'ACTIVE', CURRENT_TIMESTAMP + INTERVAL '5 MINUTES');

COMMIT;
```

---

## 7. ลำดับขั้นตอนการทำงาน (Sequence Diagram: Concurrent Allocation Flow)

อ้างอิงลำดับขั้นตอน 13 Step จาก [`sequence_diagram.png`](../../design/sequence_diagram/sequence_diagram.png) และ [`flow.md`](../../design/sequence_diagram/flow.md):

```mermaid
sequenceDiagram
    actor UserA as User A (Client)
    actor UserB as User B (Client)
    participant API as API Gateway / Search Service
    participant DB as Database (Lottery DB)
    participant Redis as Redis (Distributed Lock)

    UserA->>API: 1. Search pattern: 1****5
    UserB->>API: 2. Search pattern: 1****5

    critical Concurrency Control (User A)
        API->>Redis: 3. Acquire lock (pattern: 1****5)
        Redis-->>API: 4. Lock acquired (TTL)

        API->>DB: 5. Query available tickets (using index)
        DB-->>DB: 6. Find matching tickets (not allocated)
        DB-->>API: 7. Return candidate ticket IDs

        API->>DB: 8. Reserve ticket (atomic update: set allocated = true)
        DB-->>API: 9. Success (ticket ID)

        API->>Redis: 10. Release lock
        Redis-->>API: 11. Lock released
    end

    API-->>UserA: 12. Return ticket to User A
    API-->>UserB: 13. Return next available ticket to User B
```

---

## 8. ข้อกำหนด API Interface และวงจรการทำงาน (API Specification & Lifecycle)

การจัดสรรสลากทำงานภายใต้วงจร:
$$\text{FIND (ค้นหาสลากที่ว่าง)} \longrightarrow \text{LOCK (ล็อคแถวสลาก)} \longrightarrow \text{RESERVE (บันทึกการจอง)} \longrightarrow \text{RETURN (ส่งผลลัพธ์)}$$

### ตัวอย่าง Interface:
- **HTTP Method & Path**: `POST /lottery/search` (หรือ `POST /lottery/allocate`)
- **Request Body**:
  ```json
  {
    "user_id": 1001,
    "pattern": "1****5",
    "quantity": 1
  }
  ```
- **Response Body (200 OK)**:
  ```json
  {
    "reservation_id": 900001,
    "ticket_id": 500006,
    "lottery_number": "123455",
    "pattern": "1****5",
    "status": "ACTIVE",
    "allocated_at": "2026-09-14T16:00:00Z",
    "expires_at": "2026-09-14T16:05:00Z",
    "expires_in_seconds": 300
  }
  ```

---

## 9. การเปรียบเทียบเทคโนโลยีจัดเก็บข้อมูล (Technology Evaluation & Selection)

อ้างอิงการวิเคราะห์จาก [`tech-stack.md`](../../design/database/tech-stack.md):

| ปัจจัยการพิจารณา | PostgreSQL 16+ *(Primary แนะนำ)* | Redis 7+ *(Cache / Lock)* | Elasticsearch *(Alternative)* |
|---|:---:|:---:|:---:|
| **ความถูกต้องของข้อมูล (Consistency)** | **Strong Consistency (ACID)** | Eventual / Memory | Eventual Consistency (มี Delay 1s) |
| **กลไกการล็อค (Concurrency Lock)** | **Row-level `SKIP LOCKED` ในตัว** | Redis SETNX / Redlock | ไม่มี Row-level Lock สำหรับตัดสต็อก |
| **ความเร็วในการค้นหา (Query Latency)** | **1 - 15 ms** (Index ใน RAM) | **< 1 ms** (In-Memory) | **5 - 20 ms** (Inverted Index) |
| **การใช้หน่วยความจำ (RAM)** | ประหยัด (~2GB Cached in RAM) | สูงมาก (In-Memory 100%) | สูงมาก (JVM Heap + OS Cache) |
| **ความซับซ้อนในการดูแล (Ops Complexity)** | **ต่ำ - ปานกลาง** | ต่ำ | สูง (Cluster, Shards, Rebalance) |
| **ข้อสรุปบทบาทในระบบ** | **Primary Database (Source of Truth)** | **Auxiliary Cache / Lock** | **ไม่จำเป็นสำหรับตัวเลข 6 หลัก** |

---

## 10. การวิเคราะห์ประสิทธิภาพและการรองรับโหลดสูง (Performance, Scaling & KPIs)

### 10.1 Memory Footprint (10 ล้านรายการ)
- สลาก 10,000,000 แถว ใน PostgreSQL ใช้พื้นที่ข้อมูลประมาณ **800 MB - 1.2 GB**
- ดัชนี `idx_tickets_allocated_number` ใช้พื้นที่ประมาณ **500 MB - 800 MB**
- **รวมทั้งหมด ~2.0 GB**: สามารถเก็บไว้ใน RAM ของเครื่องเซิร์ฟเวอร์มาตรฐาน (RAM 16GB - 32GB) ได้ทั้งหมด ทำให้ Query ทุกคำสั่งรันบน RAM โดยไม่ต้องรอ Disk I/O

### 10.2 Database Monitoring KPIs

| ตัวชี้วัด (Metrics) | เกณฑ์ที่ยอมรับได้ | ความสำคัญต่องาน Allocation |
|---|:---:|---|
| **Lock Wait Time** | **< 5 ms** | สำคัญที่สุด: ตรวจสอบว่ามี Request ติดคิวรอนานเกินไปหรือไม่ |
| **Deadlock Count** | **0** | ต้องเป็นศูนย์เสมอด้วยการใช้ `SKIP LOCKED` |
| **Index Hit Ratio** | **> 99%** | การันตีว่าคำขอค้นหาวิ่งผ่าน Memory Buffer 100% |
| **Connection Pool Usage** | **< 80%** | ป้องกันปัญหาสายเชื่อมต่อเต็มระหว่าง Traffic Surge |

---

## 11. ความน่าเชื่อถือและการรับมือความล้มเหลว (Failure Handling & Recovery)

```mermaid
flowchart TD
    TX_Start[1. BEGIN TRANSACTION] --> Lock_Row[2. LOCK TICKET ROW & SET allocated = true]
    Lock_Row --> Insert_Alloc[3. INSERT INTO allocations]
    
    Insert_Alloc --> Check_Crash{Application Crashes?}
    
    Check_Crash -->|Crash ก่อน Commit| Rollback[PostgreSQL Auto Rollback\n- ล็อคถูกปลดทันที\n- สลากกลับเป็น AVAILABLE]
    Check_Crash -->|Crash หลัง Commit| Persisted[ข้อมูลบันทึกสำเร็จ\n- Status = ACTIVE\n- มี expires_at คุ้มครอง]
    
    Persisted --> Check_Payment{User ชำระเงินสำเร็จ?}
    Check_Payment -->|ชำระเงินสำเร็จ| Completed[Update Status = 'COMPLETED'\nสลากถูกขายถาวร]
    Check_Payment -->|ไม่ชำระเงินใน 5 นาที| Reaper[Expiry Reaper Worker\n- Set allocations.status = 'EXPIRED'\n- Set tickets.allocated = false]
```

---

## 12. สรุปผลการออกแบบตามเกณฑ์การประเมิน (Evaluation Criteria Alignment)

| เกณฑ์การประเมิน (Criteria) | วิธีการตอบโจทย์ของสถาปัตยกรรมนี้ |
|---|---|
| **1. Feasibility** | สถาปัตยกรรมครอบคลุมครบถ้วนทั้งชุดข้อมูล 10M, การค้นหา Wildcard, และการกระจายสลากโดยไม่ซ้ำกัน |
| **2. Performance** | วิเคราะห์ขนาดข้อมูลชัดเจน (2GB Fit ใน RAM 100%) เลือกใช้ B-Tree Range Scans และ Partial Indexes ทำ Response Time < 50ms |
| **3. Correctness** | ใช้กลไก Pessimistic Row Lock (`FOR UPDATE SKIP LOCKED`) หรือ Redis Distributed Lock ขจัดปัญหา Duplicate Selection 100% |
| **4. Real-world Practicality** | เลือกใช้ Go + PostgreSQL ซึ่งเป็นเทคโนโลยีระดับองค์กร ดูแลรักษาง่าย มี Tooling และ High Availability ครบถ้วน |
| **5. Creativity** | การค้นพบคุณสมบัติ Bounded Domain (1M Combinations ใน 10M Tickets) และการใช้ Non-blocking Concurrent Lock ข้ามแถวอัตโนมัติ |

---

## 13. เอกสารและไฟล์สคริปต์ที่เกี่ยวข้อง (Cross-References & Design Artifacts)

- 📘 **[Tech Stack & Database Rationale](../../design/database/tech-stack.md)**: รายละเอียดการวิเคราะห์เปรียบเทียบเทคโนโลยี
- 🧭 **[Sequence Flow & Architecture Guide](../../design/sequence_diagram/flow.md)**: คำอธิบายลำดับการทำงานและสถาปัตยกรรม
- 📐 **[Database Schema Image](../../design/database/schema/table_5.png)**: แผนผังโครงสร้างตารางข้อมูล 5 ตาราง
- 📊 **[ER Diagram Image](../../design/database/er_diagram/er_5.png)**: แผนผังความสัมพันธ์ Entity
- 🖼️ **[Sequence Diagram Image](../../design/sequence_diagram/sequence_diagram.png)**: แผนภาพ Sequence Flow 13 ขั้นตอน
- 💻 **[PlantUML Sequence Source](../../design/sequence_diagram/sequence_diagram.puml)**: โค้ด PlantUML ไดอะแกรม
- 🗄️ **[SQL DDL Scripts Directory](../../design/database/sql_script/README.md)**:
  - [`00_init_db_lottery.sql`](../../design/database/sql_script/00_init_db_lottery.sql) *(Database Bootstrap Script — สร้าง DB `lottery_system` + รันทุกตารางและ Indexes ครบถ้วน)*
  - [`00_init_schema.sql`](../../design/database/sql_script/00_init_schema.sql) *(Master DDL Runner — รวมคำสั่ง `\i` รันไฟล์แยก)*
  - [`01_users.sql`](../../design/database/sql_script/01_users.sql) *(Table: users)*
  - [`02_tickets.sql`](../../design/database/sql_script/02_tickets.sql) *(Table: tickets)*
  - [`03_patterns.sql`](../../design/database/sql_script/03_patterns.sql) *(Table: patterns)*
  - [`04_allocations.sql`](../../design/database/sql_script/04_allocations.sql) *(Table: allocations)*
  - [`05_pattern_index.sql`](../../design/database/sql_script/05_pattern_index.sql) *(Table: pattern_index)*

