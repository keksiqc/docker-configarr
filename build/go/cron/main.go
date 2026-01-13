package main

import (
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/go-co-op/gocron/v2"
)

const SCHEDULE = "CONFIGARR_SCHEDULE"

var (
	PID    int
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
)

func main() {
	// catch syscalls
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT)
	go func() {
		<-signalChannel
		os.Exit(0)
	}()

	// check arguments
	if len(os.Args) > 1 {
		args := os.Args[1:]
		switch args[0] {
		case "--ping":
			_, err := os.FindProcess(PID)
			if err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
	} else {
		// set schedule
		if _, ok := os.LookupEnv(SCHEDULE); ok {
			logger.Info("setting schedule", "cron", os.Getenv(SCHEDULE))
			scheduler, err := gocron.NewScheduler()
			if err != nil {
				logger.Error("cron error", "err", err)
			}
			_, err = scheduler.NewJob(gocron.CronJob(os.Getenv(SCHEDULE), false), gocron.NewTask(run))
			if err != nil {
				logger.Error("cron error", "err", err)
			}
			scheduler.Start()
		}

		// execute
		run()

		// wait
		select {}
	}
}

func run() {
	cmd := exec.Command("/usr/local/bin/node", "/opt/configarr/bundle.cjs")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// start process
	err := cmd.Start()
	PID = cmd.Process.Pid
	logger.Info("starting configarr sync process", "pid", PID)
	if err != nil {
		logger.Error("sync error", "err", err)
	} else {
		err = cmd.Wait()
		if err != nil {
			logger.Error("sync error", "err", err)
		} else {
			logger.Info("sync complete")
		}
	}
}
