# Data Dictionary for ECMS (พจนานุกรมข้อมูลสำหรับระบบ ECMS)

เอกสารฉับนี้รวบรวมรายละเอียดของตารางและฟิลด์ต่างๆ ตาม GORM models ใน `internal/domain/entity` โดยใช้ภาษาไทยเป็นมาตรฐานสำหรับการอธิบายข้อมูล

---

## 1. การจัดการสิทธิ์และการเข้าใช้งาน (Management & Auth Domain)

### ตาราง: login_token_histories

ใช้สำหรับเก็บประวัติการเข้าใช้งานและข้อมูล Token ของผู้ใช้งาน

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                  |
| :------------ | :------------------ | :-------------------------------------- |
| id            | uuid                | ไอดีหลัก (Primary Key) ของรายการประวัติ |
| username      | varchar(255)        | ชื่อผู้ใช้งานที่ใช้เข้าสู่ระบบ (Unique) |
| access_token  | varchar(255)        | รหัสโทเค็นที่ใช้สำหรับการเข้าถึงระบบ    |
| expire_date   | timestamptz         | วันและเวลาที่โทเค็นหมดอายุ              |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างรายการ             |
| created_by    | uuid                | ไอดีผู้สร้างรายการ (FK → users.id)      |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด             |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด (FK → users.id)      |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)  |

### ตาราง: users

ตารางข้อมูลผู้ใช้งานหลักในระบบ

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                            |
| :------------ | :------------------ | :------------------------------------------------ |
| id            | uuid                | ไอดีหลักของผู้ใช้งาน                              |
| role_id       | uuid                | ไอดีบทบาทของผู้ใช้งาน (FK → roles.id)             |
| profile_id    | uuid                | ไอดีโปรไฟล์ที่กำหนดกลุ่มสิทธิ์ (FK → profiles.id) |
| email         | varchar(255)        | ที่อยู่อีเมลของผู้ใช้งาน (Unique)                 |
| name_th       | varchar(255)        | ชื่อ-นามสกุล ภาษาไทย                              |
| name_en       | varchar(255)        | ชื่อ-นามสกุล ภาษาอังกฤษ                           |
| is_active     | boolean             | สถานะการใช้งาน (true = ใช้งาน, false = ระงับ)     |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างข้อมูล                       |
| created_by    | uuid                | ไอดีผู้สร้างข้อมูล (FK → users.id)                |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด                       |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด (FK → users.id)                |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)            |

### ตาราง: roles

ตารางกำหนดบทบาทหน้าที่ของกลุ่มผู้ใช้งาน

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                 |
| :------------ | :------------------ | :------------------------------------- |
| id            | uuid                | ไอดีหลักของบทบาท                       |
| name          | varchar(255)        | ชื่อของบทบาท                           |
| code          | varchar(255)        | รหัสอ้างอิงของบทบาท (Unique)           |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างข้อมูล            |
| created_by    | uuid                | ไอดีผู้สร้างข้อมูล                     |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด            |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด                     |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete) |

### ตาราง: profiles

ตารางกำหนดกลุ่มข้อมูลหรือชุดของสิทธิ์ (Profiles)

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                 |
| :------------ | :------------------ | :------------------------------------- |
| id            | uuid                | ไอดีหลักของโปรไฟล์                     |
| name          | varchar(255)        | ชื่อของโปรไฟล์                         |
| code          | varchar(255)        | รหัสอ้างอิงของโปรไฟล์ (Unique)         |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างข้อมูล            |
| created_by    | uuid                | ไอดีผู้สร้างข้อมูล                     |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด            |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด                     |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete) |

### ตาราง: permissions (Permissions Entity)

ตารางกำหนดสิทธิ์การเข้าถึงและการกระทำภายในระบบ

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                                                  |
| :------------ | :------------------ | :---------------------------------------------------------------------- |
| id            | uuid                | ไอดีหลักของสิทธิ์                                                       |
| name          | varchar(255)        | ชื่อที่ใช้เรียกสิทธิ์                                                   |
| source        | varchar(255)        | แหล่งที่มาหรือโมดูลที่สิทธิ์นี้ใช้ (เช่น CONTENT_DECISION_RULE)         |
| action        | varchar(255)        | การกระทำที่ได้รับอนุญาต (เช่น CREATE, EDIT, DELETE, VIEW_ALL, EDIT_ALL) |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างข้อมูล                                             |
| created_by    | uuid                | ไอดีผู้สร้างข้อมูล                                                      |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด                                             |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด                                                      |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)                                  |

### ตาราง: profile_permissions

