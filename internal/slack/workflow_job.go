package slack

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/sql"
)

// CreateWorkflowJobMessage lager meldingen for en jobb som venter på godkjenning, og de
// oppdaterte variantene når jobben starter eller fullfører. Den som utløste kjøringen pinges.
func CreateWorkflowJobMessage(ctx context.Context, log *slog.Logger, db sql.Database, channel string, pingSlack bool, event github.Event) *Message {
	job := event.WorkflowJob

	what := fmt.Sprintf("Job `%s`", job.Name)
	if event.Deployment != nil && event.Deployment.Environment != "" {
		what = fmt.Sprintf("Deployment to `%s`", event.Deployment.Environment)
	}

	triggeredBy := event.Sender.ToSlack()
	if pingSlack {
		userID, err := db.GetUserSlackID(ctx, event.Sender.Login)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Error("Getting user Slack ID", "user", event.Sender.Login, "error", err)
		}

		if userID != "" {
			triggeredBy = fmt.Sprintf("<@%s>", userID)
		}
	}

	var icon, verb, color string
	switch event.Action {
	case "waiting":
		icon, verb, color = ":hourglass_flowing_sand:", "is waiting for approval", ColorMedium
	case "in_progress":
		icon, verb, color = ":arrow_forward:", "was approved and is running", ColorDefault
	case "completed":
		switch job.Conclusion {
		case "success":
			icon, verb, color = ":white_check_mark:", "completed successfully", ColorOpened
		case "failure":
			icon, verb, color = ":x:", "failed", ColorFailed
		case "cancelled":
			icon, verb, color = ":no_entry_sign:", "was cancelled", ColorDraft
		default:
			icon, verb, color = ":grey_question:", fmt.Sprintf("finished with conclusion `%s`", job.Conclusion), ColorDefault
		}
	default:
		icon, verb, color = ":grey_question:", fmt.Sprintf("is `%s`", event.Action), ColorDefault
	}

	text := fmt.Sprintf("%s %s in %s %s, triggered by %s.", icon, what, event.Repository.ToSlack(), verb, triggeredBy)

	return &Message{
		Channel: channel,
		Text:    text,
		Attachments: []Attachment{
			{
				Text:       fmt.Sprintf("<%s|%s / %s> [%s]", job.RunURL(event.Repository), job.WorkflowName, job.Name, job.HeadBranch),
				Type:       "mrkdwn",
				Color:      color,
				FooterIcon: neutralGithubIcon,
				Footer:     fmt.Sprintf("<%s|%s>", event.Repository.URL, event.Repository.FullName),
			},
		},
	}
}
