package service_test

// GAP-016 DoD③ · pins the "invalid MYSQL_TEST_DATABASE must Fatalf, never skip"
// invariant (docs/GAP台账.md, GAP-016 DoD③).
//
// Why this file exists: the ledger carries a one-off manual command as the only
// evidence that an unsafe database name aborts the run. A one-off command is not
// a gate — nobody re-runs it. If someone "fixes" a red CI by turning the Fatalf
// at internal/service/test_helper_test.go:102 into a t.Skipf, every integration
// test in this package silently turns green and NOTHING fails. That is the exact
// failure mode D3 warns about ("failure is silent"), so it gets a resident test.
//
// Two independent pins, because neither alone is enough:
//
//	① safeDatabaseName table test — proves the regex classifies names correctly.
//	   It CANNOT prove that a rejected name actually aborts; it only looks at a
//	   bool. (The validator is a package-level var in test_helper_test.go, i.e. an
//	   internal test symbol; external test package service_test cannot see it, so
//	   the classification cases are replicated here over the source-extracted
//	   pattern. If the pattern is widened — e.g. to allow '-' — the table goes red.)
//	② AST pin — parses test_helper_test.go itself and asserts ensureTestDatabase
//	   rejects an invalid name with t.Fatalf. This is the half that pins the
//	   actual abort. Mutation-verified: changing :102 to t.Skipf must turn it red.
//
// ensureTestDatabase is deliberately NOT called: it opens a real MySQL connection
// (this is a unit test, and calling it would also hit the invalid-name branch only
// under a specific env, making the pin environment-dependent).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	qaDBNameHelperFile     = "test_helper_test.go"
	qaDBNameHelperFunc     = "ensureTestDatabase"
	qaDBNameValidatorIdent = "safeDatabaseName"
)

// qaDBNameSourcePath resolves the helper file relative to THIS file, not the
// process CWD: `go test` sets the CWD to the package dir today, but a pin that
// depends on CWD breaks the moment anything runs it differently (and it would
// break as a confusing parse error, not as an honest failure).
func qaDBNameSourcePath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot resolve caller path")
	}
	return filepath.Join(filepath.Dir(thisFile), qaDBNameHelperFile)
}

func qaParseDBNameHelper(t *testing.T) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, qaDBNameSourcePath(t), nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", qaDBNameHelperFile, err)
	}
	return fset, file
}

// qaSafeDatabaseNamePattern extracts the literal passed to regexp.MustCompile in
// the `var safeDatabaseName = regexp.MustCompile(...)` declaration of the helper
// file. It fails loudly if the declaration moved or stopped being a literal —
// silently returning an empty pattern would make the table test below vacuous.
func qaSafeDatabaseNamePattern(t *testing.T) string {
	t.Helper()
	_, file := qaParseDBNameHelper(t)

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if name.Name != qaDBNameValidatorIdent || i >= len(vs.Values) {
					continue
				}
				call, ok := vs.Values[i].(*ast.CallExpr)
				if !ok || len(call.Args) != 1 {
					t.Fatalf("%s is no longer initialized by a single-argument call", qaDBNameValidatorIdent)
				}
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Fatalf("%s pattern is no longer a string literal; this pin cannot read it statically", qaDBNameValidatorIdent)
				}
				pattern, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s pattern %s: %v", qaDBNameValidatorIdent, lit.Value, err)
				}
				if pattern == "" {
					t.Fatalf("%s pattern is empty: an empty regexp matches everything and would disable the guard", qaDBNameValidatorIdent)
				}
				return pattern
			}
		}
	}
	t.Fatalf("%s declaration not found in %s", qaDBNameValidatorIdent, qaDBNameHelperFile)
	return "" // unreachable
}

// TestQADatabaseNameClassification pins the accept/reject table of the database
// name guard. The pattern is read from the source, so widening it (for example
// to admit '-', which is exactly how CI's old
// `go-order-management-system_test` slipped in) makes the invalid cases fail.
func TestQADatabaseNameClassification(t *testing.T) {
	re := regexp.MustCompile(qaSafeDatabaseNamePattern(t))

	cases := []struct {
		name  string
		input string
		valid bool
	}{
		{"lowercase", "a", true},
		{"upper_and_digits_and_underscore", "A_1", true},
		{"digits_only", "123", true},
		{"hyphen_is_rejected", "a-b", false}, // the GAP-016 CI regression class
		{"space_is_rejected", "a b", false},  // allows identifier injection into CREATE DATABASE
		{"statement_injection_is_rejected", "a;drop", false},
		{"empty_is_rejected", "", false},              // empty would produce `` in CREATE DATABASE
		{"backtick_escape_is_rejected", "a`b", false}, // would break out of the backticked identifier
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := re.MatchString(tc.input); got != tc.valid {
				t.Fatalf("safeDatabaseName(%q)=%v, want %v (pattern %q)",
					tc.input, got, tc.valid, re.String())
			}
		})
	}
}

