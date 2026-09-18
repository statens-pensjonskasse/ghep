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

// mention gir Slack-omtalen av en GitHub-bruker. Alle meldinger som nevner en bruker skal gå
// gjennom denne. Med pingSlack slås brukeren opp i slack_ids og omtales som <@ID>, slik at
// vedkommende varsles. Finnes ingen kobling, logges det og lenken til GitHub-profilen brukes.
// Bots slås ikke opp.
func mention(ctx context.Context, log *slog.Logger, db sql.Database, pingSlack bool, user github.User) string {
	if !pingSlack || user.Login == "" || user.IsBot() {
		return user.ToSlack()
	}

	userID, err := db.GetUserSlackID(ctx, user.Login)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Error("Getting user Slack ID", "user", user.Login, "error", err)
		return user.ToSlack()
	}

	if userID == "" {
		log.Warn("No Slack user mapped for GitHub user", "user", user.Login)
		return user.ToSlack()
	}

	return fmt.Sprintf("<@%s>", userID)
}

// mentionAuthor gjør det samme for en commit-forfatter. Commit-forfattere og co-authors har ofte
// bare navn og e-post; da slås e-posten opp mot e-postene lagret fra GitHub for å finne brukernavnet.
func mentionAuthor(ctx context.Context, log *slog.Logger, db sql.Database, pingSlack bool, author github.Author) string {
	user := author.AsUser()
	if !pingSlack || author.Username != "" || author.IsBot() {
		return mention(ctx, log, db, pingSlack, user)
	}

	if author.Email != "" {
		login, err := db.GetUserByEmail(ctx, author.Email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Error("Getting user by email", "email", author.Email, "error", err)
		}

		if login != "" {
			user.Login = login
			user.URL = "https://github.com/" + login
			return mention(ctx, log, db, pingSlack, user)
		}
	}

	log.Warn("No GitHub user mapped for commit author", "name", author.Name, "email", author.Email)
	return user.ToSlack()
}
