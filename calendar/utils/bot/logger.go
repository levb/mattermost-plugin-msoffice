// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package bot

import (
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils"
)

const timed = "__since"
const Elapsed = "Elapsed"

type LogContext map[string]interface{}

func level(l string) int {
	switch l {
	case "debug":
		return 4
	case "info":
		return 3
	case "warn":
		return 2
	case "error":
		return 1
	}
	return 0
}

type Logger interface {
	With(LogContext) Logger
	Timed() Logger
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}

func toKeyValuePairs(in map[string]interface{}) (out []interface{}) {
	for k, v := range in {
		out = append(out, k)
		out = append(out, v)
	}
	return out
}

func (bot *bot) With(logContext LogContext) Logger {
	newbot := *bot
	if len(newbot.logContext) == 0 {
		newbot.logContext = map[string]interface{}{}
	}
	for k, v := range logContext {
		newbot.logContext[k] = v
	}
	return &newbot
}

func (bot *bot) Timed() Logger {
	return bot.With(LogContext{
		timed: time.Now(),
	})
}

func (bot *bot) Debugf(format string, args ...interface{}) {
	measure(bot.logContext)
	message := fmt.Sprintf(format, args...)
	bot.pluginAPI.LogDebug(message, toKeyValuePairs(bot.logContext)...)
	if level(bot.AdminLogLevel) >= 4 {
		bot.logToAdmins("DEBUG", message)
	}
}

func (bot *bot) Errorf(format string, args ...interface{}) {
	measure(bot.logContext)
	message := fmt.Sprintf(format, args...)
	bot.pluginAPI.LogError(message, toKeyValuePairs(bot.logContext)...)
	if level(bot.AdminLogLevel) >= 1 {
		bot.logToAdmins("ERROR", message)
	}
}

func (bot *bot) Infof(format string, args ...interface{}) {
	measure(bot.logContext)
	message := fmt.Sprintf(format, args...)
	bot.pluginAPI.LogInfo(message, toKeyValuePairs(bot.logContext)...)
	if level(bot.AdminLogLevel) >= 3 {
		bot.logToAdmins("INFO", message)
	}
}

func (bot *bot) Warnf(format string, args ...interface{}) {
	measure(bot.logContext)
	message := fmt.Sprintf(format, args...)
	bot.pluginAPI.LogWarn(message, toKeyValuePairs(bot.logContext)...)
	if level(bot.AdminLogLevel) >= 2 {
		bot.logToAdmins("WARN", message)
	}
}

func (bot *bot) logToAdmins(level, message string) {
	if bot.AdminLogVerbose && len(bot.logContext) > 0 {
		message += "\n" + utils.JSONBlock(bot.logContext)
	}
	bot.dmAdmins("(log " + level + ") " + message)
}

type NilLogger struct{}

func (l *NilLogger) With(_ LogContext) Logger          { return l }
func (l *NilLogger) Timed() Logger                     { return l }
func (l *NilLogger) Debugf(_ string, _ ...interface{}) {}
func (l *NilLogger) Errorf(_ string, _ ...interface{}) {}
func (l *NilLogger) Infof(_ string, _ ...interface{})  {}
func (l *NilLogger) Warnf(_ string, _ ...interface{})  {}

type TestLogger struct {
	testing.TB
	logContext LogContext
}

func (l *TestLogger) With(logContext LogContext) Logger {
	newl := *l
	if len(newl.logContext) == 0 {
		newl.logContext = map[string]interface{}{}
	}
	for k, v := range logContext {
		newl.logContext[k] = v
	}
	return &newl
}

func (l *TestLogger) Timed() Logger {
	return l.With(LogContext{
		timed: time.Now(),
	})
}

func (l *TestLogger) logf(prefix, format string, args ...interface{}) {
	out := fmt.Sprintf(prefix+": "+format, args...)
	if len(l.logContext) > 0 {
		measure(l.logContext)
		out += fmt.Sprintf(" -- %+v", l.logContext)
	}
	l.TB.Logf(out)
}

func measure(lc LogContext) {
	if lc[timed] == nil {
		return
	}
	started := lc[timed].(time.Time)
	lc[Elapsed] = time.Since(started).String()
	delete(lc, timed)
}

func (l *TestLogger) Debugf(format string, args ...interface{}) { l.logf("DEBUG", format, args...) }
func (l *TestLogger) Errorf(format string, args ...interface{}) { l.logf("ERROR", format, args...) }
func (l *TestLogger) Infof(format string, args ...interface{})  { l.logf("INFO", format, args...) }
func (l *TestLogger) Warnf(format string, args ...interface{})  { l.logf("WARN", format, args...) }
