package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/logging"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/plugin"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/socket"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/utils"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

const envPrefix = "VAULT_KMS_"

type options struct {
	socket string `env:"SOCKET"`

	debug bool `env:"DEBUG"`

	vaultAddress   string `env:"VAULT_ADDR"`
	vaultNamespace string `env:"VAULT_NAMESPACE"`
	vaultToken     string `env:"VAULT_TOKEN"`

	vaultK8sMount string `env:"VAULT_K8S_MOUNT"`
	vaultK8sRole  string `env:"VAULT_K8S_ROLE"`

	vaultTransitMount string `env:"VAULT_TRANSIT_MOUNT"`
	vaultTransitKey   string `env:"VAULT_TRANSIT_KEY"`

	version bool
}

func defaultOptions() *options {
	return &options{
		socket: "unix:///opt/vaultkms.socket",

		vaultK8sMount: "kubernetes",

		vaultTransitMount: "transit",
		vaultTransitKey:   "kms",
	}
}

// nolint: funlen, cyclop
func main() {
	opts := defaultOptions()

	// first parse any env vars
	if err := opts.parseEnvs(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

	// then flags, since the have precedence over env vars
	flag.StringVar(&opts.socket, "socket", opts.socket, "Destination path of the socket (required)")

	flag.BoolVar(&opts.debug, "debug", opts.debug, "Enable debug logs")

	flag.StringVar(&opts.vaultAddress, "vault-address", opts.vaultAddress, "Vault API address (required)")
	flag.StringVar(&opts.vaultNamespace, "vault-namespace", opts.vaultNamespace, "Vault Namespace (only when Vault Enterprise)")

	flag.StringVar(&opts.vaultToken, "vault-token", opts.vaultToken, "Vault Token (when Token auth) ")

	flag.StringVar(&opts.vaultK8sMount, "vault-k8s-mount", opts.vaultK8sMount, "Vault Kubernetes mount name (when Kubernetes auth)")
	flag.StringVar(&opts.vaultK8sRole, "vault-k8s-role", opts.vaultK8sRole, "Vault Kubernetes role name (when Kubernetes auth)")

	flag.StringVar(&opts.vaultTransitMount, "vault-transit-mount", opts.vaultTransitMount, "Vault Transit mount name")
	flag.StringVar(&opts.vaultTransitKey, "vault-transit-key", opts.vaultTransitKey, "Vault Transit key name")

	flag.BoolVar(&opts.version, "version", opts.version, "")

	flag.Parse()

	if err := opts.validateFlags(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

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
		fmt.Fprintf(os.Stderr, "failed to configure logging")

		os.Exit(1)
	}

	zap.ReplaceGlobals(l)

	zap.L().Info("starting kms plugin",
		zap.String("socket", opts.socket),

		zap.Bool("debug", opts.debug),

		zap.String("vault-address", opts.vaultAddress),
		zap.String("vault-namespace", opts.vaultNamespace),

		zap.String("vault-token", opts.vaultToken),

		zap.String("vault-k8s-mount", opts.vaultK8sMount),
		zap.String("vault-k8s-role", opts.vaultK8sRole),

		zap.String("vault-transit-mount", opts.vaultTransitMount),
		zap.String("vault-transit-key", opts.vaultTransitKey),
	)

	c, err := vault.NewClient(
		vault.WithVaultAddress(opts.vaultAddress),
		vault.WithVaultToken(opts.vaultToken),
		vault.WithVaultNamespace(opts.vaultNamespace),
		vault.WithK8sAuth(opts.vaultK8sMount, opts.vaultK8sRole),
		vault.WithTransit(opts.vaultTransitMount, opts.vaultTransitKey),
	)
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
	pluginV1 := plugin.NewPluginV1(c)
	pluginV1.Register(g)

	zap.L().Info("Successfully registered kms plugin v1")

	pluginV2 := plugin.NewPluginV2(c)
	pluginV2.Register(g)

	zap.L().Info("Successfully registered kms plugin v2")

	go func() {
		if err := g.Serve(listener); err != nil {
			zap.L().Fatal("Failed to start kms plugin", zap.Error(err))
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	signal := <-signals

	zap.L().Info("Received singal", zap.Stringer("signal", signal))
	zap.L().Info("Shutting down server")

	g.GracefulStop()

	zap.L().Info("Exiting...")

	os.Exit(0)
}

func (o *options) validateFlags() error {
	switch {
	case o.vaultAddress == "":
		return errors.New("vault address required")
	case o.vaultToken != "" && o.vaultK8sRole != "":
		return errors.New("cannot use vault-token with vault-k8s-role")
	case o.vaultToken == "" && o.vaultK8sRole == "":
		return errors.New("either vault-token or vault-k8s-role required")
	}

	return nil
}

func (o *options) parseEnvs() error {
	return env.Parse(o, env.Options{
		Prefix: envPrefix,
	})
}
