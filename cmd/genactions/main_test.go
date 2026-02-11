package main

import (
	"go/ast"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBindDirectives_RejectsUnknownKey(t *testing.T) {
	doc := &ast.CommentGroup{List: []*ast.Comment{{Text: "//jjui:bind scope=revisions.squash action=apply nope=x"}}}
	parsed := parseBindDirectives(doc, "Apply")
	require.Len(t, parsed, 1)
	require.NotEmpty(t, parsed[0].Errs)
}

func TestValidateRules_RejectsTypeMismatch(t *testing.T) {
	rules := []bindRule{{
		Owner:  "revisions.squash",
		Action: "apply",
		Intent: "Apply",
		Set:    map[string]string{"Force": "1"},
	}}
	intents := map[string]intentTypeMeta{"Apply": {Fields: map[string]string{"Force": "bool"}}}
	err := validateRules(rules, intents, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected bool")
}

func TestValidateRules_RejectsInvalidScopeFormat(t *testing.T) {
	rules := []bindRule{{Owner: "ui..global", Action: "apply", Intent: "Apply"}}
	intents := map[string]intentTypeMeta{"Apply": {Fields: map[string]string{}}}
	err := validateRules(rules, intents, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid owner")
}

func TestValidateRules_RejectsInvalidActionFormat(t *testing.T) {
	rules := []bindRule{{Owner: "ui", Action: "set.target", Intent: "Apply"}}
	intents := map[string]intentTypeMeta{"Apply": {Fields: map[string]string{}}}
	err := validateRules(rules, intents, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid action")
}

func TestValidateRules_RejectsDuplicateFullActionID(t *testing.T) {
	rules := []bindRule{
		{Owner: "ui", Action: "apply", Intent: "Apply"},
		{Owner: "ui", Action: "apply", Intent: "Apply"},
	}
	intents := map[string]intentTypeMeta{"Apply": {Fields: map[string]string{}}}
	err := validateRules(rules, intents, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate full action id")
}

func TestValidateActionMetadata_RejectsMissingOwnersOrFallback(t *testing.T) {
	err := validateActionMetadata([]string{"apply", ""}, map[string][]string{"apply": []string{"ui"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no owners")
	require.Contains(t, err.Error(), "empty description fallback")
}

func TestGeneratedCatalogIsUpToDate(t *testing.T) {
	root := repoRoot(t)

	intents, err := collectIntentTypeMeta(filepath.Join(root, "internal/ui/intents"))
	require.NoError(t, err)

	rules, err := collectBindRules(filepath.Join(root, "internal/ui/intents"))
	require.NoError(t, err)
	enums, err := collectEnumTypeMeta(filepath.Join(root, "internal/ui/intents"))
	require.NoError(t, err)

	err = validateRules(rules, intents, enums)
	require.NoError(t, err)
	actionIDs := deriveActionIDs(rules)

	generated, err := generateCatalogSource(rules, actionIDs, intents, enums)
	require.NoError(t, err)

	current, err := os.ReadFile(filepath.Join(root, "internal/ui/actions/catalog_gen.go"))
	require.NoError(t, err)

	require.Equal(t, string(current), string(generated), "generated catalog is stale; run `go run ./cmd/genactions`")

	schemas, requiredArgs, err := deriveActionArgSchemas(rules, intents, enums)
	require.NoError(t, err)
	owners := deriveActionOwners(rules)
	err = validateActionMetadata(actionIDs, owners)
	require.NoError(t, err)
	metaGenerated, err := generateActionMetaSource(actionIDs, schemas, requiredArgs, owners)
	require.NoError(t, err)
	metaCurrent, err := os.ReadFile(filepath.Join(root, "internal/ui/actionmeta/builtins_gen.go"))
	require.NoError(t, err)
	require.Equal(t, string(metaCurrent), string(metaGenerated), "generated action meta is stale; run `go run ./cmd/genactions`")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
