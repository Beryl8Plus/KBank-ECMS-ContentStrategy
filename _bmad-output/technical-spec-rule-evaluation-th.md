# เอกสารข้อกำหนดทางเทคนิค: การประมวลผลกฎการตัดสินใจในหน่วยความจำแคช (Technical Specification: Decision Rule Evaluation in Cache Runtime)

**สถานะ:** ฉบับร่าง (Draft)
**วันที่:** 8 เมษายน 2569
**ระบบ:** KBank ECMS Backend
**ขอบเขต:** การประมวลผลกฎ (Evaluation) และการจัดเก็บแคช (Caching) สำหรับ Orchestrator

---

## 1. บทนำ (Introduction)

เอกสารฉบับนี้กำหนดรายละเอียดทางเทคนิคสำหรับการออกแบบและพัฒนาระบบการประมวลผลกฎการตัดสินใจ (Decision Rule Evaluation) และการส่งมอบเนื้อหา (Content Delivery) โดยใช้เทคนิคการประมวลผลแบบเบื้องหลัง (Background Processing) และการจัดเก็บข้อมูลในหน่วยความจำแคช (Redis) เพื่อตอบสนองความต้องการด้านประสิทธิภาพ (Latency < 200ms) และความยืดหยุ่นในการจัดการแคมเปญ

## 2. สถาปัตยกรรมระบบ (System Architecture)

ระบบประกอบด้วย 2 ส่วนหลักที่ทำงานแยกกันอย่างอิสระ (Decoupled):

### 2.1 cms-runtime (ตัวประมวลผลกฎเบื้องหลัง)

- **หน้าที่:** ดำเนินการคำนวณคะแนนตามกฎการตัดสินใจ (Decision Rules) และจัดเก็บผลลัพธ์ลงใน Redis
- **กลไกการทำงาน:**
  - **Periodic Evaluation:** ทำงานตามรอบเวลา (Ticker) ทุกๆ 5 นาที (ค่าเริ่มต้น)
  - **Reactive Evaluation:** ทำงานทันทีเมื่อมีการสร้างหรือแก้ไขตารางเวลา (Schedule) ผ่าน `EvaluatePlacement`
- **อินเทอร์เฟซหลัก:** `RuntimeService`

### 2.2 cms-delivery-service (บริการส่งมอบเนื้อหา)

- **หน้าที่:** รับคำขอจาก Orchestrator และดึงข้อมูลจาก Redis ทันที
- **กลไกการทำงาน:** เน้นความเร็วสูงสุด โดยจะไม่อ่านข้อมูลจากฐานข้อมูลหลัก (PostgreSQL) ในขั้นตอนการดึงเนื้อหา (Cache-Only Delivery)
- **อินเทอร์เฟซหลัก:** `DeliveryService`

## 3. รายละเอียดการประมวลผล (Evaluation Logic)

### 3.1 การเลือกข้อมูล (Selection Logic)

ระบบจะเลือกเฉพาะตารางเวลา (Schedule) ที่ตรงตามเงื่อนไขดังนี้:

1.  สถานะการใช้งานเป็นเปิด (is_active = true)
2.  ไม่อยู่ในสถานะถูกลบ (deleted_at IS NULL)
3.  อยู่ในช่วงเวลาที่มีผล (effective_from <= NOW AND effective_until > NOW)
    _อ้างอิงฟังก์ชัน:_ `ListActiveSchedulesInWindow(ctx, at)`

### 3.2 รูปแบบการคำนวณคะแนน (Evaluation Strategies)

ใช้ **Strategy Pattern** ในการคำนวณตามประเภทของกฎ (Rule Type):

- **SCORING:** ให้คะแนนตามที่ระบุในฟิลด์ `score` ของกฎ
- **SEGMENT:** ตรวจสอบความสอดคล้องกับกลุ่มเป้าหมาย (User Segment)
- **ELIGIBLE:** ตรวจสอบคุณสมบัติหรือสิทธิ์ในการได้รับเนื้อหา (Pass/Fail)

