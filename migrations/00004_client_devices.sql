-- +goose Up
CREATE TABLE client_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name TEXT NOT NULL
        CONSTRAINT client_devices_display_name_not_blank CHECK (btrim(display_name) <> '')
        CONSTRAINT client_devices_display_name_trimmed CHECK (display_name = btrim(display_name)),
    os TEXT NOT NULL
        CONSTRAINT client_devices_os_not_blank CHECK (btrim(os) <> '')
        CONSTRAINT client_devices_os_trimmed CHECK (os = btrim(os)),
    app_version TEXT NOT NULL
        CONSTRAINT client_devices_app_version_not_blank CHECK (btrim(app_version) <> '')
        CONSTRAINT client_devices_app_version_trimmed CHECK (app_version = btrim(app_version)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT client_devices_last_seen_not_before_created CHECK (last_seen_at >= created_at)
);
