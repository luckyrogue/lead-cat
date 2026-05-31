package asynqqueue

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

const TaskRunScenario = "scenario:run"

type RunPayload struct {
	RunID      string `json:"run_id"`
	ScenarioID string `json:"scenario_id"`
	Trigger    string `json:"trigger"`
}

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

func (c *Client) EnqueueRun(ctx context.Context, runID, scenarioID uuid.UUID, trigger string) error {
	p, _ := json.Marshal(RunPayload{
		RunID:      runID.String(),
		ScenarioID: scenarioID.String(),
		Trigger:    trigger,
	})
	task := asynq.NewTask(TaskRunScenario, p)
	_, err := c.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

const TaskMeetingCreated = "meeting:created"

type MeetingCreatedPayload struct {
	WorkspaceID string `json:"workspace_id"`
	MeetingID   string `json:"meeting_id"`
}

func (c *Client) EnqueueMeetingCreated(ctx context.Context, workspaceID, meetingID uuid.UUID) error {
	p, _ := json.Marshal(MeetingCreatedPayload{
		WorkspaceID: workspaceID.String(),
		MeetingID:   meetingID.String(),
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

func ParsePayload(t *asynq.Task) (RunPayload, error) {
	var p RunPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}
