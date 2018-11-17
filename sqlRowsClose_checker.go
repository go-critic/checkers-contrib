package checkers

import (
	"go/ast"
	"go/types"

	"github.com/go-lintpack/lintpack"
	"github.com/go-lintpack/lintpack/astwalk"
)

func init() {
	var info lintpack.CheckerInfo
	info.Name = "sqlRowsClose"
	info.Tags = []string{"diagnostic", "experimental"}
	info.Summary = "Detects uses of *sql.Rows without call Close method"
	info.Before = `
rows, _ := db.Query( /**/ )
for rows.Next {
}`
	info.After = `
rows, _ := db.Query( /**/ )
for rows.Next {
}
rows.Close()`

	collection.AddChecker(&info, func(ctx *lintpack.CheckerContext) lintpack.FileWalker {
		return astwalk.WalkerForFuncDecl(&sqlRowsCloseChecker{ctx: ctx})
	})
}

type sqlRowsCloseChecker struct {
	astwalk.WalkHandler
	ctx *lintpack.CheckerContext
}

// Warning if sql.Rows local variables (including function parameters):
// 1. Not using as parameter in other functions call;
// 2. Not returning in functions results;
// 3. Not call Close method for variable;

func (c *sqlRowsCloseChecker) VisitFuncDecl(decl *ast.FuncDecl) {
	const rowsTypePTR = "*database/sql.Rows"

	localVars := []types.Object{}
	returnVars := []types.Object{}
	closeVars := []types.Object{}

	for _, b := range decl.Body.List {
		switch b := b.(type) {
		case *ast.AssignStmt:
			// Detect local vars with sql.Rows types
			if b.Lhs != nil {
				for _, l := range b.Lhs {
					if c.typeString(l) == rowsTypePTR {
						localVars = append(localVars, c.getType(l))
					}
				}
			}
		case *ast.ReturnStmt:
			// Detect return vars with sql.Rows types
			if b.Results != nil && len(b.Results) > 0 {
				for _, r := range b.Results {
					if c.typeString(r) == rowsTypePTR {
						returnVars = append(returnVars, c.getType(r))
					}
				}
			}
		case *ast.ExprStmt:
			if sel := c.getCloseSelectorExpr(b.X); sel != nil {
				closeVars = append(closeVars, c.getType(sel.X))
			}
		case *ast.DeferStmt:
			// Detect call Close for sql.Rows variables over defer declaration

			// is it `defer rowsVar.Close()`
			if b, ok := b.Call.Fun.(*ast.SelectorExpr); ok {
				funcName := qualifiedName(b.Sel)
				if funcName == "Close" {
					closeVars = append(closeVars, c.getType(b.X))
				}
			}

			// looking for `rowsVar.Close()` inside `defer func() { ... }()`
			if f, ok := b.Call.Fun.(*ast.FuncLit); ok {
				if f.Body == nil || f.Body.List == nil {
					continue
				}

				for _, s := range f.Body.List {
					expr, ok := s.(*ast.ExprStmt)
					if !ok {
						continue
					}
					ss := c.getCloseSelectorExpr(expr.X)
					if ss != nil {
						closeVars = append(closeVars, c.getType(ss.X))
					}
				}
			}
		}
	}

	// Check local variables
	for _, l := range localVars {
		// If local variable present in return or Close present - PASS
		if !c.varInList(l, returnVars) && !c.varInList(l, closeVars) {
			c.ctx.Warn(l.Parent(), "local variable db.Rows have not Close call")
		}
	}
}

func (c *sqlRowsCloseChecker) getType(x ast.Node) types.Object {
	return c.ctx.TypesInfo.ObjectOf(identOf(x))
}

func (c *sqlRowsCloseChecker) getCloseSelectorExpr(x ast.Node) *ast.SelectorExpr {
	call, ok := x.(*ast.CallExpr)
	if !ok {
		return nil
	}
	if bb, ok := call.Fun.(*ast.SelectorExpr); ok {
		// Detect call Close for sql.Rows variables
		if qualifiedName(bb.Sel) == "Close" {
			return bb
		}
	}
	return nil
}

func (c *sqlRowsCloseChecker) typeString(x ast.Expr) string {
	if typ := c.ctx.TypesInfo.TypeOf(x); typ != nil {
		return typ.String()
	}
	return ""
}

func (c *sqlRowsCloseChecker) varInList(v types.Object, list []types.Object) bool {
	for _, r := range list {
		if v == r {
			return true
		}
	}
	return false
}
