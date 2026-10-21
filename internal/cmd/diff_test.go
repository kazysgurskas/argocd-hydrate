package cmd

import "testing"

func TestPRNumberFromRef(t *testing.T) {
	tests := map[string]int{
		"refs/pull/42/merge": 42,
		"refs/pull/7/head":   7,
		"refs/heads/master":  0,
		"refs/pull/x/merge":  0,
		"refs/pull/42":       0,
		"":                   0,
	}
	for ref, want := range tests {
		t.Run(ref, func(t *testing.T) {
			if got := prNumberFromRef(ref); got != want {
				t.Errorf("prNumberFromRef(%q) = %d, want %d", ref, got, want)
			}
		})
	}
}

func TestPRNumberFromEnv(t *testing.T) {
	t.Setenv("GITHUB_PR_NUMBER", "")
	t.Setenv("GITHUB_REF", "refs/pull/314228/merge")
	if got := prNumberFromEnv(); got != 314228 {
		t.Errorf("from GITHUB_REF: %d", got)
	}
	t.Setenv("GITHUB_PR_NUMBER", "9")
	if got := prNumberFromEnv(); got != 9 {
		t.Errorf("GITHUB_PR_NUMBER should win: %d", got)
	}
}
