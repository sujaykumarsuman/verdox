CREATE TYPE notification_type AS ENUM ('system', 'admin_message', 'ban_review', 'test_complete', 'team_invite');

CREATE TABLE notifications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type          notification_type NOT NULL,
    subject       VARCHAR(255) NOT NULL,
    body          TEXT NOT NULL DEFAULT '',
    is_read       BOOLEAN NOT NULL DEFAULT false,
    action_type   VARCHAR(64),
    action_payload JSONB,
    sender_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read) WHERE is_read = false;
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);
