# Technical Specification: Rule Conditions Key Normalization

## 1. บทนำ (Introduction)
เอกสารฉบับนี้กำหนดมาตรฐานการทำ Key Normalization สำหรับเงื่อนไขการตัดสินใจ (`rule_conditions`) เพื่อใช้ในการจัดเก็บข้อมูลใน Redis โดยมีวัตถุประสงค์เพื่อเพิ่มประสิทธิภาพในการเข้าถึงข้อมูล (Caching) และลดความซ้ำซ้อนของ Logic (Deduplication)

## 2. วัตถุประสงค์ (Objectives)
1. เพื่อสร้าง Unique Identifier (Hash) ที่เป็นตัวแทนของตรรกะ (Logic) ของเงื่อนไขที่เหมือนกัน
2. เพื่อลดภาระการ Query Database (`rule_conditions` table) ในช่วง Runtime
3. เพื่อรองรับการทำ Evaluation Result Reuse ในอนาคต

## 3. กระบวนการทำ Normalization (Normalization Logic)
การสร้าง Normalized Key จะต้องเป็นแบบ Deterministic (อินพุตเดิมได้ผลลัพธ์เดิมเสมอ) โดยมีขั้นตอนดังนี้:

### 3.1 การจัดลำดับ (Canonical Sorting)
- เงื่อนไขในระดับเดียวกัน (Siblings) ต้องถูกเรียงลำดับตาม `sequence` จากน้อยไปมาก
- หาก `sequence` เท่ากัน ให้ใช้ `id` (UUID) เป็นตัวตัดสิน

### 3.2 การสร้าง Canonical String (String Representation)
แปลงโครงสร้าง Tree ของเงื่อนไขให้เป็นข้อความตามรูปแบบดังนี้:
- **Leaf Node:** `{AttributeID}:{LogicalOperator}:{Value}`
- **Nested Node:** `({Child1} {Connector} {Child2} ...)`
- **Value Normalization:** ค่าในฟิลด์ `value` (JSON) ต้องถูก Minify และ Sort Key (ในกรณีที่เป็น Object) ก่อนนำมาต่อ String

### 3.3 การทำ Hashing
- นำ Canonical String ที่ได้มาทำ `SHA-256`
- ผลลัพธ์ที่ได้จะเป็น Hexadecimal String ความยาว 64 ตัวอักษร

## 4. มาตรฐานการเก็บข้อมูลใน Redis (Redis Schema)
| ประเภทข้อมูล | รูปแบบ Key | Data Structure | TTL |
| :--- | :--- | :--- | :--- |
| **Logic Structure** | `cms:rule_logic:v1:{sha256}` | String (JSON) | 24 Hours |
| **Evaluation Result**| `cms:eval:v1:{user_id}:{sha256}` | String (Boolean) | 5-15 Mins |