ตารางเชื่อมโยงระหว่างโปรไฟล์และสิทธิ์การใช้งาน (Relation Table)

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                 |
| :------------ | :------------------ | :------------------------------------- |
| id            | uuid                | ไอดีหลักของรายการเชื่อมโยง             |
| profile_id    | uuid                | ไอดีโปรไฟล์ (FK → profiles.id)         |
| permission_id | uuid                | ไอดีสิทธิ์ (FK → permissions.id)       |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างรายการ            |
| created_by    | uuid                | ไอดีผู้สร้างรายการ                     |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด            |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด                     |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete) |

---

## 2. ระบบหลักของกฎการตัดสินใจ (Decision Rule Domain)

### ตาราง: decision_rules

ตารางหลักที่เก็บข้อมูลกฎการตัดสินใจ (Decision Rule Entity)

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                          |
| :------------ | :------------------ | :---------------------------------------------- |
| id            | uuid                | ไอดีหลักของกฎการตัดสินใจ                        |
| name          | varchar(255)        | ชื่อของกฎการตัดสินใจ                            |
| type          | enum                | ประเภทของกฎ (กำหนดตาม Enum ในระบบ)              |
| content_path  | varchar(255)        | เส้นทางที่เก็บไฟล์หรือข้อมูลเนื้อหาของกฎ        |
| score         | integer             | คะแนนรวมหรือน้ำหนักของกฎ                        |
| status        | enum                | สถานะของกฎ (DRAFT, ACTIVE, INACTIVE)            |
| inactive_by   | uuid                | ไอดีผู้ที่ทำการระงับการใช้งานกฎ (FK → users.id) |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างกฎ                         |
| created_by    | uuid                | ไอดีผู้สร้างกฎ (FK → users.id)                  |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด                     |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด (FK → users.id)              |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)          |

### ตาราง: rules

ตารางเก็บข้อมูลรูปแบบย่อยหรือตัวแปรของกฎ (Rule Variation Entity)

| ฟิลด์ (Field)    | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                     |
| :--------------- | :------------------ | :----------------------------------------- |
| id               | uuid                | ไอดีหลักของรูปแบบกฎย่อย                    |
| decision_rule_id | uuid                | ไอดีอ้างอิงกฎหลัก (FK → decision_rules.id) |
| variation_name   | varchar(255)        | ชื่อเรียกรูปแบบหรือตัวแปรของกฎ             |
| score            | integer             | คะแนนเฉพาะสำหรับรูปแบบนี้                  |
| order_no         | integer             | ลำดับการประมวลผล (เรียงจากน้อยไปมาก)       |
| created_at       | timestamptz         | วันที่และเวลาที่สร้างข้อมูล                |
| created_by       | uuid                | ไอดีผู้สร้างข้อมูล                         |
| updated_at       | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด                |
| updated_by       | uuid                | ไอดีผู้แก้ไขล่าสุด                         |
| deleted_at       | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)     |

---

## 3. เครื่องมือจัดการเงื่อนไข (Condition Engine Domain)

### ตาราง: rule_conditions

ตารางเก็บเงื่อนไขที่ใช้ในการตัดสินใจ (Rule Condition Entity)

| ฟิลด์ (Field)            | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                                    |
| :----------------------- | :------------------ | :-------------------------------------------------------- |
| id                       | uuid                | ไอดีหลักของเงื่อนไข                                       |
| sequence                 | integer             | ลำดับของเงื่อนไขภายในกลุ่ม                                |
| decision_rule_id         | uuid                | ไอดีอ้างอิงกฎหลัก (FK → decision_rules.id)                |
| rule_id                  | uuid                | ไอดีอ้างอิงกฎย่อย (FK → rules.id) (Nullable)              |
| parent_rule_condition_id | uuid                | ไอดีเงื่อนไขแม่ สำหรับการทำเงื่อนไขแบบซ้อน (Recursive FK) |
| attribute_id             | uuid                | ไอดีแอตทริบิวต์ที่ใช้ตรวจสอบ (FK → attributes.id)         |
| logical_operator         | enum                | ตัวดำเนินการเปรียบเทียบ (เช่น LT, GT, EQ, IN, BETWEEN)    |
| value                    | jsonb               | ค่าที่ใช้ในการเปรียบเทียบ (จัดเก็บในรูปแบบ JSON)          |
| connector_operator       | enum                | ตัวเชื่อมความสัมพันธ์กับเงื่อนไขถัดไป (AND, OR)           |
| created_at               | timestamptz         | วันที่และเวลาที่สร้างเงื่อนไข                             |
| created_by               | uuid                | ไอดีผู้สร้างเงื่อนไข                                      |
| updated_at               | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด                               |
| updated_by               | uuid                | ไอดีผู้แก้ไขล่าสุด                                        |
| deleted_at               | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)                    |

