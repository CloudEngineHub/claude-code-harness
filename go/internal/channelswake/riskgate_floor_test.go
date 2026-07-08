package channelswake

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/runtimefloor"
)

func TestRiskGateFiveCategoryFloorUnchanged(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	srcPath := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "runtimefloor", "runtimefloor.go"))
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)

	categories := []string{
		string(runtimefloor.CategoryMoneyBilling),
		string(runtimefloor.CategoryEgress),
		string(runtimefloor.CategorySecretRead),
		string(runtimefloor.CategoryProdDeploy),
		string(runtimefloor.CategoryWorktreeEscape),
	}
	// grep-friendly aliases used in distribution contract docs/tests
	aliases := map[string]string{
		string(runtimefloor.CategoryMoneyBilling):   "money",
		string(runtimefloor.CategoryEgress):         "egress",
		string(runtimefloor.CategorySecretRead):     "secret",
		string(runtimefloor.CategoryProdDeploy):     "prod-deploy",
		string(runtimefloor.CategoryWorktreeEscape): "worktree-escape",
	}

	for _, cat := range categories {
		if !strings.Contains(body, cat) {
			t.Fatalf("runtimefloor.go missing category constant %q", cat)
		}
		if alias := aliases[cat]; !strings.Contains(body, alias) {
			t.Fatalf("runtimefloor.go missing grep alias %q for category %q", alias, cat)
		}
	}
}
