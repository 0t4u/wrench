package wrench

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/0t4u/wrench/internal/errors"
)

type Bot struct {
	ConfigFile string
	Config     *Config

	started bool
	logFile *os.File
}

func (b *Bot) loadConfig() error {
	if b.Config == nil {
		if b.ConfigFile == "" {
			return errors.New("expected Config or ConfigFile to be specified")
		}
		b.Config = &Config{}
		if absPath, err := filepath.Abs(b.ConfigFile); err == nil {
			b.logf("loading config file: %s", absPath)
		}
		return LoadConfig(b.ConfigFile, b.Config)
	}
	return nil
}

func (b *Bot) logf(format string, v ...any) {
	b.idLogf("general", format, v...)
}

func (b *Bot) idLogf(id, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	timeNow := time.Now().Format(time.RFC3339)
	for _, line := range strings.Split(msg, "\n") {
		fmt.Fprintf(b.logFile, "%s %s: %s\n", timeNow, id, line)
		fmt.Fprintf(os.Stderr, "%s %s: %s\n", timeNow, id, line)
	}
}

func (b *Bot) idWriter(id string) io.Writer {
	return writerFunc(func(p []byte) (n int, err error) {
		b.idLogf(id, "%s", p)
		return len(p), nil
	})
}

type writerFunc func(p []byte) (n int, err error)

func (w writerFunc) Write(p []byte) (n int, err error) {
	return w(p)
}

func (b *Bot) Start() error {
	go func() {
		if err := b.run(); err != nil {
			log.Fatal(err)
		}
	}()
	return nil
}

func (b *Bot) run() error {
	if err := b.loadConfig(); err != nil {
		return errors.Wrap(err, "loading config")
	}

	logFilePath := b.Config.LogFilePath()
	var err error
	b.logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("creating log file %s", logFilePath))
	}

	if err := b.httpStart(); err != nil {
		return errors.Wrap(err, "http")
	}

	b.started = true

	// Wait here until CTRL-C or other term signal is received.
	b.logf("Running (press CTRL-C to exit.)")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGTERM)
	<-sc

	b.logf("Interrupted, shutting down..")

	return errors.Wrap(b.Stop(), "stop")
}

func (b *Bot) Stop() error {
	if !b.started {
		return nil
	}
	b.logFile.Close()
	return nil
}
