-- +goose Up
CREATE TABLE workshop_settings (
    singleton_id SMALLINT PRIMARY KEY DEFAULT 1
        CONSTRAINT workshop_settings_singleton CHECK (singleton_id = 1),
    workshop_name TEXT NOT NULL DEFAULT 'Workshop'
        CONSTRAINT workshop_settings_name_valid CHECK (
            char_length(btrim(workshop_name)) BETWEEN 1 AND 200
        ),
    logo_file_id UUID,
    default_locale TEXT NOT NULL DEFAULT 'pt-BR'
        CONSTRAINT workshop_settings_locale_valid CHECK (
            default_locale ~ '^[a-z]{2}-[A-Z]{2}$'
        ),
    default_currency TEXT NOT NULL DEFAULT 'BRL'
        CONSTRAINT workshop_settings_currency_valid CHECK (
            default_currency ~ '^[A-Z]{3}$'
        ),
    display_timezone TEXT NOT NULL DEFAULT 'America/Sao_Paulo'
        CONSTRAINT workshop_settings_timezone_valid CHECK (
            char_length(display_timezone) BETWEEN 1 AND 100
        ),
    default_theme TEXT NOT NULL DEFAULT 'system'
        CONSTRAINT workshop_settings_theme_valid CHECK (
            default_theme IN ('light', 'dark', 'system')
        ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT workshop_settings_updated_after_creation CHECK (updated_at >= created_at)
);
