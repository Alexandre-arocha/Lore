package main

import "testing"

func TestGitHubTokenHintForAnonymousRateLimit(t *testing.T) {
	hint := githubTokenHint("", []checkResult{
		{Slug: "demo", Status: "fail", Error: "status 403: API rate limit exceeded"},
	})
	if hint == "" {
		t.Fatal("githubTokenHint returned empty hint for anonymous rate limit")
	}
}

func TestGitHubTokenHintSkipsWhenTokenIsSet(t *testing.T) {
	hint := githubTokenHint("token", []checkResult{
		{Slug: "demo", Status: "fail", Error: "status 403: API rate limit exceeded"},
	})
	if hint != "" {
		t.Fatalf("githubTokenHint = %q, want empty when token is set", hint)
	}
}
