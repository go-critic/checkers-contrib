// Package checkers is a gocritic linter auxiliary checks submited by contributors.
package checkers

import (
	"github.com/go-lintpack/lintpack"
)

var collection = &lintpack.CheckerCollection{
	URL: "https://github.com/go-critic/checkers-contrib",
}
