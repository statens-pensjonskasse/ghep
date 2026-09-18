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

func CreatePullRequestMessage(ctx context.Context, log *slog.Logger, db sql.Database, channel, threadTimestamp string, pingSlack, minimalist bool, event github.Event) *Message {
	color := ColorOpened
	switch event.Action {
	case "merged":
		color = ColorMerged
	case "closed":
		color = ColorClosed
	case "ready_for_review":
		event.Action = "was marked ready for review"
	}

	eventType := "Pull request"
	if event.PullRequest.Draft {
		eventType = "Draft pull request"
		color = ColorDraft
	}

	sender := mention(ctx, log, db, pingSlack, event.Sender)

	text := ""
	attachments := []Attachment{}
	if minimalist {
		text = fmt.Sprintf("%s <%s|#%d %s> %s in `%s` by %s", eventType, event.PullRequest.URL, event.PullRequest.Number, html.EscapeString(event.PullRequest.Title), event.Action, event.Repository.ToSlack(), sender)
	} else {
		text = fmt.Sprintf("%s <%s|#%d> %s in `%s` by %s", eventType, event.PullRequest.URL, event.PullRequest.Number, event.Action, event.Repository.ToSlack(), sender)
		attachmentText := fmt.Sprintf("*<%s|#%d %s>*", event.PullRequest.URL, event.PullRequest.Number, html.EscapeString(event.PullRequest.Title))

		if event.Action != "closed" && event.PullRequest.Body != "" {
			attachmentText = fmt.Sprintf("%s\n%s", attachmentText, event.PullRequest.Body)
		}

		if len(event.PullRequest.RequestedReviewers) > 0 {
			reviewers := make([]string, len(event.PullRequest.RequestedReviewers))
			for i, reviewer := range event.PullRequest.RequestedReviewers {
				reviewers[i] = mention(ctx, log, db, pingSlack, reviewer)
			}

			attachmentText += fmt.Sprintf("\n*Requested reviewers:* %s", strings.Join(reviewers, ", "))
		}

		attachments = []Attachment{
			{
				Text:       attachmentText,
				Type:       "mrkdwn",
				Color:      color,
				FooterIcon: neutralGithubIcon,
				Footer:     fmt.Sprintf("<%s|%s>", event.Repository.URL, event.Repository.FullName),
			},
		}
	}

	return &Message{
		Channel:         channel,
		ThreadTimestamp: threadTimestamp,
		Text:            text,
		Attachments:     attachments,
	}
}