### 3.3 การจัดการผลลัพธ์ (Result Processing)

1.  **การลบข้อมูลซ้ำ (Deduplication):** หากมีหลายกฎที่ชี้ไปยัง `content_path` (เส้นทางเนื้อหา) เดียวกัน ระบบจะเลือกเฉพาะรายการที่มีคะแนนสูงสุดเพียงรายการเดียว
2.  **การจัดลำดับ (Ranking):** เรียงลำดับตามคะแนน (Score) จากมากไปน้อย
3.  **ข้อจำกัดจำนวน (Top N):** เลือกเฉพาะจำนวนสูงสุดตามที่กำหนดในฟิลด์ `max_results` ของแต่ละตำแหน่ง (Placement)

## 4. การจัดการหน่วยความจำแคช (Cache Strategy)

### 4.1 รูปแบบการจัดเก็บ (Storage Format)

- **Redis Key:** `placement:{placement_name}` (เช่น `placement:wsaHomeBanner`)
- **Data Type:** JSON String (Array of Scored Content)
- **ตัวอย่างข้อมูล:**
  ```json
  [
    {
      "contentPath": "/content/kbank/homepage/banner-promo",
      "score": 95.5,
      "ruleId": "...",
      "ruleType": "SCORING",
      "evaluatedAt": "2026-04-08T10:00:00Z"
    }
  ]
  ```

### 4.2 การปรับปรุงและล้างข้อมูล (Invalidation)

1.  **TTL (Time-To-Live):** กำหนดให้ข้อมูลในแคชหมดอายุโดยอัตโนมัติ (เช่น 5 นาที)
2.  **Selective Flush:** สามารถล้างแคชเฉพาะตำแหน่งที่ต้องการได้ผ่านฟังก์ชัน `Delete(ctx, key)`
3.  **Full Flush:** ล้างข้อมูลแคชทั้งหมดในกรณีฉุกเฉินผ่าน `FlushDB(ctx)`

## 5. รายละเอียดฟิลด์ข้อมูลที่เกี่ยวข้อง (Data Field Details)

| ชื่อฟิลด์ (Technical Name)    | ชื่อภาษาไทย (Thai Name) | คำอธิบาย (Thai Description)                       |
| :---------------------------- | :---------------------- | :------------------------------------------------ |
| **decision_rules.score**      | คะแนนรวมหรือน้ำหนัก     | น้ำหนักที่ใช้ในการตัดสินใจเลือกเนื้อหา            |
| **placements.name**           | ชื่อตำแหน่งการวาง       | ชื่อเรียกตำแหน่งแสดงผล เช่น wsaHomeBanner         |
| **placements.max_results**    | จำนวนผลลัพธ์สูงสุด      | จำนวนกฎสูงสุดที่จะถูกจัดเก็บในแคชสำหรับตำแหน่งนี้ |
| **schedules.effective_from**  | วันและเวลาเริ่มต้น      | จุดเริ่มต้นที่กฎนี้จะมีผลในตำแหน่งที่กำหนด        |
| **schedules.effective_until** | วันและเวลาสิ้นสุด       | จุดสิ้นสุดที่กฎนี้จะหมดวาระการทำงาน               |

## 6. ข้อกำหนดด้านประสิทธิภาพ (Performance SLAs)

- **API Response Time:** < 200ms สำหรับ `GET /delivery`
- **Cache Refresh Rate:** ข้อมูลใน Redis ต้องไม่ล้าหลังเกิน 5 นาที สำหรับรอบการประมวลผลปกติ
- **Reactive Latency:** ทันทีที่มีการเปลี่ยนแปลงตารางเวลา แคชของตำแหน่งนั้นควรได้รับการปรับปรุงภายใน < 1 วินาที

---

_จัดทำโดย: Gemini CLI (Senior Software Engineer)_
_อ้างอิงเอกสาร: docs/diagram/data_dictionary_th.md, internal/domain/entity/placement.go_
