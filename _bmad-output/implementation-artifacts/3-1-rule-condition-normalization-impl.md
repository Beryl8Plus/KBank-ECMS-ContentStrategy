# Implementation Plan: Rule Condition Normalization

## 1. งานที่ต้องดำเนินการ (Tasks)

### Task 1: เพิ่มฟังก์ชัน Normalization ใน Domain Service

- **ไฟล์:** `internal/service/evaluator/condition_normalization.go` (สร้างใหม่)
- **หน้าที่:**
  - สร้างฟังก์ชัน `GenerateConditionHash(conds []entity.RuleCondition) (string, error)`
  - พัฒนา Recursive Function `buildCanonicalString` เพื่อแปลง Tree เป็น String
  - ใช้ `crypto/sha256` ในการสร้าง Hash

### Task 2: แก้ไขโครงสร้างการโหลดข้อมูลใน CMS Runtime

- **ไฟล์:** `internal/service/cms_runtime_service.go`
- **หน้าที่:**
  - ตรวจสอบความมีอยู่ของ Logic ใน Redis ก่อนเริ่มประมวลผล (Cache-Aside Pattern)
  - หากไม่มีใน Cache ให้โหลดจาก DB และทำ `Set` ลง Redis ด้วย Normalized Key

### Task 3: Unit Testing

- **ไฟล์:** `internal/service/evaluator/condition_normalization_test.go`
- **Case ที่ต้องทดสอบ:**
  - สลับลำดับเงื่อนไขใน Input (แต่ Logic เหมือนเดิม) ต้องได้ Hash เดิม
  - เปลี่ยนค่าเล็กน้อยใน Value ต้องได้ Hash ต่างจากเดิม
  - ตรวจสอบความถูกต้องของ Nested Logic (AND/OR)

## 2. ลำดับการทำงาน (Step-by-Step)

1.  **Phase 1: Logic Core**
    - Implement `compactJSON` helper เพื่อจัดการฟิลด์ `datatypes.JSON`
    - Implement `buildCanonicalString` และทดสอบด้วยข้อมูลจำลอง

2.  **Phase 2: Integration**
    - เชื่อมต่อกับ `cacheRepo` ใน `CMSRuntimeService`
    - ปรับปรุง `EvaluateAll` ให้บันทึก Logic สรุปลง Redis

3.  **Phase 3: Verification**
    - รัน `go test ./internal/service/evaluator/...`
    - ตรวจสอบ Key ใน Redis ด้วย `redis-cli keys "cms:rule_logic:*"`

## 3. นิยามความสำเร็จ (Definition of Done)

- [ ] ฟังก์ชันสามารถสร้าง Hash ที่ไม่ซ้ำกันสำหรับ Logic ที่ต่างกัน
- [ ] ลำดับของ Input ไม่มีผลต่อ Hash ที่ได้ (Stable Result)
- [ ] มี Unit Test ครอบคลุมเคสพื้นฐานและเคสซับซ้อน
- [ ] ข้อมูล Logic ถูกจัดเก็บใน Redis สำเร็จเมื่อมีการรัน Runtime Service
