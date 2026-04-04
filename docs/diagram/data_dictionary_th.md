# Data Dictionary for ECMS (พจนานุกรมข้อมูลสำหรับระบบ ECMS)

เอกสารฉบับนี้รวบรวมรายละเอียดของตารางและฟิลด์ต่างๆ ที่ระบุไว้ใน `models.md` พร้อมคำอธิบายภาษาไทย

## 1. การจัดการสิทธิ์และการเข้าใช้งาน (Management & Auth)

### ตาราง: login_token_histories

ใช้สำหรับเก็บประวัติการเข้าใช้งานและข้อมูล Token

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                  |
| :------------ | :------------------ | :-------------------------------------- |
| id            | uuid                | ไอดีหลัก (Primary Key) ของรายการประวัติ |
| username      | varchar             | ชื่อผู้ใช้งานที่ใช้เข้าสู่ระบบ          |
| access_token  | varchar             | รหัสโทเค็นที่ใช้สำหรับการเข้าถึงระบบ    |
| expire_date   | timestamptz         | วันและเวลาที่โทเค็นหมดอายุ              |

### ตาราง: users

ตารางข้อมูลผู้ใช้งานระบบ

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                               |
| :------------ | :------------------ | :--------------------------------------------------- |
| id            | uuid                | ไอดีหลักของผู้ใช้งาน                                 |
| role_id       | uuid                | ไอดีของสิทธิ์การใช้งาน (เชื่อมโยงกับตาราง roles)     |
| profile_id    | uuid                | ไอดีของโปรไฟล์ผู้ใช้งาน (เชื่อมโยงกับตาราง profiles) |
| email         | varchar             | ที่อยู่อีเมลของผู้ใช้งาน                             |
| name_th       | varchar             | ชื่อ-นามสกุล (ภาษาไทย)                               |
| name_en       | varchar             | ชื่อ-นามสกุล (ภาษาอังกฤษ)                            |
| is_active     | boolean             | สถานะการใช้งาน (เปิด/ปิด)                            |
| created_at    | timestamptz         | วันที่สร้างข้อมูล                                    |
| created_by    | uuid                | ผู้สร้างข้อมูล                                       |
| updated_at    | timestamptz         | วันที่แก้ไขข้อมูลล่าสุด                              |
| updated_by    | uuid                | ผู้แก้ไขข้อมูลล่าสุด                                 |

### ตาราง: roles

ตารางกำหนดบทบาทหน้าที่

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description) |
| :------------ | :------------------ | :--------------------- |
| id            | uuid                | ไอดีหลักของบทบาท       |
| name          | varchar             | ชื่อของบทบาท           |
| code          | varchar             | รหัสอ้างอิงของบทบาท    |
| created_at    | timestamptz         | วันที่สร้างข้อมูล      |
| created_by    | uuid                | ผู้สร้างข้อมูล         |
| updated_at    | timestamptz         | วันที่แก้ไขล่าสุด      |
| updated_by    | uuid                | ผู้แก้ไขล่าสุด         |

### ตาราง: profiles

ตารางกำหนดกลุ่มข้อมูลผู้ใช้งาน

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description) |
| :------------ | :------------------ | :--------------------- |
| id            | uuid                | ไอดีหลักของโปรไฟล์     |
| name          | varchar             | ชื่อของโปรไฟล์         |
| code          | varchar             | รหัสอ้างอิงของโปรไฟล์  |
| created_at    | timestamptz         | วันที่สร้างข้อมูล      |
| created_by    | uuid                | ผู้สร้างข้อมูล         |
| updated_at    | timestamptz         | วันที่แก้ไขล่าสุด      |
| updated_by    | uuid                | ผู้แก้ไขล่าสุด         |

### ตาราง: permissions

