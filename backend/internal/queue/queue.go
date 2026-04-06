package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

const (
	keyPrefix      = "verdox:jobs:repo:"
	activePrefix   = "verdox:jobs:active:"
	processingKey  = "verdox:jobs:processing:"
	logPrefix      = "verdox:logs:"
	logStreamSfx   = ":stream"
	cancelPrefix   = "verdox:cancel:"
	popTimeout     = 1 * time.Second
	logTTL         = 1 * time.Hour
)

type RedisQueue struct {
	client        *redis.Client
	runnerTimeout int // seconds — used to compute active lock TTL
	log           zerolog.Logger
}

func NewRedisQueue(client *redis.Client, runnerTimeout int, log zerolog.Logger) *RedisQueue {
	return &RedisQueue{client: client, runnerTimeout: runnerTimeout, log: log}
}

// Push enqueues a job to the per-repo FIFO queue.
func (q *RedisQueue) Push(ctx context.Context, job *model.JobPayload) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	key := keyPrefix + job.RepoID
	return q.client.LPush(ctx, key, data).Err()
}

// Pop scans per-repo queues and returns the next available job.
// It skips repos that already have an active run. Returns nil if no work available.
func (q *RedisQueue) Pop(ctx context.Context, workerID string) (*model.JobPayload, error) {
	// Scan all repo queues
	var cursor uint64
	for {
		keys, nextCursor, err := q.client.Scan(ctx, cursor, keyPrefix+"*", 50).Result()
		if err != nil {
			return nil, fmt.Errorf("scan keys: %w", err)
		}

		for _, key := range keys {
			repoID := key[len(keyPrefix):]
			activeKey := activePrefix + repoID

			// Skip repos with an active run
			exists, err := q.client.Exists(ctx, activeKey).Result()
			if err != nil {
				q.log.Warn().Err(err).Str("repo_id", repoID).Msg("check active key failed")
				continue
			}
			if exists > 0 {
				continue
			}

			// Try to pop from this repo's queue
			data, err := q.client.RPop(ctx, key).Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				q.log.Warn().Err(err).Str("key", key).Msg("rpop failed")
				continue
			}

			var job model.JobPayload
			if err := json.Unmarshal([]byte(data), &job); err != nil {
				q.log.Error().Err(err).Str("data", data).Msg("unmarshal job failed")
				continue
			}

			// Set active lock with TTL = 2x runner timeout
			ttl := time.Duration(q.runnerTimeout*2) * time.Second
			if err := q.client.Set(ctx, activeKey, job.TestRunID, ttl).Err(); err != nil {
				q.log.Error().Err(err).Msg("set active lock failed")
				// Re-enqueue the job since we couldn't lock
				q.client.RPush(ctx, key, data)
				continue
			}

			// Store in processing list for crash recovery
			procKey := processingKey + workerID
			q.client.LPush(ctx, procKey, data)

			return &job, nil
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil, nil
}

// Ack acknowledges a completed job by releasing the active lock.
func (q *RedisQueue) Ack(ctx context.Context, workerID string, job *model.JobPayload) error {
	activeKey := activePrefix + job.RepoID
	q.client.Del(ctx, activeKey)

	procKey := processingKey + workerID
	data, _ := json.Marshal(job)
	q.client.LRem(ctx, procKey, 1, data)

	return nil
}

// RemoveByRunID removes a queued job from a repo's queue (for cancellation of queued runs).
func (q *RedisQueue) RemoveByRunID(ctx context.Context, repoID, runID string) error {
	key := keyPrefix + repoID

	// Get all items, find and remove the matching one
	items, err := q.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return err
	}

	for _, item := range items {
		var job model.JobPayload
		if json.Unmarshal([]byte(item), &job) == nil && job.TestRunID == runID {
			q.client.LRem(ctx, key, 1, item)
			return nil
		}
	}

	return nil
}

// PublishCancel sends a cancel signal for a running test run.
func (q *RedisQueue) PublishCancel(ctx context.Context, runID string) error {
	channel := cancelPrefix + runID
	return q.client.Publish(ctx, channel, "cancel").Err()
}

// SubscribeCancel returns a PubSub subscription for cancel signals.
func (q *RedisQueue) SubscribeCancel(ctx context.Context, runID string) *redis.PubSub {
	channel := cancelPrefix + runID
	return q.client.Subscribe(ctx, channel)
}

// AppendLog appends a log line for a run and publishes it on the stream channel.
func (q *RedisQueue) AppendLog(ctx context.Context, runID, line string) error {
	logKey := logPrefix + runID
	q.client.Append(ctx, logKey, line+"\n")
	q.client.Expire(ctx, logKey, logTTL)

	streamChannel := logPrefix + runID + logStreamSfx
	q.client.Publish(ctx, streamChannel, line)

	return nil
}

// GetLogs retrieves the buffered log output for a run.
func (q *RedisQueue) GetLogs(ctx context.Context, runID string) (string, error) {
	logKey := logPrefix + runID
	data, err := q.client.Get(ctx, logKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	return data, err
}
