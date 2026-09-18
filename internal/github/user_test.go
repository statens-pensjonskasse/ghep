package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// roundTripFunc lar testen svare på HTTP-kall uten å åpne en port.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetchOrgMembersWithEmails(t *testing.T) {
	pages := map[string]string{
		// Første side: ingen cursor.
		"": `{"data":{"organization":{"membersWithRole":{
			"nodes":[
				{"login":"kari","email":"","organizationVerifiedDomainEmails":["Kari.Nordmann@spk.no","kno@spk.no"]},
				{"login":"ola","email":"ola@example.com","organizationVerifiedDomainEmails":["ola.nordmann@spk.no"]}
			],
			"pageInfo":{"hasNextPage":true,"endCursor":"side2"}}}}}`,
		// Andre side: duplikat mellom offentlig e-post og verifisert e-post, ulik casing.
		"side2": `{"data":{"organization":{"membersWithRole":{
			"nodes":[
				{"login":"per","email":"per.hansen@spk.no","organizationVerifiedDomainEmails":["Per.Hansen@spk.no"]},
				{"login":"uten-epost","email":"","organizationVerifiedDomainEmails":[]}
			],
			"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
	}

	var gotVariables []map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer token")
		}

		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		gotVariables = append(gotVariables, payload.Variables)

		cursor, _ := payload.Variables["cursor"].(string)
		page, ok := pages[cursor]
		if !ok {
			t.Fatalf("unexpected cursor %q", cursor)
		}

		return jsonResponse(http.StatusOK, page), nil
	})}

	got, err := fetchOrgMembersWithEmails(context.Background(), httpClient, "https://example.invalid/graphql", "token", "spk")
	if err != nil {
		t.Fatal(err)
	}

	want := []memberEmails{
		{Login: "kari", Emails: []string{"Kari.Nordmann@spk.no", "kno@spk.no"}},
		{Login: "ola", Emails: []string{"ola@example.com", "ola.nordmann@spk.no"}},
		{Login: "per", Emails: []string{"per.hansen@spk.no"}},
		{Login: "uten-epost", Emails: []string{}},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("fetchOrgMembersWithEmails mismatch (-want +got):\n%s", diff)
	}

	wantVariables := []map[string]any{
		{"org": "spk"},
		{"org": "spk", "cursor": "side2"},
	}
	if diff := cmp.Diff(wantVariables, gotVariables); diff != "" {
		t.Errorf("GraphQL variables mismatch (-want +got):\n%s", diff)
	}
}

func TestFetchOrgMembersWithEmailsGraphQLError(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"data":{"organization":null},"errors":[{"type":"FORBIDDEN","path":["organization"],"message":"Resource not accessible by integration"}]}`), nil
	})}

	_, err := fetchOrgMembersWithEmails(context.Background(), httpClient, "https://example.invalid/graphql", "token", "spk")
	if err == nil {
		t.Fatal("expected error from GraphQL errors, got nil")
	}
}

func TestFetchOrgMembersWithEmailsHTTPError(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, `{"message":"Bad credentials"}`), nil
	})}

	_, err := fetchOrgMembersWithEmails(context.Background(), httpClient, "https://example.invalid/graphql", "token", "spk")
	if err == nil {
		t.Fatal("expected error from HTTP 401, got nil")
	}
}

func TestUniqueNonEmpty(t *testing.T) {
	got := uniqueNonEmpty([]string{"", "A@spk.no", "a@spk.no", "b@spk.no", "", "B@SPK.NO"})
	want := []string{"A@spk.no", "b@spk.no"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("uniqueNonEmpty mismatch (-want +got):\n%s", diff)
	}
}
