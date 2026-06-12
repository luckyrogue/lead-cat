package application

import (
	"context"

	"github.com/google/uuid"
)

// JobQueue enqueues asynchronous notification jobs (meeting + participant
// lifecycle). Implemented in infrastructure/queue/asynq.
type JobQueue interface {
	EnqueueMeetingCreated(ctx context.Context, organizationID, meetingID uuid.UUID) error
	EnqueueMeetingUpdated(ctx context.Context, organizationID, meetingID uuid.UUID) error
	EnqueueMeetingCancelled(ctx context.Context, organizationID, meetingID uuid.UUID) error
	EnqueueParticipantAdded(ctx context.Context, organizationID, meetingID uuid.UUID, email string) error
	EnqueueParticipantRemoved(ctx context.Context, organizationID, meetingID uuid.UUID, email string) error
}