---

## 4. แอตทริบิวต์และแหล่งข้อมูล (Attributes Domain)

### ตาราง: attributes

ตารางกำหนดคุณสมบัติข้อมูลที่นำมาใช้ในเงื่อนไข (Attribute Entity)

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                     |
| :------------ | :------------------ | :----------------------------------------- |
| id            | uuid                | ไอดีหลักของแอตทริบิวต์                     |
| field_name    | varchar(255)        | ชื่อฟิลด์ทางเทคนิคของข้อมูล                |
| display_name  | varchar(255)        | ชื่อที่ใช้แสดงผลให้ผู้ใช้งานเห็น           |
| data_type     | enum                | ประเภทข้อมูล (Text, Date, Number, Boolean) |
| value         | varchar(255)        | ค่าที่เป็นไปได้หรือรายการ Enum (ถ้ามี)     |
| description   | text                | รายละเอียดอธิบายความหมายของแอตทริบิวต์     |
| source_system | varchar(255)        | ระบบต้นทางที่ให้ข้อมูลนี้มา                |
| is_active     | boolean             | สถานะการใช้งานแอตทริบิวต์                  |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างข้อมูล                |
| created_by    | uuid                | ไอดีผู้สร้างข้อมูล                         |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด                |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด                         |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)     |

---

## 5. การจัดตารางและตำแหน่ง (Delivery & Schedule Domain)

### ตาราง: placements

ตารางมาสเตอร์สำหรับกำหนดตำแหน่งการแสดงผล (Placement Entity)

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                    |
| :------------ | :------------------ | :---------------------------------------- |
| id            | uuid                | ไอดีหลักของตำแหน่งการวาง                  |
| name          | varchar(255)        | ชื่อตำแหน่งการวาง                         |
| description   | text                | รายละเอียดหรือหมายเหตุเกี่ยวกับตำแหน่งนี้ |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างข้อมูล               |
| created_by    | uuid                | ไอดีผู้สร้างข้อมูล                        |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด               |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด                        |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)    |

### ตาราง: schedules

ตารางกำหนดช่วงเวลาการทำงานของกฎในแต่ละตำแหน่ง (Schedule Entity)

| ฟิลด์ (Field)     | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                                                                     |
| :---------------- | :------------------ | :----------------------------------------------------------------------------------------- |
| id                | uuid                | ไอดีหลักของตารางเวลา                                                                       |
| decision_rule_id  | uuid                | ไอดีอ้างอิงกฎการตัดสินใจ (FK → decision_rules.id)                                          |
| placement_id      | uuid                | ไอดีอ้างอิงตำแหน่งการวาง (FK → placements.id)                                              |
| calendar_id       | uuid                | ไอดีปฏิทินที่ใช้อ้างอิง (FK → calendars.id) (Nullable)                                     |
| recurrence_type   | enum                | ประเภทการเกิดซ้ำ (ONCE = ครั้งเดียว, RRULE = กฎ iCal RFC5545, CALENDAR = ตามปฏิทินอ้างอิง) |
| recurrence_rule   | text                | กฎการเกิดซ้ำ (รูปแบบ iCal RRULE)                                                           |
| effective_from    | timestamptz         | วันและเวลาเริ่มต้นการมีผลใช้งาน                                                            |
| effective_until   | timestamptz         | วันและเวลาสิ้นสุดการมีผลใช้งาน                                                             |
| time_of_day_start | varchar(5)          | เวลาที่เริ่มต้นในแต่ละวัน (HH:mm)                                                          |
| time_of_day_end   | varchar(5)          | เวลาที่สิ้นสุดในแต่ละวัน (HH:mm)                                                           |
| all_day           | boolean             | สถานะการทำงานตลอดทั้งวัน (true = ใช่)                                                      |
| timezone          | varchar(255)        | เขตเวลา (Default: 'Asia/Bangkok')                                                          |
| is_active         | boolean             | สถานะการเปิดใช้งานตารางเวลานี้                                                             |
| created_at        | timestamptz         | วันที่และเวลาที่สร้างข้อมูล                                                                |
| created_by        | uuid                | ไอดีผู้สร้างข้อมูล                                                                         |
| updated_at        | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด                                                                |
| updated_by        | uuid                | ไอดีผู้แก้ไขล่าสุด                                                                         |
| deleted_at        | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)                                                     |

### ตาราง: schedule_occurrences

