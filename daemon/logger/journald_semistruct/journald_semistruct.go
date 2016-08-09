// +build linux

// Package journald provides the log driver for forwarding server logs
// to endpoints that receive the systemd format.
package journald_semistruct

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	cp "github.com/andyleap/parser"
	sp "github.com/awakenetworks/semistruct-parser"
	"github.com/coreos/go-systemd/journal"
	"github.com/docker/docker/daemon/logger"
	"github.com/docker/docker/daemon/logger/loggerutils"
)

const name = "journald_semistruct"

type journald struct {
	vars    map[string]string // additional variables and values to send to the journal along with the log message
	readers readerList
	parser  *cp.Grammar
}

type readerList struct {
	mu      sync.Mutex
	readers map[*logger.LogWatcher]*logger.LogWatcher
}

func init() {
	if err := logger.RegisterLogDriver(name, New); err != nil {
		logrus.Fatal(err)
	}
	if err := logger.RegisterLogOptValidator(name, validateLogOpt); err != nil {
		logrus.Fatal(err)
	}
}

// New creates a journald logger using the configuration passed in on
// the context.
func New(ctx logger.Context) (logger.Logger, error) {
	if !journal.Enabled() {
		return nil, fmt.Errorf("journald is not enabled on this host")
	}
	// Strip a leading slash so that people can search for
	// CONTAINER_NAME=foo rather than CONTAINER_NAME=/foo.
	name := ctx.ContainerName
	if name[0] == '/' {
		name = name[1:]
	}

	// parse log tag
	tag, err := loggerutils.ParseLogTag(ctx, "")
	if err != nil {
		return nil, err
	}

	vars := map[string]string{
		"CONTAINER_ID":      ctx.ContainerID[:12],
		"CONTAINER_ID_FULL": ctx.ContainerID,
		"CONTAINER_NAME":    name,
		"CONTAINER_TAG":     tag,
	}

	extraAttrs := ctx.ExtraAttributes(strings.ToTitle)

	for k, v := range extraAttrs {
		vars[k] = v
	}

	pr := sp.ParseSemistruct()
	return &journald{vars: vars, parser: pr, readers: readerList{readers: make(map[*logger.LogWatcher]*logger.LogWatcher)}}, nil
}

// We don't actually accept any options, but we have to supply a callback for
// the factory to pass the (probably empty) configuration map to.
func validateLogOpt(cfg map[string]string) error {
	for key := range cfg {
		switch key {
		case "labels":
		case "env":
		case "tag":
		default:
			return fmt.Errorf("unknown log opt '%s' for journald log driver", key)
		}
	}
	return nil
}

func (s *journald) Log(msg *logger.Message) error {

	var semistruct_line cp.Match

	journald_vars := s.vars

	line := string(msg.Line)

	// Peak at the first few characters, if they start with the
	// sentinel then attempt a parse, otherwise don't parse and just
	// shove the whole line out to journald.
	if len(line) > 2 && line[:2] == "!<" {
		semistruct_line, _ = s.parser.ParseString(line)
	}

	// If we have a successful parse, let's set the journal priority
	// using the integer priority value from the semistructured log
	// line, if not let's just set it to Err or Info as-per the stock
	// journald logging driver.
	var priority journal.Priority

	if semistruct_line != nil {
		res := semistruct_line.(sp.Semistruct_log)

		priority = journal.Priority(res.Priority)

		journald_vars["TAGS"] = strings.Join(res.Tags, ":")

		for k, v := range res.Attrs {
			journald_vars[k] = v
		}
	} else {
		if msg.Source == "stderr" {
			priority = journal.PriErr
		} else {
			priority = journal.PriInfo
		}
	}

	// NOTE: we always send the whole line to journald even though
	// it's semi-structured, the fact that we have some structure to
	// parse just gives us more fields to filter by with journalctl.
	return journal.Send(line, priority, journald_vars)
}

func (s *journald) Name() string {
	return name
}
