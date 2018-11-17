package checkers

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/go-lintpack/lintpack"
	"github.com/go-lintpack/lintpack/astwalk"
)

func init() {
	var info lintpack.CheckerInfo
	info.Name = "boolFuncPrefix"
	info.Tags = []string{"style", "experimental", "opinionated"}
	info.Summary = "Detects function returning only bool and suggests to add Is/Has/Contains prefix to it's name"
	info.Before = "func Enabled() bool"
	info.After = "func IsEnabled() bool"

	collection.AddChecker(&info, func(ctx *lintpack.CheckerContext) lintpack.FileWalker {
		return astwalk.WalkerForFuncDecl(&boolFuncPrefixChecker{ctx: ctx})
	})
}

type boolFuncPrefixChecker struct {
	astwalk.WalkHandler
	ctx *lintpack.CheckerContext
}

func (c *boolFuncPrefixChecker) VisitFuncDecl(decl *ast.FuncDecl) {
	params := decl.Type.Params
	results := decl.Type.Results

	if params.NumFields() > 0 ||
		results.NumFields() != 1 ||
		!c.isBoolType(results.List[0].Type) ||
		c.hasProperPrefix(decl.Name.Name) {
		return
	}
	c.warn(decl)
}

func (c *boolFuncPrefixChecker) warn(fn *ast.FuncDecl) {
	c.ctx.Warn(fn, "consider to add Is/Has/Contains prefix to function name")
}

func (c *boolFuncPrefixChecker) isBoolType(expr ast.Expr) bool {
	typ, ok := c.ctx.TypesInfo.TypeOf(expr).(*types.Basic)
	return ok && typ.Kind() == types.Bool
}

func (c *boolFuncPrefixChecker) hasProperPrefix(name string) bool {
	name = strings.ToLower(name)
	excluded := []string{"exit", "quit"}
	for _, ex := range excluded {
		if name == ex {
			return true
		}
	}

	prefixes := []string{"is", "has", "contains", "check", "get", "should", "need", "may", "should"}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
