package sentry

import (
	"fmt"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/oasislabs/oasis-core/go/common/grpc"
	"github.com/oasislabs/oasis-core/go/common/identity"
	"github.com/oasislabs/oasis-core/go/common/logging"
	"github.com/oasislabs/oasis-core/go/sentry/api"
	workerCommon "github.com/oasislabs/oasis-core/go/worker/common"
)

const (
	// CfgEnabled enables the sentry worker.
	CfgEnabled = "worker.sentry.enabled"
	// CfgControlPort configures the sentry worker's control port.
	CfgControlPort = "worker.sentry.control_port"
)

// Flags has the configuration flags.
var Flags = flag.NewFlagSet("", flag.ContinueOnError)

// Enabled returns true if Sentry worker is enabled.
func Enabled() bool {
	return viper.GetBool(CfgEnabled)
}

// Worker is a sentry node worker providing its address(es) to other nodes and
// enabling them to hide their real address(es).
type Worker struct {
	enabled bool

	workerCommonCfg *workerCommon.Config

	backend api.Backend

	grpcServer *grpc.Server

	quitCh chan struct{}
	initCh chan struct{}

	logger *logging.Logger
}

// Name returns the service name.
func (w *Worker) Name() string {
	return "sentry worker"
}

// Start starts the service.
func (w *Worker) Start() error {
	if !w.enabled {
		w.logger.Info("not starting sentry worker as it is disabled")

		return nil
	}

	// Start the sentry gRPC server.
	if err := w.grpcServer.Start(); err != nil {
		w.logger.Error("failed to start sentry gRPC server",
			"err", err,
		)
		return err
	}

	close(w.initCh)

	return nil
}

// Stop halts the service.
func (w *Worker) Stop() {
	if !w.enabled {
		close(w.quitCh)
		return
	}

	w.grpcServer.Stop()
	close(w.quitCh)
}

// Enabled returns true if worker is enabled.
func (w *Worker) Enabled() bool {
	return w.enabled
}

// Quit returns a channel that will be closed when the service terminates.
func (w *Worker) Quit() <-chan struct{} {
	return w.quitCh
}

// Cleanup performs the service specific post-termination cleanup.
func (w *Worker) Cleanup() {
	if !w.enabled {
		return
	}

	w.grpcServer.Cleanup()
}

// New creates a new sentry worker.
func New(workerCommonCfg *workerCommon.Config, backend api.Backend, identity *identity.Identity) (*Worker, error) {
	w := &Worker{
		enabled:         Enabled(),
		workerCommonCfg: workerCommonCfg,
		backend:         backend,
		quitCh:          make(chan struct{}),
		initCh:          make(chan struct{}),
		logger:          logging.GetLogger("worker/sentry"),
	}

	if w.enabled {
		if len(w.workerCommonCfg.SentryAddresses) > 0 {
			return nil, fmt.Errorf(
				"worker/sentry: invalid configuration: sentry worker is enabled, but the node is configured to connect to sentry node(s)",
			)
		}
		grpcServer, err := grpc.NewServer(&grpc.ServerConfig{
			Name:        "sentry",
			Port:        uint16(viper.GetInt(CfgControlPort)),
			Certificate: identity.TLSCertificate,
		})
		if err != nil {
			return nil, fmt.Errorf("worker/sentry: failed to create a new gRPC server: %w", err)
		}
		w.grpcServer = grpcServer
		// Initialize and register the sentry gRPC service.
		api.RegisterService(w.grpcServer.Server(), backend)
	}

	return w, nil
}

func init() {
	Flags.Bool(CfgEnabled, false, "Enable Sentry worker (NOTE: This should only be enabled on Sentry nodes.)")
	Flags.Uint16(CfgControlPort, 9009, "Sentry worker's gRPC server port (NOTE: This should only be enabled on Sentry nodes.)")

	_ = viper.BindPFlags(Flags)
}
