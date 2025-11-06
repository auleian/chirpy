-- +goose Up
CREATE TABLE refresh_token (
    token VARCHAR PRIMARY KEY ,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP,
    revoked_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE refresh_token;