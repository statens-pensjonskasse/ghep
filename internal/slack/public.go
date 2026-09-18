package slack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/sql"
)

func CreatePublicizedMessage(ctx context.Context, log *slog.Logger, db sql.Database, channel string, pingSlack bool, event github.Event) *Message {
	return &Message{
		Channel: channel,
		Text:    fmt.Sprintf("%s made %s public!", mention(ctx, log, db, pingSlack, event.Sender), event.Repository.ToSlack()),
	}
}
