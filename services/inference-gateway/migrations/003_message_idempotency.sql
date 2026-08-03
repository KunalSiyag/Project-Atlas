ALTER TABLE conversation_messages
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS conversation_messages_idempotency_idx
    ON conversation_messages (conversation_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
