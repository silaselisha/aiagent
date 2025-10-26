package cmdlog

import (
	"starseed/internal/logging"
	"starseed/internal/metrics"
)

func Run(cmd string, f func() error) error {
	metrics.IncCommandRun(cmd)
	err := f()
	if err != nil {
		metrics.IncCommandError(cmd)
		logging.Error(cmd+"_error", map[string]any{"error": err.Error()})
	} else {
		logging.Info(cmd+"_ok", nil)
	}
	return err
}
