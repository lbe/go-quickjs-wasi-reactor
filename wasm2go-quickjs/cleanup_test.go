package quickjs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWazeroFullyRemoved verifies that wazero has been fully removed from the
// project and all repo artifacts have been updated to reference wasm2go instead.
func TestWazeroFullyRemoved(t *testing.T) {
	repoRoot := findRepoRoot(t)

	// 1. wazero-quickjs/ directory must not exist
	t.Run("wazero_quickjs_dir_removed", func(t *testing.T) {
		wazeroDir := filepath.Join(repoRoot, "wazero-quickjs")
		_, err := os.Stat(wazeroDir)
		if err == nil {
			t.Fatalf("wazero-quickjs/ directory still exists at %s and should have been removed", wazeroDir)
		}
		if !os.IsNotExist(err) {
			t.Fatalf("unexpected error checking wazero-quickjs/: %v", err)
		}
		t.Log("wazero-quickjs/ directory does not exist (correct)")
	})

	// 2. No .go file in the project imports wazero
	t.Run("no_wazero_imports", func(t *testing.T) {
		wazeroImports := []string{
			"github.com/tetratelabs/wazero",
			"github.com/aperturerobotics/wazero",
		}
		violations := []string{}
		err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				// Skip vendor and hidden directories
				if d.Name() == "vendor" || strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// Skip this test file itself (it references wazero import paths in its checks)
			if strings.HasSuffix(path, "cleanup_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)
			for _, imp := range wazeroImports {
				if strings.Contains(content, imp) {
					violations = append(violations, path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("error walking repo: %v", err)
		}
		if len(violations) > 0 {
			t.Fatalf("found wazero imports in %d file(s): %s", len(violations), strings.Join(violations, ", "))
		}
		t.Log("no wazero imports found in any .go files (correct)")
	})

	// 3. Root go.mod must not contain wazero dependency
	t.Run("root_go_mod_no_wazero", func(t *testing.T) {
		goModPath := filepath.Join(repoRoot, "go.mod")
		data, err := os.ReadFile(goModPath)
		if err != nil {
			t.Fatalf("failed to read root go.mod: %v", err)
		}
		content := string(data)
		wazeroDeps := []string{
			"github.com/tetratelabs/wazero",
			"github.com/aperturerobotics/wazero",
		}
		for _, dep := range wazeroDeps {
			if strings.Contains(content, dep) {
				t.Fatalf("root go.mod still contains wazero dependency: %s", dep)
			}
		}
		t.Log("root go.mod does not contain wazero dependencies (correct)")
	})

	// 4. CI tests.yml must reference wasm2go-quickjs, not wazero-quickjs
	t.Run("ci_tests_yml_updated", func(t *testing.T) {
		testsYml := filepath.Join(repoRoot, ".github", "workflows", "tests.yml")
		data, err := os.ReadFile(testsYml)
		if err != nil {
			t.Fatalf("failed to read tests.yml: %v", err)
		}
		content := string(data)
		if strings.Contains(content, "wazero-quickjs") {
			t.Fatalf("tests.yml still references wazero-quickjs; should reference wasm2go-quickjs")
		}
		if !strings.Contains(content, "wasm2go-quickjs") {
			t.Fatalf("tests.yml does not reference wasm2go-quickjs")
		}
		t.Log("tests.yml references wasm2go-quickjs (correct)")
	})

	// 5. CI update-quickjs.yml must reference wasm2go-quickjs, not wazero-quickjs
	t.Run("ci_update_quickjs_yml_updated", func(t *testing.T) {
		updateYml := filepath.Join(repoRoot, ".github", "workflows", "update-quickjs.yml")
		data, err := os.ReadFile(updateYml)
		if err != nil {
			t.Fatalf("failed to read update-quickjs.yml: %v", err)
		}
		content := string(data)
		if strings.Contains(content, "wazero-quickjs") {
			t.Fatalf("update-quickjs.yml still references wazero-quickjs; should reference wasm2go-quickjs")
		}
		if !strings.Contains(content, "wasm2go-quickjs") {
			t.Fatalf("update-quickjs.yml does not reference wasm2go-quickjs")
		}
		t.Log("update-quickjs.yml references wasm2go-quickjs (correct)")
	})

	// 6. update-quickjs.bash must reference wasm2go, not wazero
	t.Run("update_quickjs_bash_updated", func(t *testing.T) {
		script := filepath.Join(repoRoot, "update-quickjs.bash")
		data, err := os.ReadFile(script)
		if err != nil {
			t.Fatalf("failed to read update-quickjs.bash: %v", err)
		}
		content := string(data)
		if strings.Contains(content, "wazero") {
			t.Fatalf("update-quickjs.bash still references wazero; should reference wasm2go")
		}
		t.Log("update-quickjs.bash does not reference wazero (correct)")
	})

	// 7. Root README.md must not reference wazero
	t.Run("readme_no_wazero", func(t *testing.T) {
		readme := filepath.Join(repoRoot, "README.md")
		data, err := os.ReadFile(readme)
		if err != nil {
			t.Fatalf("failed to read README.md: %v", err)
		}
		content := string(data)
		if strings.Contains(content, "wazero") {
			t.Fatalf("README.md still references wazero")
		}
		t.Log("README.md does not reference wazero (correct)")
	})

	// 8. embed.go must not contain QuickJSWASM or //go:embed directive
	t.Run("embed_go_cleaned", func(t *testing.T) {
		embed := filepath.Join(repoRoot, "embed.go")
		data, err := os.ReadFile(embed)
		if err != nil {
			t.Fatalf("failed to read embed.go: %v", err)
		}
		content := string(data)
		if strings.Contains(content, "QuickJSWASM") {
			t.Fatalf("embed.go still contains QuickJSWASM")
		}
		if strings.Contains(content, "//go:embed") {
			t.Fatalf("embed.go still contains //go:embed directive")
		}
		t.Log("embed.go does not contain QuickJSWASM or //go:embed (correct)")
	})

	// 9. embed_test.go must not test QuickJSWASM
	t.Run("embed_test_go_cleaned", func(t *testing.T) {
		embedTest := filepath.Join(repoRoot, "embed_test.go")
		data, err := os.ReadFile(embedTest)
		if err != nil {
			t.Fatalf("failed to read embed_test.go: %v", err)
		}
		content := string(data)
		if strings.Contains(content, "QuickJSWASM") {
			t.Fatalf("embed_test.go still tests QuickJSWASM")
		}
		t.Log("embed_test.go does not test QuickJSWASM (correct)")
	})
}

// findRepoRoot walks up from the test file's package directory to find the repo root.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	// The test runs in wasm2go-quickjs/, so the repo root is one level up.
	abs, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}
	repoRoot := filepath.Dir(abs)
	return repoRoot
}
