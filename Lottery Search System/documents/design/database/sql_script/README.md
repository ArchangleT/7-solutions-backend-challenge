# Database SQL Scripts — Lottery Search System

โฟลเดอร์นี้รวบรวมไฟล์ DDL `.sql` สำหรับสร้างตารางบน **PostgreSQL** โดยอ้างอิงโครงสร้างจากภาพ [table_5.png](../schema/table_5.png)

---

## รายการไฟล์ DDL ตามลำดับ Dependency

| ลำดับ | ไฟล์ | ชื่อตาราง | คำอธิบาย |
|:---:|---|---|---|
| **00** | [`00_init_schema.sql`](./00_init_schema.sql) | *(Master Script)* | รวมคำสั่งรันทุกตารางตามลำดับ |
| **01** | [`01_users.sql`](./01_users.sql) | `users` | ข้อมูลผู้ใช้งานในระบบ |
| **02** | [`02_tickets.sql`](./02_tickets.sql) | `tickets` | ข้อมูลสลาก 10 ล้านใบ (Inventory) |
| **03** | [`03_patterns.sql`](./03_patterns.sql) | `patterns` | ข้อความ Search Pattern ที่ไม่ซ้ำกัน |
| **04** | [`04_allocations.sql`](./04_allocations.sql) | `allocations` | บันทึกประวัติและสถานะการจัดสรร/จองสลาก |
| **05** | [`05_pattern_index.sql`](./05_pattern_index.sql) | `pattern_index` | Inverted Index จับคู่ Pattern กับ Ticket |

---

## วิธีการรันสคริปต์ (Execution)

### รันทุกไฟล์ผ่าน Master Script:
```bash
psql -U <username> -d <dbname> -f 00_init_schema.sql
```

### หรือรันทีละไฟล์ตามลำดับ:
```bash
psql -U <username> -d <dbname> -f 01_users.sql
psql -U <username> -d <dbname> -f 02_tickets.sql
psql -U <username> -d <dbname> -f 03_patterns.sql
psql -U <username> -d <dbname> -f 04_allocations.sql
psql -U <username> -d <dbname> -f 05_pattern_index.sql
```

---

## สรุป Constraints และ Indexes ที่แนะนำ (ตาม Schema)

- **`tickets`**:
  - `CONSTRAINT uq_tickets_ticket_number UNIQUE (ticket_number)`
  - `INDEX idx_tickets_allocated_number ON tickets (allocated, ticket_number)`
- **`patterns`**:
  - `CONSTRAINT uq_patterns_pattern_text UNIQUE (pattern_text)`
  - `INDEX idx_patterns_pattern_text ON patterns (pattern_text)`
- **`allocations`**:
  - `FK users(user_id)`
  - `FK tickets(ticket_id)`
  - `INDEX idx_allocations_user_allocated_at ON allocations (user_id, allocated_at)`
  - `INDEX idx_allocations_active_expires_at ON allocations (expires_at) WHERE status = 'ACTIVE'`
- **`pattern_index`**:
  - `FK patterns(pattern_id)`
  - `FK tickets(ticket_id)`
  - `CONSTRAINT uq_pattern_index UNIQUE (pattern_id, position, digit, ticket_id)`
  - `INDEX idx_pattern_index_pattern_pos_digit ON pattern_index (pattern_id, position, digit)`
