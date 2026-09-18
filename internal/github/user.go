package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/navikt/ghep/internal/sql/gensql"
)

const (
	graphqlEndpoint = "https://api.github.com/graphql"
	// membersWithEmailGraphQL henter alle medlemmer i organisasjonen med de e-postadressene
	// en GitHub App med "members: read" får se: den offentlige profil-e-posten og alle
	// verifiserte e-poster på organisasjonens verifiserte domener. SAML-identiteter
	// (samlIdentityProvider.externalIdentities) er ikke tilgjengelige når SAML er satt opp
	// på enterprise-nivå, slik det er i SPK.
	membersWithEmailGraphQL = `query FetchMembersWithEmail($org: String!, $cursor: String) {
	 organization(login: $org) {
	   membersWithRole(first: 100, after: $cursor) {
	     nodes {
	       login
	       email
	       organizationVerifiedDomainEmails(login: $org)
	     }
	     pageInfo {
	       hasNextPage
	       endCursor
	     }
	   }
	 }
	}`
)

type membersWithEmailResponse struct {
	Data struct {
		Organization struct {
			MembersWithRole struct {
				Nodes []struct {
					Login                            string   `json:"login"`
					Email                            string   `json:"email"`
					OrganizationVerifiedDomainEmails []string `json:"organizationVerifiedDomainEmails"`
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"membersWithRole"`
		} `json:"organization"`
	} `json:"data"`
	Errors []struct {
		Type    string   `json:"type"`
		Path    []string `json:"path"`
		Message string   `json:"message"`
	} `json:"errors"`
}

// memberEmails er et organisasjonsmedlem med alle kjente e-postadresser, uten duplikater og tomme verdier.
type memberEmails struct {
	Login  string
	Emails []string
}

// FetchOrgUsersWithEmail lagrer alle medlemmer i organisasjonen og e-postadressene deres,
// slik at Slack-brukere senere kan kobles til GitHub-brukere på e-post.
func (c *Client) FetchOrgUsersWithEmail(ctx context.Context) error {
	bearerToken, err := c.createBearerToken()
	if err != nil {
		return fmt.Errorf("creating bearer token: %v", err)
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	members, err := fetchOrgMembersWithEmails(ctx, httpClient, graphqlEndpoint, bearerToken, c.org)
	if err != nil {
		return err
	}

	for _, member := range members {
		if err := c.db.CreateUser(ctx, member.Login); err != nil {
			return fmt.Errorf("creating user %s: %w", member.Login, err)
		}

		for _, email := range member.Emails {
			if err := c.db.CreateEmail(ctx, gensql.CreateEmailParams{
				Login: member.Login,
				Email: email,
			}); err != nil {
				return fmt.Errorf("creating email for user %s: %w", member.Login, err)
			}
		}
	}

	return nil
}

// fetchOrgMembersWithEmails henter alle medlemmer sidevis fra GraphQL-endepunktet.
func fetchOrgMembersWithEmails(ctx context.Context, httpClient *http.Client, endpoint, bearerToken, org string) ([]memberEmails, error) {
	var members []memberEmails
	cursor := ""
	for {
		variables := map[string]any{"org": org}
		if cursor != "" {
			variables["cursor"] = cursor
		}

		body, err := json.Marshal(map[string]any{
			"query":     membersWithEmailGraphQL,
			"variables": variables,
		})
		if err != nil {
			return nil, fmt.Errorf("marshalling query: %v", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", bearerToken))
		req.Header.Add("Content-Type", "application/json")

		httpResp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("doing request: %v", err)
		}

		var resp membersWithEmailResponse
		err = json.NewDecoder(httpResp.Body).Decode(&resp)
		httpResp.Body.Close() // #nosec G104 -- closing response body, error intentionally ignored
		if err != nil {
			return nil, fmt.Errorf("decoding response: %v", err)
		}

		if httpResp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error fetching members (%v): %s", httpResp.Status, resp.Errors)
		}

		if len(resp.Errors) > 0 {
			var b strings.Builder
			for _, err := range resp.Errors {
				fmt.Fprintf(&b, "%s (type=%s, path=[%s])\n", err.Message, err.Type, strings.Join(err.Path, " "))
			}

			return nil, fmt.Errorf("graphql error: %s", b.String())
		}

		connection := resp.Data.Organization.MembersWithRole
		for _, node := range connection.Nodes {
			members = append(members, memberEmails{
				Login:  node.Login,
				Emails: uniqueNonEmpty(append([]string{node.Email}, node.OrganizationVerifiedDomainEmails...)),
			})
		}

		if !connection.PageInfo.HasNextPage {
			return members, nil
		}
		cursor = connection.PageInfo.EndCursor
	}
}

// uniqueNonEmpty fjerner tomme strenger og duplikater (case-insensitivt), og beholder rekkefølgen.
func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		key := strings.ToLower(v)
		if v == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, v)
	}

	return result
}