ตารางเก็บรายการการทำงานจริงที่ถูกสร้างขึ้นจากตารางเวลา (Schedule Occurrence Entity)

| ฟิลด์ (Field)    | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                                                                          |
| :--------------- | :------------------ | :---------------------------------------------------------------------------------------------- |
| id               | uuid                | ไอดีหลักของรายการการทำงาน                                                                       |
| schedule_id      | uuid                | ไอดีตารางเวลาอ้างอิง (FK → schedules.id)                                                        |
| occurrence_start | timestamptz         | วันและเวลาเริ่มต้นของรายการนี้                                                                  |
| occurrence_end   | timestamptz         | วันและเวลาสิ้นสุดของรายการนี้                                                                   |
| status           | enum                | สถานะของรายการ (ACTIVE = ใช้งาน, CANCELLED = ยกเลิก, MODIFIED = แก้ไขแล้ว)                      |
| source           | enum                | แหล่งที่มาของรายการ (RECURRENCE = สร้างจากกฎ, CALENDAR = สร้างจากปฏิทิน, MANUAL = สร้างด้วยมือ) |
| created_at       | timestamptz         | วันที่และเวลาที่สร้างรายการ                                                                     |
| created_by       | uuid                | ไอดีผู้สร้างรายการ                                                                              |
| updated_at       | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด                                                                     |
| updated_by       | uuid                | ไอดีผู้แก้ไขล่าสุด                                                                              |
| deleted_at       | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)                                                          |

### ตาราง: calendars

ตารางมาสเตอร์สำหรับปฏิทินอ้างอิง (Calendar Entity)

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                                                           |
| :------------ | :------------------ | :------------------------------------------------------------------------------- |
| id            | uuid                | ไอดีหลักของปฏิทิน                                                                |
| name          | varchar(255)        | ชื่อปฏิทิน (เช่น วันหยุดธนาคาร 2569)                                             |
| type          | enum                | ประเภทปฏิทิน (HOLIDAY = วันหยุดราชการ, PERSONAL = วันส่วนตัว, CUSTOM = กำหนดเอง) |
| is_active     | boolean             | สถานะการใช้งานปฏิทิน                                                             |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างข้อมูล                                                      |
| created_by    | uuid                | ไอดีผู้สร้างข้อมูล                                                               |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด                                                      |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด                                                               |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete)                                           |

### ตาราง: calendar_dates

ตารางเก็บวันที่ระบุภายในปฏิทิน (Calendar Date Entity)

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                 |
| :------------ | :------------------ | :------------------------------------- |
| id            | uuid                | ไอดีหลักของรายการวันที่                |
| calendar_id   | uuid                | ไอดีปฏิทินอ้างอิง (FK → calendars.id)  |
| date          | date                | วันที่ที่ระบุ                          |
| name          | varchar(255)        | ชื่อเรียกของวัน (เช่น วันสงกรานต์)     |
| is_recurring  | boolean             | สถานะการเกิดซ้ำทุกปี                   |
| created_at    | timestamptz         | วันที่และเวลาที่สร้างข้อมูล            |
| created_by    | uuid                | ไอดีผู้สร้างข้อมูล                     |
| updated_at    | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด            |
| updated_by    | uuid                | ไอดีผู้แก้ไขล่าสุด                     |
| deleted_at    | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete) |

---

## 6. ข้อมูลโครงสร้างส่วนกลาง (Schema Domain)

### ตาราง: mdp_schema_registry

ระบบลงทะเบียนโครงสร้างข้อมูลสำหรับ Frontend และการประมวลผล (Registry Entity)

| ฟิลด์ (Field)     | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                 |
| :---------------- | :------------------ | :------------------------------------- |
| id                | uuid                | ไอดีหลักของรายการรีจิสทรี              |
| schema_name       | varchar(255)        | ชื่อของโครงสร้างข้อมูล (Schema Name)   |
| version           | varchar(255)        | เวอร์ชันของโครงสร้างข้อมูล             |
| schema_definition | jsonb               | รายละเอียดโครงสร้างข้อมูลในรูปแบบ JSON |
| is_active         | boolean             | สถานะการใช้งาน Schema นี้              |
| created_at        | timestamptz         | วันที่และเวลาที่สร้างข้อมูล            |
| created_by        | uuid                | ไอดีผู้สร้างข้อมูล                     |
| updated_at        | timestamptz         | วันที่และเวลาที่แก้ไขล่าสุด            |
| updated_by        | uuid                | ไอดีผู้แก้ไขล่าสุด                     |
| deleted_at        | timestamptz         | วันที่และเวลาที่ลบข้อมูล (Soft-delete) |
