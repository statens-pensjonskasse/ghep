package events

import (
	"context"
	"log/slog"

	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/slack"
	"github.com/navikt/ghep/internal/sql"
)

func handleRenamedEvent(ctx context.Context, log *slog.Logger, db sql.Database, channel string, pingSlack bool, event github.Event) *slack.Message {
	log.Info("Posting renamed repository message", "channel", channel)
	return slack.CreateRenamedMessage(ctx, log, db, channel, pingSlack, event)
}