// qaIsTestingCall reports whether call is t.<method>(...) where receiver is the
// function's *testing.T parameter.
func qaIsTestingCall(call *ast.CallExpr, tParam string) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	recv, ok := sel.X.(*ast.Ident)
	if !ok || recv.Name != tParam {
		return "", false
	}
	return sel.Sel.Name, true
}

// TestQADatabaseNameGuardAbortsInsteadOfSkipping is the half that pins behaviour
// rather than classification: it walks ensureTestDatabase's body and asserts the
// branch reached when safeDatabaseName does NOT match calls t.Fatalf, and that no
// skip call exists anywhere in the function.
func TestQADatabaseNameGuardAbortsInsteadOfSkipping(t *testing.T) {
	fset, file := qaParseDBNameHelper(t)

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if ok && d.Name.Name == qaDBNameHelperFunc && d.Recv == nil {
			fn = d
			break
		}
	}
	if fn == nil {
		t.Fatalf("%s not found in %s: the guard under pin has been removed or renamed", qaDBNameHelperFunc, qaDBNameHelperFile)
	}

	tParam := ""
	if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
		first := fn.Type.Params.List[0]
		// `t *testing.T` parses as StarExpr{X: Ident(testing).Sel(Ident(T))};
		// accept a plain Ident too so the pin survives a receiver rename of the
		// testing package (e.g. `t *testing2.T`).
		var typeName string
		switch typ := first.Type.(type) {
		case *ast.StarExpr:
			switch inner := typ.X.(type) {
			case *ast.SelectorExpr:
				typeName = inner.Sel.Name
			case *ast.Ident:
				typeName = inner.Name
			}
		case *ast.Ident:
			typeName = typ.Name
		}
		if typeName == "T" && len(first.Names) > 0 {
			tParam = first.Names[0].Name
		}
	}
	if tParam == "" {
		t.Fatalf("%s no longer takes *testing.T as its first parameter; unable to pin the abort", qaDBNameHelperFunc)
	}

	var guardBody *ast.BlockStmt
	var skipCalls []string

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			cond, ok := node.Cond.(*ast.UnaryExpr)
			if !ok || cond.Op != token.NOT {
				return true
			}
			call, ok := cond.X.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == qaDBNameValidatorIdent && sel.Sel.Name == "MatchString" {
				guardBody = node.Body
			}
		case *ast.CallExpr:
			method, ok := qaIsTestingCall(node, tParam)
			if ok && strings.HasPrefix(method, "Skip") {
				skipCalls = append(skipCalls, method+" at "+fset.Position(node.Pos()).String())
			}
		}
		return true
	})

	if guardBody == nil {
		t.Fatalf("no `if !%s.MatchString(...) { ... }` branch found in %s: the reject path is gone or restructured, so DoD③ is unpinned",
			qaDBNameValidatorIdent, qaDBNameHelperFunc)
	}

	var rejectCalls []string
	ast.Inspect(guardBody, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if method, ok := qaIsTestingCall(call, tParam); ok {
			rejectCalls = append(rejectCalls, method)
		}
		return true
	})
	sort.Strings(rejectCalls)

	if len(skipCalls) > 0 {
		t.Fatalf("%s must abort on an invalid MYSQL_TEST_DATABASE, but contains skip call(s): %s; "+
			"skipping makes every integration test in this package silently green (GAP-016 DoD③)",
			qaDBNameHelperFunc, strings.Join(skipCalls, ", "))
	}
	if len(rejectCalls) != 1 || rejectCalls[0] != "Fatalf" {
		t.Fatalf("invalid-name branch of %s must call exactly t.Fatalf, got %v; "+
			"that abort IS the negative behaviour DoD③ pins — anything else is silent",
			qaDBNameHelperFunc, rejectCalls)
	}

	t.Logf("%s: `if !%s.MatchString(...)` → %s (no skip path) — DoD③ pinned",
		qaDBNameHelperFunc, qaDBNameValidatorIdent, rejectCalls[0])
}
