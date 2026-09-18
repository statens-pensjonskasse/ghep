package events

import (
	"context"
	"log/slog"
	"testing"

	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/mock"
	"github.com/navikt/ghep/internal/sql/gensql"
	"github.com/navikt/ghep/internal/testdata"
)

func TestHandleWorkflowJob(t *testing.T) {
	team := github.Team{
		Name:   "test",
		Config: github.Config{PingSlackUsers: true},
		Sources: []github.Source{
			{
				SourceType: "workflows",
				Channel:    "#test",
			},
		},
	}

	t.Run("waiting job posts a message and stores it", func(t *testing.T) {
		db := &mock.Database{}
		slack := &mock.Slack{}
		handler := NewHandler(db, slack, map[string]github.Team{"test": team})

		event, err := testdata.AsEvent("workflow-job-waiting-1.json")
		if err != nil {
			t.Fatal(err)
		}

		if err := handler.handleSource(context.TODO(), slog.Default(), team, team.Sources[0], event); err != nil {
			t.Error(err)
		}

		slack.Ensure(t, event.GetEventType(), 1, 0, 0)

		if len(db.SlackMessages) != 1 || db.SlackMessages[0].EventID != "workflow_job/41234567890" {
			t.Errorf("expected stored message with event id workflow_job/41234567890, got %+v", db.SlackMessages)
		}
	})

	t.Run("completed job updates the waiting message", func(t *testing.T) {
		db := &mock.Database{SlackMessages: []gensql.CreateSlackMessageParams{
			{TeamSlug: "test", EventID: "workflow_job/41234567890", ThreadTs: "1726650000.000100", Channel: "#test"},
		}}
		slack := &mock.Slack{}
		handler := NewHandler(db, slack, map[string]github.Team{"test": team})

		event, err := testdata.AsEvent("workflow-job-completed-1.json")
		if err != nil {
			t.Fatal(err)
		}

		if err := handler.handleSource(context.TODO(), slog.Default(), team, team.Sources[0], event); err != nil {
			t.Error(err)
		}

		slack.Ensure(t, event.GetEventType(), 0, 0, 1)
	})

	t.Run("completed job that never waited is ignored", func(t *testing.T) {
		db := &mock.Database{}
		slack := &mock.Slack{}
		handler := NewHandler(db, slack, map[string]github.Team{"test": team})

		event, err := testdata.AsEvent("workflow-job-completed-1.json")
		if err != nil {
			t.Fatal(err)
		}

		if err := handler.handleSource(context.TODO(), slog.Default(), team, team.Sources[0], event); err != nil {
			t.Error(err)
		}

		slack.Ensure(t, event.GetEventType(), 0, 0, 0)
	})

	t.Run("waiting job from bot is ignored when ignoreBots is set", func(t *testing.T) {
		db := &mock.Database{}
		slack := &mock.Slack{}
		botTeam := team
		botTeam.Sources = []github.Source{{
			SourceType: "workflows",
			Channel:    "#test",
			Config:     github.SourceConfig{Workflows: github.Workflows{IgnoreBots: true}},
		}}
		handler := NewHandler(db, slack, map[string]github.Team{"test": botTeam})

		event, err := testdata.AsEvent("workflow-job-waiting-1.json")
		if err != nil {
			t.Fatal(err)
		}
		event.Sender = github.User{Login: "dependabot[bot]", Type: "Bot"}

		if err := handler.handleSource(context.TODO(), slog.Default(), botTeam, botTeam.Sources[0], event); err != nil {
			t.Error(err)
		}

		slack.Ensure(t, event.GetEventType(), 0, 0, 0)
	})
}
