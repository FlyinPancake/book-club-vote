package server

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	wishbubbletea "charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/ssh"

	"bookclubvote/internal/config"
	"bookclubvote/internal/ui"
)

func Run(ctx context.Context, cfg config.Config) error {
	srv, err := wish.NewServer(
		wish.WithAddress(cfg.Server.Listen),
		wish.WithHostKeyPath(cfg.Server.HostKeyPath),
		wish.WithMiddleware(
			wishbubbletea.Middleware(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				return ui.NewModel(cfg, now()), nil
			}),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return err
	}

	log.Info("starting ssh server", "listen", cfg.Server.Listen)
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

var (
	now             = func() time.Time { return time.Now().UTC() }
	shutdownTimeout = 30 * time.Second
)