ตารางกำหนดสิทธิ์การเข้าถึงและการกระทำในระบบ

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                                                      |
| :------------ | :------------------ | :-------------------------------------------------------------------------- |
| id            | uuid                | ไอดีหลักของสิทธิ์                                                           |
| name          | varchar             | ชื่อของสิทธิ์                                                               |
| source        | varchar             | แหล่งที่มาของสิทธิ์ (เช่น CONTENT_DECISION_RULE)                            |
| action        | varchar             | การกระทำที่ได้รับอนุญาต (เช่น CREATE,EDIT,DELETE,VIEW_ALL,EDIT_ALL,DELETE_ALL) |
| created_at    | timestamptz         | วันที่สร้างข้อมูล                                                           |
| created_by    | uuid                | ผู้สร้างข้อมูล                                                              |
| updated_at    | timestamptz         | วันที่แก้ไขล่าสุด                                                           |
| updated_by    | uuid                | ผู้แก้ไขล่าสุด                                                              |

### ตาราง: profile_permisssions

ตารางเชื่อมโยงโปรไฟล์กับสิทธิ์การเข้าถึง (Relation Table)

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description) |
| :------------ | :------------------ | :--------------------- |
| id            | uuid                | ไอดีหลักของรายการ      |
| profile_id    | uuid                | ไอดีโปรไฟล์            |
| permission_id | uuid                | ไอดีสิทธิ์             |

---

## 2. ระบบหลักของกฎการตัดสินใจ (Decision Rule Core)

### ตาราง: decision_rules

ตารางหลักสำหรับเก็บข้อมูลกฎการตัดสินใจ

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)          |
| :------------ | :------------------ | :------------------------------ |
| id            | uuid                | ไอดีหลักของกฎการตัดสินใจ        |
| name          | varchar             | ชื่อของกฎการตัดสินใจ            |
| type          | enum                | ประเภทของกฎ                     |
| content_path  | varchar             | เส้นทางที่เก็บไฟล์เนื้อหาของกฎ  |
| score         | integer             | คะแนนรวมของกฎ                   |
| status        | enum                | สถานะ (DRAFT, ACTIVE, INACTIVE) |
| created_at    | timestamptz         | วันที่สร้างกฎ                   |
| created_by    | uuid                | ผู้สร้างกฎ                      |
| updated_at    | timestamptz         | วันที่แก้ไขล่าสุด               |
| updated_by    | uuid                | ผู้แก้ไขล่าสุด                  |
| inactive_by   | varchar             | ผู้ที่ทำการปิดการใช้งานกฎ       |

### ตาราง: rules

ตารางสำหรับเก็บข้อมูลกฎย่อยหรือตัวแปรของกฎ

| ฟิลด์ (Field)    | ประเภทข้อมูล (Type) | คำอธิบาย (Description)             |
| :--------------- | :------------------ | :--------------------------------- |
| id               | uuid                | ไอดีหลักของกฎย่อย                  |
| decision_rule_id | uuid                | ไอดีอ้างอิงถึงตาราง decision_rules |
| variation_name   | varchar             | ชื่อของรูปแบบหรือตัวแปรของกฎ       |
| score            | integer             | คะแนนของรูปแบบนี้                  |
| order_no         | integer             | ลำดับการทำงาน (เรียงจากน้อยไปมาก)  |

---

## 3. เครื่องมือตรรกะขั้นสูง (Advanced Logic Engine)

### ตาราง: rule_condition_groups

ตารางจัดกลุ่มเงื่อนไขของกฎ

| ฟิลด์ (Field)                | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                        |
| :--------------------------- | :------------------ | :-------------------------------------------- |
| id                           | uuid                | ไอดีหลักของกลุ่มเงื่อนไข                      |
| parent_rule_condition_groups | uuid                | ไอดีของกลุ่มแม่ (สำหรับกรณีเงื่อนไขซ้อนกลุ่ม) |

### ตาราง: rule_condition

ตารางเก็บเงื่อนไขแต่ละรายการ

