package config

import "testing"

func TestServiceCredentialsValidate(t *testing.T) {
	t.Parallel()

	creds := ServiceCredentials{Email: "user@example.com", APIToken: "token"}
	if err := creds.validate("jira"); err != nil {
		t.Fatalf("expected credentials to be valid, got %v", err)
	}

	creds = ServiceCredentials{OAuthToken: "token"}
	if err := creds.validate("jira"); err != nil {
		t.Fatalf("expected oauth credentials to be valid, got %v", err)
	}

	creds = ServiceCredentials{Email: "user@example.com"}
	if err := creds.validate("jira"); err == nil {
		t.Fatalf("expected error for incomplete credentials")
	}
}
