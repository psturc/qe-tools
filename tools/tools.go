//go:build tools

package tools

// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

//go:generate go install github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go install mvdan.cc/gofumpt
//go:generate go install github.com/daixiang0/gci
//go:generate go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt
//go:generate go install golang.org/x/tools/cmd/goimports
//go:generate go install golang.org/x/lint/golint
//go:generate go install github.com/go-critic/go-critic/cmd/gocritic
//go:generate go install honnef.co/go/tools/cmd/staticcheck
//go:generate go install github.com/mgechev/revive
//go:generate go install github.com/securego/gosec/v2/cmd/gosec
//go:generate go install github.com/sqs/goreturns
// nolint
import (
	// gci
	_ "github.com/daixiang0/gci"
	// gocritic
	_ "github.com/go-critic/go-critic/cmd/gocritic"
	// golangci-lint
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	// gotestfmt
	_ "github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt"
	// golint
	_ "golang.org/x/lint/golint"
	// goimports
	_ "golang.org/x/tools/cmd/goimports"
	// gofumpt
	_ "mvdan.cc/gofumpt"
	// go-staticcheck
	_ "honnef.co/go/tools/cmd/staticcheck"
	// go-revive
	_ "github.com/mgechev/revive"
	// go-sec
	_ "github.com/securego/gosec/v2/cmd/gosec"
	// go-returns
	_ "github.com/sqs/goreturns"
)