| ฟิลด์ (Field)           | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                              |
| :---------------------- | :------------------ | :-------------------------------------------------- |
| id                      | uuid                | ไอดีหลักของเงื่อนไข                                 |
| sequence                | integer             | ลำดับที่ของเงื่อนไขภายในกลุ่ม                       |
| decision_rule_id        | uuid                | ไอดีอ้างอิงกฎหลัก                                   |
| rule_id                 | uuid                | ไอดีอ้างอิงกฎย่อย                                   |
| rule_condition_group_id | uuid                | ไอดีอ้างอิงกลุ่มเงื่อนไข                            |
| attribute_id            | uuid                | ไอดีอ้างอิงแอตทริบิวต์ (คุณสมบัติข้อมูล)            |
| logical_operator        | enum                | ตัวดำเนินการเปรียบเทียบ (เช่น <, >, =, IN, BETWEEN) |
| value                   | jsonb               | ค่าข้อมูลที่ใช้เปรียบเทียบ                          |
| connector_operator      | enum                | ตัวเชื่อมความสัมพันธ์ (AND, OR) กับเงื่อนไขถัดไป    |

---

## 4. แอตทริบิวต์และแหล่งที่มา (Attributes & Sources)

### ตาราง: attributes

ตารางกำหนดคุณสมบัติข้อมูลที่ใช้ในเงื่อนไข

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description)                     |
| :------------ | :------------------ | :----------------------------------------- |
| id            | uuid                | ไอดีหลักของแอตทริบิวต์                     |
| field_name    | varchar             | ชื่อฟิลด์ของข้อมูล                         |
| display_name  | varchar             | ชื่อที่ใช้แสดงผลบนหน้าจอ                   |
| data_type     | enum                | ประเภทข้อมูล (Text, Date, Number, Boolean) |
| value         | varchar             | ค่าที่เป็นไปได้                            |
| description   | text                | คำอธิบายแอตทริบิวต์เพิ่มเติม               |
| source_system | varchar             | ระบบแหล่งที่มาของข้อมูล                    |
| is_active     | boolean             | สถานะการใช้งาน                             |
| created_at    | timestamptz         | วันที่สร้าง                                |
| updated_at    | timestamptz         | วันที่แก้ไขล่าสุด                          |

---

## 5. การจัดส่งและกำหนดเวลา (Delivery & Scheduling)

### ตาราง: placements

ตารางมาสเตอร์สำหรับตำแหน่งการวางเนื้อหา

| ฟิลด์ (Field) | ประเภทข้อมูล (Type) | คำอธิบาย (Description) |
| :------------ | :------------------ | :--------------------- |
| id            | uuid                | ไอดีหลักของตำแหน่ง     |
| name          | varchar             | ชื่อตำแหน่ง            |
| description   | text                | รายละเอียดตำแหน่ง      |

### ตาราง: schedules

ตารางกำหนดเวลาการใช้งานกฎในตำแหน่งต่างๆ

| ฟิลด์ (Field)    | ประเภทข้อมูล (Type) | คำอธิบาย (Description)  |
| :--------------- | :------------------ | :---------------------- |
| id               | uuid                | ไอดีหลักของการกำหนดเวลา |
| decision_rule_id | uuid                | ไอดีอ้างอิงกฎ           |
| placement_id     | uuid                | ไอดีอ้างอิงตำแหน่ง      |
| start_timestamp  | timestamptz         | วันและเวลาเริ่มต้น      |
| end_timestamp    | timestamptz         | วันและเวลาสิ้นสุด       |
| is_active        | boolean             | สถานะการใช้งาน          |

---

## 6. ข้อมูลโครงสร้าง (Schema)

### ตาราง: schema_registry

ระบบลงทะเบียนโครงสร้างข้อมูลสำหรับ Frontend

| ฟิลด์ (Field)     | ประเภทข้อมูล (Type) | คำอธิบาย (Description)           |
| :---------------- | :------------------ | :------------------------------- |
| id                | uuid                | ไอดีหลักของรีจิสทรี              |
| schema_name       | varchar             | ชื่อโครงสร้างข้อมูล              |
| version           | varchar             | เวอร์ชัน                         |
| schema_definition | jsonb               | รายละเอียดโครงสร้างในรูปแบบ JSON |
| is_active         | boolean             | สถานะการใช้งาน                   |
| created_at        | timestamp           | วันที่สร้าง                      |
