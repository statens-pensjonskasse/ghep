package events

import (
	"context"
	"log/slog"

	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/slack"
	"github.com/navikt/ghep/internal/sql"
)

func handlePublicizedEvent(ctx context.Context, log *slog.Logger, db sql.Database, channel string, pingSlack bool, event github.Event) *slack.Message {
	log.Info("Received repository publicized", "channel", channel)
	return slack.CreatePublicizedMessage(ctx, log, db, channel, pingSlack, event)
}
