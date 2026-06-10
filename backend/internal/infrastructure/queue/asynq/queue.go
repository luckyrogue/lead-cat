package asynqqueue

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type Client struct {
	client *asynq.Client
	log    *zap.Logger
}

func NewClient(redisURL string, log *zap.Logger) (*Client, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, err
	}
	return &Client{client: asynq.NewClient(opt), log: log}, nil
}

func (c *Client) Close() error { return c.client.Close() }

const TaskMeetingCreated = "meeting:created"

type MeetingCreatedPayload struct {
	OrganizationID string `json:"organization_id"`
	MeetingID      string `json:"meeting_id"`
}

func (c *Client) EnqueueMeetingCreated(ctx context.Context, organizationID, meetingID uuid.UUID) error {
	p, _ := json.Marshal(MeetingCreatedPayload{
		OrganizationID: organizationID.String(),
		MeetingID:      meetingID.String(),
	})
	task := asynq.NewTask(TaskMeetingCreated, p)
	_, err := c.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

func ParseMeetingCreated(t *asynq.Task) (MeetingCreatedPayload, error) {
	var p MeetingCreatedPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

const TaskMeetingUpdated = "meeting:updated"

type MeetingUpdatedPayload struct {
	OrganizationID string `json:"organization_id"`
	MeetingID      string `json:"meeting_id"`
}

func (c *Client) EnqueueMeetingUpdated(ctx context.Context, organizationID, meetingID uuid.UUID) error {
	p, _ := json.Marshal(MeetingUpdatedPayload{
		OrganizationID: organizationID.String(),
		MeetingID:      meetingID.String(),
	})
	task := asynq.NewTask(TaskMeetingUpdated, p)
	_, err := c.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

func ParseMeetingUpdated(t *asynq.Task) (MeetingUpdatedPayload, error) {
	var p MeetingUpdatedPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

const TaskMeetingCancelled = "meeting:cancelled"

type MeetingCancelledPayload struct {
	OrganizationID string `json:"organization_id"`
	MeetingID      string `json:"meeting_id"`
}

func (c *Client) EnqueueMeetingCancelled(ctx context.Context, organizationID, meetingID uuid.UUID) error {
	p, _ := json.Marshal(MeetingCancelledPayload{
		OrganizationID: organizationID.String(),
		MeetingID:      meetingID.String(),
	})
	task := asynq.NewTask(TaskMeetingCancelled, p)
	_, err := c.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

func ParseMeetingCancelled(t *asynq.Task) (MeetingCancelledPayload, error) {
	var p MeetingCancelledPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

const (
	TaskParticipantAdded   = "participant:added"
	TaskParticipantRemoved = "participant:removed"
)

type ParticipantPayload struct {
	OrganizationID string `json:"organization_id"`
	MeetingID      string `json:"meeting_id"`
	Email          string `json:"email"`
}

func (c *Client) EnqueueParticipantAdded(ctx context.Context, organizationID, meetingID uuid.UUID, email string) error {
	return c.enqueueParticipant(ctx, TaskParticipantAdded, organizationID, meetingID, email)
}

func (c *Client) EnqueueParticipantRemoved(ctx context.Context, organizationID, meetingID uuid.UUID, email string) error {
	return c.enqueueParticipant(ctx, TaskParticipantRemoved, organizationID, meetingID, email)
}

func (c *Client) enqueueParticipant(ctx context.Context, taskType string, organizationID, meetingID uuid.UUID, email string) error {
	p, _ := json.Marshal(ParticipantPayload{
		OrganizationID: organizationID.String(),
		MeetingID:      meetingID.String(),
		Email:          email,
	})
	task := asynq.NewTask(taskType, p)
	_, err := c.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

func ParseParticipant(t *asynq.Task) (ParticipantPayload, error) {
	var p ParticipantPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

func NewServer(redisURL string, log *zap.Logger, handlers map[string]asynq.HandlerFunc) (*Server, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, err
	}
	srv := asynq.NewServer(opt, asynq.Config{Concurrency: 4})
	mux := asynq.NewServeMux()
	for taskType, h := range handlers {
		mux.HandleFunc(taskType, h)
	}
	return &Server{server: srv, mux: mux}, nil
}

func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(s.mux)
}

func (s *Server) Shutdown() {
	s.server.Shutdown()
}
