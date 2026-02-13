package signals

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/AyakuraYuki/go-anywhere/internal/log"
)

func GraceStop(callback func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

	s := <-ch
	log.Scope("signals").Infof("server shutdown (%v)", s)

	callback()
	os.Exit(0)
}
