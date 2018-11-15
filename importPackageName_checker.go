package contrib

import (
	"go/ast"
	"strings"

	"github.com/go-lintpack/lintpack"
	"github.com/go-lintpack/lintpack/astwalk"
)

func init() {
	var info lintpack.CheckerInfo
	info.Name = "importPackageName"
	info.Tags = []string{"style"}
	info.Summary = "Detects when imported package names are unnecessary renamed"
	info.Before = `import lint "github.com/go-critic/go-critic/lint"`
	info.After = `import "github.com/go-critic/go-critic/lint"`

	collection.AddChecker(&info, func(ctx *lintpack.CheckerContext) lintpack.FileWalker {
		return &importPackageNameChecker{ctx: ctx}
	})
}

type importPackageNameChecker struct {
	astwalk.WalkHandler
	ctx *lintpack.CheckerContext
}

func (c *importPackageNameChecker) WalkFile(file *ast.File) {
	for _, imp := range file.Imports {
		var pkgName string
		for _, pkgImport := range c.ctx.Pkg.Imports() {
			if pkgImport.Path() == strings.Trim(imp.Path.Value, `"`) {
				pkgName = pkgImport.Name()
				break
			}
		}

		if imp.Name != nil && imp.Name.Name == pkgName {
			c.warn(imp)
		}
	}
}

func (c *importPackageNameChecker) warn(cause ast.Node) {
	c.ctx.Warn(cause, "unnecessary rename of import package")
}
