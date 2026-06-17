// Package validator performs semantic checks over a parsed Document: arrow and
// anchor reference resolution, ID uniqueness within a scope, and range
// constraints. It is non-fail-fast and returns all errors it finds.
//
// TODO(go-port): the real validator has not been ported yet. The TypeScript
// implementation (src/validator/validator.ts in the pre-rewrite history) is the
// reference. For now Validate is a no-op so the pipeline composes end to end;
// the parser still reports syntax and range errors.
package validator

import "github.com/kurrik/arkitecture/ast"

// Validate returns every semantic error in the document. Stub pending the port.
func Validate(doc *ast.Document) []ast.Error {
	_ = doc
	return nil
}
