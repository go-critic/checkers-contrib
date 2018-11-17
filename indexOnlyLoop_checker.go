package checkers

import (
	"go/ast"
	"go/types"

	"github.com/go-lintpack/lintpack"
	"github.com/go-lintpack/lintpack/astwalk"
	"github.com/go-toolsmith/astequal"
	"github.com/go-toolsmith/typep"
)

func init() {
	var info lintpack.CheckerInfo
	info.Name = "indexOnlyLoop"
	info.Tags = []string{"style", "experimental"}
	info.Summary = "Detects for loops that can benefit from rewrite to range loop"
	info.Details = "Suggests to use for key, v := range container form."
	info.Before = `
for i := range files {
	if files[i] != nil {
		files[i].Close()
	}
}`
	info.After = `
for _, f := range files {
	if f != nil {
		f.Close()
	}
}`

	collection.AddChecker(&info, func(ctx *lintpack.CheckerContext) lintpack.FileWalker {
		return astwalk.WalkerForStmt(&indexOnlyLoopChecker{ctx: ctx})
	})
}

type indexOnlyLoopChecker struct {
	astwalk.WalkHandler
	ctx *lintpack.CheckerContext
}

func (c *indexOnlyLoopChecker) VisitStmt(stmt ast.Stmt) {
	rng, ok := stmt.(*ast.RangeStmt)
	if !ok || rng.Key == nil || rng.Value != nil {
		return
	}
	iterated := c.ctx.TypesInfo.ObjectOf(identOf(rng.X))
	if iterated == nil || !c.elemTypeIsPtr(iterated) {
		return // To avoid redundant traversals
	}
	count := 0
	ast.Inspect(rng.Body, func(n ast.Node) bool {
		if n, ok := n.(*ast.IndexExpr); ok {
			if !astequal.Expr(n.Index, rng.Key) {
				return true
			}
			if iterated == c.ctx.TypesInfo.ObjectOf(identOf(n.X)) {
				count++
			}
		}
		// Stop DFS traverse if we found more than one usage.
		return count < 2
	})
	if count > 1 {
		c.warn(stmt, rng.Key, iterated.Name())
	}
}

func (c *indexOnlyLoopChecker) elemTypeIsPtr(obj types.Object) bool {
	switch typ := obj.Type().(type) {
	case *types.Slice:
		return typep.IsPointer(typ.Elem())
	case *types.Array:
		return typep.IsPointer(typ.Elem())
	default:
		return false
	}
}

func (c *indexOnlyLoopChecker) warn(x, key ast.Node, iterated string) {
	c.ctx.Warn(x, "%s occurs more than once in the loop; consider using for _, value := range %s",
		key, iterated)
}
