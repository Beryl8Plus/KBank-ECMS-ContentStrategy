-- +goose Up
-- 1. Placements
INSERT INTO placements ("ID", "NAME", "DESCRIPTION", "MAX_RESULTS", "CREATED_AT", "UPDATED_AT") VALUES 
('44444444-4444-4444-4444-444444444444', 'wsaHomeBanner', 'Home screen banner placement', 10, NOW(), NOW()),
('55555555-5555-5555-5555-555555555555', 'wsaPortBanner', 'Portfolio banner placement', 10, NOW(), NOW()),
('66666666-6666-6666-6666-666666666666', 'wsaSplash', 'Splash screen placement', 1, NOW(), NOW());

-- 2. clen_schema_registry
INSERT INTO clen_schema_registry ("ID", "SCHEMA_NAME", "VERSION", "SCHEMA_DEFINITION", "IS_ACTIVE", "CREATED_AT", "UPDATED_AT") VALUES 
('77777777-7777-7777-7777-777777777777', 'UserSession', '1.0.0', '{"type": "object", "properties": {"segment": {"type": "string"}}}', true, NOW(), NOW());

-- 3. Attributes
INSERT INTO attributes ("ID", "CLEN_SCHEMA_REGISTRY_ID", "FIELD_NAME", "DISPLAY_NAME", "DATA_TYPE", "VALUE", "DESCRIPTION", "SOURCE_SYSTEM", "IS_ACTIVE", "CREATED_AT", "UPDATED_AT") VALUES 
('88888888-8888-8888-8888-888888888888', '77777777-7777-7777-7777-777777777777', 'segment', 'User Segment', 'Text', '["VIP", "Standard"]', 'Segmentation of users', 'CLEN', true, NOW(), NOW()),
('99999999-9999-9999-9999-999999999999', '77777777-7777-7777-7777-777777777777', 'user_age', 'User Age', 'Number', NULL, 'Age of the user', 'CLEN', true, NOW(), NOW());

-- 4. Decision Rules
INSERT INTO decision_rules ("ID", "NAME", "TYPE", "EVALUATE_TYPE", "CONTENT_PATH", "SCORE", "STATUS", "CREATED_AT", "UPDATED_AT") VALUES 
('11111111-1111-1111-1111-111111111111', 'wsaHomeBanner', 'AUDIENCE', 'SCORING', 'personalizedContent', 1.0, 'ACTIVE', NOW(), NOW()),
('22222222-2222-2222-2222-222222222222', 'wsaPortBanner', 'AUDIENCE', 'SCORING', 'personalizedContent', 1.0, 'ACTIVE', NOW(), NOW()),
('33333333-3333-3333-3333-333333333333', 'wsaSplash', 'AUDIENCE', 'SCORING', 'personalizedContent', 1.0, 'ACTIVE', NOW(), NOW());

-- 5. Rule Conditions
INSERT INTO rule_conditions ("ID", "SEQUENCE", "DECISION_RULE_ID", "ATTRIBUTE_ID", "LOGICAL_OPERATOR", "CONNECTOR_OPERATOR", "CREATED_AT", "UPDATED_AT") VALUES 
(gen_random_uuid(), 1, '11111111-1111-1111-1111-111111111111', '88888888-8888-8888-8888-888888888888', '=', 'AND', NOW(), NOW()),
(gen_random_uuid(), 1, '22222222-2222-2222-2222-222222222222', '88888888-8888-8888-8888-888888888888', '=', 'AND', NOW(), NOW()),
(gen_random_uuid(), 1, '33333333-3333-3333-3333-333333333333', '88888888-8888-8888-8888-888888888888', '=', 'AND', NOW(), NOW());

-- 6. Rules
INSERT INTO rules ("ID", "DECISION_RULE_ID", "VARIATION_NAME", "SCORE", "ORDER_NO", "CREATED_AT", "UPDATED_AT") VALUES 
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'VIP Variation', 10, 1, NOW(), NOW()),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'VIP Variation', 10, 1, NOW(), NOW()),
('cccccccc-cccc-cccc-cccc-cccccccccccc', '33333333-3333-3333-3333-333333333333', 'VIP Variation', 10, 1, NOW(), NOW());

-- 7. Rule Attributes
INSERT INTO rule_attributes ("ID", "RULE_ID", "ATTRIBUTE_ID", "VALUE", "CREATED_AT", "UPDATED_AT") VALUES 
(gen_random_uuid(), 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '88888888-8888-8888-8888-888888888888', '"VIP"', NOW(), NOW()),
(gen_random_uuid(), 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '88888888-8888-8888-8888-888888888888', '"VIP"', NOW(), NOW()),
(gen_random_uuid(), 'cccccccc-cccc-cccc-cccc-cccccccccccc', '88888888-8888-8888-8888-888888888888', '"VIP"', NOW(), NOW());

-- 8. Schedules
INSERT INTO schedules ("ID", "DECISION_RULE_ID", "PLACEMENT_ID", "RECURRENCE_TYPE", "EFFECTIVE_FROM", "EFFECTIVE_UNTIL", "IS_ACTIVE", "CREATED_AT", "UPDATED_AT") VALUES 
(gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444444', 'ONCE', NOW(), NOW() + interval '1 year', true, NOW(), NOW()),
(gen_random_uuid(), '22222222-2222-2222-2222-222222222222', '55555555-5555-5555-5555-555555555555', 'ONCE', NOW(), NOW() + interval '1 year', true, NOW(), NOW()),
(gen_random_uuid(), '33333333-3333-3333-3333-333333333333', '66666666-6666-6666-6666-666666666666', 'ONCE', NOW(), NOW() + interval '1 year', true, NOW(), NOW());

-- +goose Down
DELETE FROM schedules WHERE "DECISION_RULE_ID" IN ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', '33333333-3333-3333-3333-333333333333');
DELETE FROM rule_attributes WHERE "RULE_ID" IN ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'cccccccc-cccc-cccc-cccc-cccccccccccc');
DELETE FROM rules WHERE "ID" IN ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'cccccccc-cccc-cccc-cccc-cccccccccccc');
DELETE FROM rule_conditions WHERE "DECISION_RULE_ID" IN ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', '33333333-3333-3333-3333-333333333333');
DELETE FROM decision_rules WHERE "ID" IN ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', '33333333-3333-3333-3333-333333333333');
DELETE FROM attributes WHERE "ID" IN ('88888888-8888-8888-8888-888888888888', '99999999-9999-9999-9999-999999999999');
DELETE FROM clen_schema_registry WHERE "ID" = '77777777-7777-7777-7777-777777777777';
DELETE FROM placements WHERE "ID" IN ('44444444-4444-4444-4444-444444444444', '55555555-5555-5555-5555-555555555555', '66666666-6666-6666-6666-666666666666');