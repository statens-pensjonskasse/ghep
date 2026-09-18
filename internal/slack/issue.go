package slack

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/sql"
)

func CreateIssueMessage(ctx context.Context, log *slog.Logger, db sql.Database, channel, threadTimestamp string, pingSlack bool, event github.Event) *Message {
	color := ColorOpened
	sender := mention(ctx, log, db, pingSlack, event.Sender)

	text := fmt.Sprintf("Issue <%s|#%d> %s in `%s` by %s", event.Issue.URL, event.Issue.Number, event.Action, event.Repository.ToSlack(), sender)
	attachmentText := fmt.Sprintf("*<%s|#%d %s>*", event.Issue.URL, event.Issue.Number, html.EscapeString(event.Issue.Title))

	if event.Action == "closed" {
		color = ColorMerged
		text = fmt.Sprintf("Issue <%s|#%d> %s as %s in `%s` by %s", event.Issue.URL, event.Issue.Number, event.Action, event.Issue.StateReason, event.Repository.ToSlack(), sender)
	}

	if event.Action != "closed" && event.Issue.Body != "" {
		attachmentText = fmt.Sprintf("%s\n%s", attachmentText, event.Issue.Body)
	}

	if len(event.Issue.Assignees) > 0 {
		assignees := make([]string, len(event.Issue.Assignees))
		for i, assignee := range event.Issue.Assignees {
			assignees[i] = mention(ctx, log, db, pingSlack, assignee)
		}

		attachmentText += fmt.Sprintf("\n*Assignees:* %s", strings.Join(assignees, ", "))
	}

	return &Message{
		Channel:         channel,
		ThreadTimestamp: threadTimestamp,
		Text:            text,
		Attachments: []Attachment{
			{
				Text:       attachmentText,
				Type:       "mrkdwn",
				Color:      color,
				FooterIcon: neutralGithubIcon,
				Footer:     fmt.Sprintf("<%s|%s>", event.Repository.URL, event.Repository.FullName),
			},
		},
	}
}
