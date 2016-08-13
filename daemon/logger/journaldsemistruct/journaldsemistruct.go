// +build linux

// Package journald provides the log driver for forwarding server logs
// to endpoints that receive the systemd format.
package journaldsemistruct

import (
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	cp "github.com/andyleap/parser"
	semistruct "github.com/awakenetworks/semistruct-parser"
	"github.com/coreos/go-systemd/journal"
	"github.com/docker/docker/daemon/logger"
	"github.com/docker/docker/daemon/logger/loggerutils"
	"strings"
	"sync"
)

const name = "journald-semistruct"

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

	pr := semistruct.NewLogParser()
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
	journaldVars := s.vars

	line := string(msg.Line)

	// If we have a successful parse, let's set the journal priority
	// using the integer priority value from the semistructured log
	// line, if not let's just set it to Err or Info as-per the stock
	// journald logging driver.
	var priority journal.Priority

	if parsedLog, err := parseSemistruct(line, s.parser); err == nil && parsedLog != nil {
		res, ok := parsedLog.(semistruct.Log)

		if ok {
			priority = journal.Priority(res.Priority)
			journaldVars["TAGS"] = strings.Join(res.Tags, ":")
			for k, v := range res.Attrs {
				journaldVars[k] = v
			}
		} else {
			priority = defaultPriority(msg.Source)
		}
	} else {
		priority = defaultPriority(msg.Source)
	}

	// NOTE: we always send the whole line to journald even though
	// it's semi-structured, the fact that we have some structure to
	// parse just gives us more fields to filter by with journalctl.
	return journal.Send(line, priority, journaldVars)
}

func defaultPriority(source string) journal.Priority {
	if msg.Source == "stderr" {
		return journal.PriErr
	} else {
		return journal.PriInfo
	}
}

func parseSemistruct(s string, parser *cp.Grammar) (cp.Match, error) {
	// Peak at the first few characters, if they start with the
	// sentinel then attempt a parse
	if len(s) > 2 && s[:2] == "!<" {
		if parsedLog, err := parser.ParseString(s); err != nil && parsedLog == nil {
			logrus.Errorf("failed to parse semistructured log line: %v", err)
			return nil, err
		} else {
			return parsedLog, nil
		}
	} else {
		return nil, errors.New("sentinel not seen")
	}
}

func (s *journald) Name() string {
	return name
}
