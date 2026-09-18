package slack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/sql"
)

func CreateSecretScanningAlertMessage(ctx context.Context, log *slog.Logger, db sql.Database, channel, timestamp string, pingSlack bool, event github.Event) *Message {
	var text string
	switch event.Action {
	case "created":
		text = fmt.Sprintf("A secret scanning alert was just created for the repository %s.\nThe secret was of type `%s`.\nRead more: %s", event.Repository.ToSlack(), *event.Alert.SecretType, event.Alert.URL)
	case "resolved":
		resolvedBy := mention(ctx, log, db, pingSlack, event.Alert.ResolvedBy)
		if event.Alert.PubliclyLeaked {
			text = fmt.Sprintf("A public secret scanning alert was just resolved for the repository %s. It was resolved as `%s` by %s.\n> %s", event.Repository.ToSlack(), event.Alert.Resolution, resolvedBy, event.Alert.ResolutionComment)
		} else {
			text = fmt.Sprintf("A secret scanning alert was just resolved for the repository %s. It was resolved as `%s` by %s.\n> %s", event.Repository.ToSlack(), event.Alert.Resolution, resolvedBy, event.Alert.ResolutionComment)
		}
	}

	var attachments []Attachment
	if event.Alert.PubliclyLeaked && event.Action == "created" {
		attachment := Attachment{
			Text:  "The secret was publicly leaked!",
			Color: ColorCritical,
		}
		attachments = append(attachments, attachment)
	}

	return &Message{
		Channel:         channel,
		Text:            text,
		Attachments:     attachments,
		ThreadTimestamp: timestamp,
	}
}
