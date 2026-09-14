# Path 2: Lottery Search System (Design Proposal เท่านั้น)

ออกแบบระบบค้นหา lottery ticket (6 หลัก) ขนาด **10 ล้านรายการ** ด้วย wildcard pattern matching

### งานหลัก (design only, ไม่มีโค้ด)
- **Data Volume**: 10M tickets, แต่ละตัวเป็นเลข 6 หลัก
- **Search Pattern**: 6 ตัวอักษร ประกอบด้วย digit + `*` เช่น `****23`, `1****5`, `123***`
- **Result Distribution**: ป้องกันไม่ให้ ticket เดียวถูก return ให้หลาย user พร้อมกัน
- **Performance**: ค้นหาได้เร็วที่ 10M+ records
- **Real-World Design**: แนะนำ database/storage, algorithm/indexing strategy, กลไกป้องกัน duplicate (locking / reservation / atomic allocation)

### Deliverables (เอกสารอย่างเดียว)
- Solution architecture, data structures, algorithms
- Production database/storage ที่แนะนำพร้อมเหตุผล
- Performance analysis (efficiency + tradeoffs)
- Concurrency/distribution strategy
