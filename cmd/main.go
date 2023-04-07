package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	cmd_commons "github.com/cyverse/s3rods/cmd/commons"
	"github.com/cyverse/s3rods/commons"
	"github.com/cyverse/s3rods/service"
	log "github.com/sirupsen/logrus"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "s3rods [args..]",
	Short: "Run S3Rods Service",
	Long:  "Run S3Rods Service that handles S3 requests.",
	RunE:  processCommand,
}

func Execute() error {
	return rootCmd.Execute()
}

func processCommand(command *cobra.Command, args []string) error {
	// check if this is subprocess running in the background
	isChildProc := false
	childProcessArgument := fmt.Sprintf("-%s", cmd_commons.ChildProcessArgument)

	for _, arg := range os.Args {
		if len(arg) >= len(childProcessArgument) {
			if arg == childProcessArgument || arg[1:] == childProcessArgument {
				// background
				isChildProc = true
				break
			}
		}
	}

	if isChildProc {
		// child process
		childMain(command, args)
	} else {
		// parent process
		parentMain(command, args)
	}

	return nil
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000000",
		FullTimestamp:   true,
	})

	log.SetLevel(log.InfoLevel)

	logger := log.WithFields(log.Fields{
		"package":  "main",
		"function": "main",
	})

	// attach common flags
	cmd_commons.SetCommonFlags(rootCmd)

	err := Execute()
	if err != nil {
		logger.Fatalf("%+v", err)
		os.Exit(1)
	}
}

// parentMain handles command-line parameters and run parent process
func parentMain(command *cobra.Command, args []string) {
	logger := log.WithFields(log.Fields{
		"package":  "main",
		"function": "parentMain",
	})

	config, logWriter, cont, err := cmd_commons.ProcessCommonFlags(command)
	if logWriter != nil {
		defer logWriter.Close()
	}

	if err != nil {
		logger.Errorf("%+v", err)
		os.Exit(1)
	}

	if !cont {
		os.Exit(0)
	}

	if !config.Foreground {
		// background
		childStdin, childStdout, err := cmd_commons.RunChildProcess(os.Args[0])
		if err != nil {
			childErr := xerrors.Errorf("failed to run S3Rods Service child process: %w", err)
			logger.Errorf("%+v", childErr)
			os.Exit(1)
		}

		err = cmd_commons.ParentProcessSendConfigViaSTDIN(config, childStdin, childStdout)
		if err != nil {
			sendErr := xerrors.Errorf("failed to send configuration to S3Rods Service child process: %w", err)
			logger.Errorf("%+v", sendErr)
			os.Exit(1)
		}
	} else {
		// run foreground
		err = run(config, false)
		if err != nil {
			runErr := xerrors.Errorf("failed to run S3Rods Service: %w", err)
			logger.Errorf("%+v", runErr)
			os.Exit(1)
		}
	}
}

// childMain runs child process
func childMain(command *cobra.Command, args []string) {
	logger := log.WithFields(log.Fields{
		"package":  "main",
		"function": "childMain",
	})

	logger.Info("Start child process")

	// read from stdin
	config, logWriter, err := cmd_commons.ChildProcessReadConfigViaSTDIN()
	if logWriter != nil {
		defer logWriter.Close()
	}

	if err != nil {
		commErr := xerrors.Errorf("failed to communicate to parent process: %w", err)
		logger.Errorf("%+v", commErr)
		cmd_commons.ReportChildProcessError()
		os.Exit(1)
	}

	config.ChildProcess = true

	logger.Info("Run child process")

	// background
	err = run(config, true)
	if err != nil {
		runErr := xerrors.Errorf("failed to run S3Rods Service: %w", err)
		logger.Errorf("%+v", runErr)
		os.Exit(1)
	}

	if logWriter != nil {
		logWriter.Close()
	}
}

// run runs S3Rods Service
func run(config *commons.Config, isChildProcess bool) error {
	logger := log.WithFields(log.Fields{
		"package":  "main",
		"function": "run",
	})

	if config.Debug {
		log.SetLevel(log.DebugLevel)
	}

	versionInfo := commons.GetVersion()
	logger.Infof("S3Rods Service version - %s, commit - %s", versionInfo.ServiceVersion, versionInfo.GitCommit)

	// make work dirs required
	err := config.MakeWorkDirs()
	if err != nil {
		mkdirErr := xerrors.Errorf("make work dir error: %w", err)
		logger.Errorf("%+v", mkdirErr)
		return err
	}

	err = config.Validate()
	if err != nil {
		configErr := xerrors.Errorf("invalid configuration: %w", err)
		logger.Errorf("%+v", configErr)
		return err
	}

	// run a service
	svc, err := service.Start(config)
	if err != nil {
		serviceErr := xerrors.Errorf("failed to start the service: %w", err)
		logger.Errorf("%+v", serviceErr)
		if isChildProcess {
			cmd_commons.ReportChildProcessError()
		}
		return err
	}

	if isChildProcess {
		cmd_commons.ReportChildProcessStartSuccessfully()
		if len(config.GetLogFilePath()) == 0 {
			cmd_commons.SetNilLogWriter()
		}
	}

	defer func() {
		svc.Stop()

		// remove work dir
		config.CleanWorkDirs()

		os.Exit(0)
	}()

	// wait
	waitForCtrlC()

	return nil
}

func waitForCtrlC() {
	var endWaiter sync.WaitGroup

	endWaiter.Add(1)
	signalChannel := make(chan os.Signal, 1)

	signal.Notify(signalChannel, os.Interrupt)

	go func() {
		<-signalChannel
		endWaiter.Done()
	}()

	endWaiter.Wait()
}
