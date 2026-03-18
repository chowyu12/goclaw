package tool

import (
	"context"

	"github.com/chowyu12/goclaw/internal/tool/browser"
	"github.com/chowyu12/goclaw/internal/tool/builtin"
	"github.com/chowyu12/goclaw/internal/tool/codeinterp"
	cronTool "github.com/chowyu12/goclaw/internal/tool/cron"
	"github.com/chowyu12/goclaw/internal/tool/crontab"
	"github.com/chowyu12/goclaw/internal/tool/result"
	"github.com/chowyu12/goclaw/internal/tool/urlreader"
	"github.com/chowyu12/goclaw/internal/tool/writefile"
)

type FileResult = result.FileResult

var ParseFileResult = result.ParseFileResult

func DefaultBuiltins() map[string]func(context.Context, string) (string, error) {
	m := builtin.Handlers()
	m["url_reader"] = urlreader.Handler
	m["browser"] = browser.Handler
	m["cron_parser"] = cronTool.Handler
	m["crontab"] = crontab.Handler
	m["code_interpreter"] = codeinterp.Handler
	m["write_file"] = writefile.Handler
	return m
}
