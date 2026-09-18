package slack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/sql"
)

func CreateTeamMessage(ctx context.Context, log *slog.Logger, db sql.Database, channel string, pingSlack bool, event github.Event) *Message {
	var text string

	switch event.Action {
	case "added_to_repository":
		text = fmt.Sprintf("Team %s was added to the repository %s", event.Team.ToSlack(), event.Repository.ToSlack())
	case "removed_from_repository":
		text = fmt.Sprintf("Team %s was removed from the repository %s", event.Team.ToSlack(), event.Repository.ToSlack())
	case "added":
		text = fmt.Sprintf("%s was added to the team %s", mention(ctx, log, db, pingSlack, event.Member), event.Team.ToSlack())
	case "removed":
		text = fmt.Sprintf("%s was removed from the team %s", mention(ctx, log, db, pingSlack, event.Member), event.Team.ToSlack())
	}

	return &Message{
		Channel: channel,
		Text:    text,
	}
}
