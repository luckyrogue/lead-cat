package scenario_executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/cats"
	"github.com/Jaryq-Lab/notify-bot/internal/domain/scenario"
	domainvcs "github.com/Jaryq-Lab/notify-bot/internal/domain/vcs"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/crypto"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	ghvcs "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/vcs/github"
	glvcs "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/vcs/gitlab"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/commitsreport"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/observability/log"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/observability/metrics"
)

type Executor struct {
	store  *postgres.Store
	cipher *crypto.TokenCipher
	bot    *bot.Bot
	log    *zap.Logger
}

func New(store *postgres.Store, cipher *crypto.TokenCipher, b *bot.Bot, log *zap.Logger) *Executor {
	return &Executor{store: store, cipher: cipher, bot: b, log: log}
}

func (e *Executor) Run(ctx context.Context, runID, scenarioID uuid.UUID, trigger string) error {
	ctx = log.WithRunID(log.WithScenarioID(ctx, scenarioID), runID)
	var finishStatus string
	defer func() {
		if finishStatus != "" {
			metrics.IncScenarioRun(finishStatus)
		}
	}()

	sc, err := e.store.GetScenario(ctx, scenarioID)
	if err != nil {
		return err
	}
	ws, err := e.store.GetWorkspace(ctx, sc.WorkspaceID)
	if err != nil {
		return err
	}
	if ws.NotifyChatID == nil {
		finishStatus = "failed"
		return e.finishRun(ctx, runID, "failed", "chat not linked")
	}
	def, err := scenario.ParseDefinition(sc.Definition)
	if err != nil {
		finishStatus = "failed"
		return e.finishRun(ctx, runID, "failed", err.Error())
	}
	adj := def.Adjacency()
	var start scenario.Node
	for _, n := range def.Nodes {
		if n.Type != scenario.NodeTriggerCron && n.Type != scenario.NodeTriggerManual {
			continue
		}
		if trigger == "cron" && n.Type == scenario.NodeTriggerCron {
			start = n
			break
		}
		if trigger != "cron" && n.Type == scenario.NodeTriggerManual {
			start = n
			break
		}
	}
	if start.ID == "" && len(def.Nodes) > 0 {
		start = def.Nodes[0]
	}
	queue := []string{start.ID}
	seen := map[string]struct{}{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		node, ok := def.NodeByID(id)
		if !ok {
			continue
		}
		if strings.HasPrefix(node.Type, "trigger.") {
			for _, next := range adj[id] {
				queue = append(queue, next)
			}
			continue
		}
		t0 := time.Now()
		if err := e.execNode(ctx, node, ws, *ws.NotifyChatID); err != nil {
			_ = e.store.AddRunStep(ctx, runID, id, node.Type, "failed", nil, int(time.Since(t0).Milliseconds()), err.Error())
			finishStatus = "failed"
			log.WithContext(ctx, e.log).Error("scenario node failed", zap.String("node_id", id), zap.Error(err))
			return e.finishRun(ctx, runID, "failed", err.Error())
		}
		_ = e.store.AddRunStep(ctx, runID, id, node.Type, "success", json.RawMessage(`{}`), int(time.Since(t0).Milliseconds()), "")
		for _, next := range adj[id] {
			queue = append(queue, next)
		}
	}
	finishStatus = "success"
	return e.finishRun(ctx, runID, "success", "")
}

func (e *Executor) finishRun(ctx context.Context, runID uuid.UUID, status, msg string) error {
	if status == "failed" && msg != "" {
		log.WithContext(ctx, e.log).Warn("scenario run failed", zap.String("error", msg))
	}
	return e.store.FinishRun(ctx, runID, status, msg)
}

func (e *Executor) execNode(ctx context.Context, node scenario.Node, ws postgres.Workspace, chatID int64) error {
	switch node.Type {
	case scenario.NodeActionTelegramMessage:
		var p struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(node.Parameters, &p)
		text := strings.ReplaceAll(p.Text, "{{meet_link}}", ws.MeetLink)
		_, err := e.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   text,
		})
		return err
	case scenario.NodeActionTelegramCat:
		url := cats.RandomImageURL(ctx)
		if url == "" {
			return fmt.Errorf("no cat image")
		}
		_, err := e.bot.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID: chatID,
			Photo:  &models.InputFileString{Data: url},
		})
		return err
	case scenario.NodeActionVCSReport:
		return e.SendCommitsReport(ctx, ws)
	default:
		return fmt.Errorf("unknown node type %s", node.Type)
	}
}

func (e *Executor) SendCommitsReport(ctx context.Context, ws postgres.Workspace) error {
	enc, err := e.store.GetVCSTokenEnc(ctx, ws.ID)
	if err != nil {
		return err
	}
	token, err := e.cipher.Decrypt(enc)
	if err != nil || token == "" {
		return fmt.Errorf("vcs token not configured")
	}
	ownerID, err := e.store.OwnerTelegramID(ctx, ws.ID)
	if err != nil {
		return err
	}
	maps, err := e.store.ListVCSMaps(ctx, ws.ID)
	if err != nil {
		return err
	}
	members, err := e.store.ListMembers(ctx, ws.ID)
	if err != nil {
		return err
	}
	var mappings []commitsreport.DevMapping
	for _, m := range members {
		logins := maps[m.TelegramUsername]
		vcsLogin := logins.GitHub
		if ws.VCSProvider == "gitlab" {
			vcsLogin = logins.GitLab
		}
		if vcsLogin == "" {
			vcsLogin = m.TelegramUsername
		}
		mappings = append(mappings, commitsreport.DevMapping{Telegram: m.TelegramUsername, VCSLogin: vcsLogin})
	}
	var vcsClient domainvcs.Client
	switch ws.VCSProvider {
	case "gitlab":
		base := ""
		if ws.VCSBaseURL != nil {
			base = *ws.VCSBaseURL
		}
		vcsClient = &glvcs.Adapter{Client: glvcs.New(token, ws.VCSNamespace, base)}
	default:
		vcsClient = &ghvcs.Adapter{Client: ghvcs.NewClient(token, ws.VCSNamespace)}
	}
	b := &commitsreport.Builder{
		VCS:      vcsClient,
		Mappings: commitsreport.SortMappings(mappings),
	}
	loc, _ := time.LoadLocation(ws.TZ)
	text, err := b.Daily(ctx, time.Now(), loc)
	if err != nil {
		return err
	}
	_, err = e.bot.SendMessage(ctx, &bot.SendMessageParams{ChatID: ownerID, Text: text})
	return err
}
