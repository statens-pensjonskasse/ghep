package slack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/sql"
)

func CreateWorkflowMessage(ctx context.Context, log *slog.Logger, db sql.Database, channel string, pingSlack bool, event github.Event) *Message {
	text := fmt.Sprintf(":x: %s has a workflow with status `%s`, triggered by %s.\n<%s|#%d %s>", event.Repository.ToSlack(), event.Workflow.Conclusion, mention(ctx, log, db, pingSlack, event.Sender), event.Workflow.URL, event.Workflow.RunNumber, event.Workflow.Title)

	var attachments []Attachment
	if event.Workflow.FailedJob.Name != "" {
		attachments = append(attachments, Attachment{
			Text:       fmt.Sprintf("The job <%s|%s>[%s] failed in step `%s`", event.Workflow.FailedJob.URL, event.Workflow.FailedJob.Name, event.Workflow.HeadBranch, event.Workflow.FailedJob.Step),
			Color:      ColorFailed,
			Footer:     fmt.Sprintf("<%s|%s>", event.Repository.URL, event.Repository.FullName),
			FooterIcon: "https://slack.github.com/static/img/favicon-neutral.png",
		})
	}

	return &Message{
		Channel:     channel,
		Text:        text,
		Attachments: attachments,
	}
}
