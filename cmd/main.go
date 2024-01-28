package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/kms"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/logging"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/socket"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/utils"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

type options struct {
	socket string

	debug bool

	vaultAddress     string
	vaultToken       string
	vaultTransitPath string
	vaultTransitKey  string

	version bool
}

func defaultOptions() *options {
	return &options{
		socket:           "unix:///opt/vaultkms.socket",
		vaultTransitPath: "transit",
		vaultTransitKey:  "kms",
	}
}

// nolint: funlen
func main() {
	opts := defaultOptions()

	flag.StringVar(&opts.socket, "socket", opts.socket, "")

	flag.BoolVar(&opts.debug, "debug", opts.debug, "Enable debug logs")

	flag.StringVar(&opts.vaultAddress, "vault-address", opts.vaultAddress, "")
	flag.StringVar(&opts.vaultToken, "vault-token", opts.vaultToken, "")
	flag.StringVar(&opts.vaultTransitPath, "vault-transit-path", opts.vaultTransitPath, "")
	flag.StringVar(&opts.vaultTransitKey, "vault-transit-key", opts.vaultTransitKey, "")

	flag.BoolVar(&opts.version, "version", opts.version, "")

	flag.Parse()

	if opts.version {
		fmt.Fprintln(os.Stdout, "version")

		os.Exit(0)
	}

	logLevel := zapcore.InfoLevel

	if opts.debug {
		logLevel = zapcore.DebugLevel
	}

	l, err := logging.NewStandardLogger(logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to configure logging")
		os.Exit(1)
	}

	zap.ReplaceGlobals(l)

	c, err := vault.NewClient(opts.vaultAddress, opts.vaultToken, opts.vaultTransitPath, opts.vaultTransitKey)
	if err != nil {
		zap.L().Fatal("Failed to create vault client", zap.Error(err))
	}

	_, err = c.Client.Auth().Token().LookupSelf()
	if err != nil {
		zap.L().Fatal("Failed to connect to vault", zap.Error(err))
	}

	zap.L().Info("Successfully authenticated to vault")

	s, err := socket.NewSocket(opts.socket)
	if err != nil {
		zap.L().Fatal("Cannot create socket", zap.Error(err))
	}

	zap.L().Info("Successfully created unix socket", zap.String("socket", s.Path))

	listener, err := s.Listen()
	if err != nil {
		log.Fatal(err)
	}

	zap.L().Info("Listening for connection")

	gprcOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.UnaryServerInterceptor),
	}

	g := grpc.NewServer(gprcOpts...)
	p := kms.NewKMSPlugin(c)

	p.Register(g)

	zap.L().Info("Successfully registered kms plugin")

	go func() {
		if err := g.Serve(listener); err != nil {
			zap.L().Fatal("Failed to start kms plugin", zap.Error(err))
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	signal := <-signals

	zap.L().Info("Received singnal", zap.Stringer("signal", signal))
	zap.L().Info("Shutting down server")

	g.GracefulStop()

	zap.L().Info("Exiting...")

	os.Exit(0)
}
