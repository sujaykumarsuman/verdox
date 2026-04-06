ALTER TABLE users ADD COLUMN ban_reason TEXT;

CREATE TABLE ban_reviews (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id),
    ban_reason    TEXT NOT NULL,
    clarification TEXT NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by   UUID REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at   TIMESTAMPTZ
);

CREATE INDEX idx_ban_reviews_user_id ON ban_reviews(user_id);
CREATE INDEX idx_ban_reviews_status ON ban_reviews(status);
