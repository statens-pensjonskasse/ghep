package slack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/sql"
)

func CreateReleaseMessage(ctx context.Context, log *slog.Logger, db sql.Database, channel string, pingSlack bool, event github.Event) *Message {
	releaseType := "release"
	if event.Release.Draft {
		releaseType = "draft release"
	} else if event.Release.Prerelease {
		releaseType = "prerelease"
	}

	text := fmt.Sprintf("%s created a <%s|%s> (`%s`) in %s", mention(ctx, log, db, pingSlack, event.Sender), event.Release.URL, releaseType, event.Release.Tag, event.Repository.ToSlack())

	return &Message{
		Channel: channel,
		Text:    text,
		Attachments: []Attachment{
			{
				Text:       event.Release.Body,
				Color:      ColorDefault,
				FooterIcon: neutralGithubIcon,
				Footer:     fmt.Sprintf("<%s|%s>", event.Repository.URL, event.Repository.FullName),
			},
		},
	}
}
