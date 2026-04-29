-- +goose Up
-- +goose StatementBegin

-- ────────────────────────────────────────────────
-- OAuth2 Clients linked to Profiles
-- Each client_id maps to a profile, which determines its scopes via profile_permissions.
-- ────────────────────────────────────────────────
INSERT INTO oauth2_clients ("ID", "CREATED_AT", "UPDATED_AT", "CLIENT_ID", "CLIENT_SECRET", "PROFILE_ID", "DESCRIPTION", "IS_ACTIVE") VALUES
    ('66666666-6666-6666-6666-000000000001', NOW(), NOW(),
        'service-cmc', 'super-secret-key-cmc',
        '22222222-2222-2222-2222-000000000002',
        'CMC service with Content Strategy Super Admin permissions',
        true),
    ('66666666-6666-6666-6666-000000000002', NOW(), NOW(),
        'service-marker', 'super-secret-key-marker',
        '22222222-2222-2222-2222-000000000001',
        'Marker service with Content Strategy Marker permissions',
        true),
    ('66666666-6666-6666-6666-000000000003', NOW(), NOW(),
        'service-it', 'super-secret-key-it',
        '22222222-2222-2222-2222-000000000003',
        'IT service with IT Admin permissions',
        true),
    ('66666666-6666-6666-6666-000000000004', NOW(), NOW(),
        'service-viewer', 'super-secret-key-viewer',
        '22222222-2222-2222-2222-000000000004',
        'Viewer service with read-only permissions',
        true)
ON CONFLICT ("CLIENT_ID") DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM oauth2_clients WHERE "CLIENT_ID" IN (
    'service-cmc',
    'service-marker',
    'service-it',
    'service-viewer'
);
-- +goose StatementEnd
