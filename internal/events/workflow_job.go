package events

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/navikt/ghep/internal/github"
	"github.com/navikt/ghep/internal/slack"
	"github.com/navikt/ghep/internal/sql/gensql"
)

// workflowJobEventID er nøkkelen ventemeldingen lagres under, slik at senere
// workflow_job-hendelser for samme jobb kan oppdatere den.
func workflowJobEventID(job *github.WorkflowJob) string {
	return "workflow_job/" + strconv.Itoa(job.ID)
}

// handleWorkflowJobEvent poster en melding når en jobb venter på godkjenning av et
// deployment-miljø, og oppdaterer meldingen når jobben starter eller fullfører.
// Jobber som aldri ventet gir ingen melding.
func (h *Handler) handleWorkflowJobEvent(ctx context.Context, log *slog.Logger, team github.Team, source github.Source, event github.Event) (*slack.Message, error) {
	job := event.WorkflowJob

	if source.Config.Workflows.IgnoreBots && event.Sender.IsBot() {
		return nil, nil
	}

	if len(source.Config.Workflows.Repositories) > 0 && !slices.Contains(source.Config.Workflows.Repositories, event.Repository.Name) {
		return nil, nil
	}

	if len(source.Config.Workflows.Workflows) > 0 && !slices.Contains(source.Config.Workflows.Workflows, job.WorkflowName) {
		return nil, nil
	}

	if event.Action == "waiting" {
		log.Info("Workflow job is waiting for approval", "workflow", job.WorkflowName, "job", job.Name, "run_id", job.RunID, "channel", source.Channel)
		return slack.CreateWorkflowJobMessage(ctx, log, h.db, source.Channel, team.Config.PingSlackUsers, event), nil
	}

	stored, err := h.db.GetSlackMessage(ctx, gensql.GetSlackMessageParams{
		TeamSlug: team.Name,
		EventID:  workflowJobEventID(job),
		Channel:  source.Channel,
	})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Error("Getting workflow job message", "error", err, "job_id", job.ID)
		}
		return nil, nil
	}

	if stored.ThreadTs == "" {
		return nil, nil
	}

	updated := slack.CreateWorkflowJobMessage(ctx, log, h.db, source.Channel, team.Config.PingSlackUsers, event)
	updated.Timestamp = stored.ThreadTs

	log.Info("Updating workflow job message", "action", event.Action, "conclusion", job.Conclusion, "timestamp", stored.ThreadTs)
	if err := h.slack.PostUpdatedMessage(*updated); err != nil {
		log.Error("Updating workflow job message", "error", err, "channel", source.Channel, "timestamp", stored.ThreadTs)
	}

	return nil, nil
}
