package cmd

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	g "github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/grpc"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/logging"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/plugin"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/socket"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/utils"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

type Options struct {
	Socket string `env:"SOCKET" envDefault:"unix:///opt/kms/vaultkms.socket"`

	Debug bool `env:"DEBUG"`

	// vault server
	VaultAddress   string `env:"VAULT_ADDR"`
	VaultNamespace string `env:"VAULT_NAMESPACE"`

	// vault auth
	VaultToken    string `env:"VAULT_TOKEN"`
	VaultK8sMount string `env:"VAULT_K8S_MOUNT" envDefault:"kubernetes"`
	VaultK8sRole  string `env:"VAULT_K8S_ROLE"`

	// vault transit
	VaultTransitKey   string `env:"VAULT_TRANSIT_KEY"   envDefault:"kms"`
	VaultTransitMount string `env:"VAULT_TRANSIT_MOUNT" envDefault:"transit"`

	Version bool
}

// NewPlugin instantiates the plugin.
// nolint: funlen, cyclop
func NewPlugin() error {
	opts := &Options{}

	// first parse any env vars
	if err := utils.ParseEnvs("VAULT_KMS_", opts); err != nil {
		return fmt.Errorf("error parsing env vars: %w", err)
	}

	fmt.Println(opts)

	// then flags, since the have precedence over env vars
	flag.StringVar(&opts.Socket, "socket", opts.Socket, "Destination path of the socket (required)")

	flag.BoolVar(&opts.Debug, "debug", opts.Debug, "Enable debug logs")

	flag.StringVar(&opts.VaultAddress, "vault-address", opts.VaultAddress, "Vault API address (required)")
	flag.StringVar(&opts.VaultNamespace, "vault-namespace", opts.VaultNamespace, "Vault Namespace (only when Vault Enterprise)")

	flag.StringVar(&opts.VaultToken, "vault-token", opts.VaultToken, "Vault Token (when Token auth) ")

	flag.StringVar(&opts.VaultK8sMount, "vault-k8s-mount", opts.VaultK8sMount, "Vault Kubernetes mount name (when Kubernetes auth)")
	flag.StringVar(&opts.VaultK8sRole, "vault-k8s-role", opts.VaultK8sRole, "Vault Kubernetes role name (when Kubernetes auth)")

	flag.StringVar(&opts.VaultTransitMount, "vault-transit-mount", opts.VaultTransitMount, "Vault Transit mount name")
	flag.StringVar(&opts.VaultTransitKey, "vault-transit-key", opts.VaultTransitKey, "Vault Transit key name")

	flag.BoolVar(&opts.Version, "version", opts.Version, "")

	flag.Parse()

	if err := opts.validateFlags(); err != nil {
		return fmt.Errorf("error validating args: %w", err)
	}

	if opts.Version {
		fmt.Fprintln(os.Stdout, "version")

		os.Exit(0)
	}

	logLevel := zapcore.InfoLevel

	if opts.Debug {
		logLevel = zapcore.DebugLevel
	}

	l, err := logging.NewStandardLogger(logLevel)
	if err != nil {
		return fmt.Errorf("failed to configure logging: %w", err)
	}

	zap.ReplaceGlobals(l)

	zap.L().Info("starting kms plugin",
		zap.String("socket", opts.Socket),

		zap.Bool("debug", opts.Debug),

		zap.String("vault-address", opts.VaultAddress),
		zap.String("vault-namespace", opts.VaultNamespace),

		zap.String("vault-token", opts.VaultToken),

		zap.String("vault-k8s-mount", opts.VaultK8sMount),
		zap.String("vault-k8s-role", opts.VaultK8sRole),

		zap.String("vault-transit-mount", opts.VaultTransitMount),
		zap.String("vault-transit-key", opts.VaultTransitKey),
	)

	vc, err := vault.NewClient(
		vault.WithVaultAddress(opts.VaultAddress),
		vault.WithVaultToken(opts.VaultToken),
		vault.WithVaultNamespace(opts.VaultNamespace),
		vault.WithK8sAuth(opts.VaultK8sMount, opts.VaultK8sRole),
		vault.WithTransit(opts.VaultTransitMount, opts.VaultTransitKey),
	)
	if err != nil {
		zap.L().Fatal("Failed to create vault client", zap.Error(err))
	}

	_, err = vc.Client.Auth().Token().LookupSelf()
	if err != nil {
		zap.L().Fatal("Failed to connect to vault", zap.Error(err))
	}

	zap.L().Info("Successfully authenticated to vault")

	s, err := socket.NewSocket(opts.Socket)
	if err != nil {
		zap.L().Fatal("Cannot create socket", zap.Error(err))
	}

	zap.L().Info("Successfully created unix socket", zap.String("socket", s.Path))

	listener, err := s.Listen()
	if err != nil {
		log.Fatal(err)
	}

	zap.L().Info("Listening for connection")

	grpcOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(g.UnaryServerInterceptor),
	}

	grpc := grpc.NewServer(grpcOpts...)
	pluginV1 := plugin.NewPluginV1(vc)
	pluginV1.Register(grpc)

	zap.L().Info("Successfully registered kms plugin v1")

	pluginV2 := plugin.NewPluginV2(vc)
	pluginV2.Register(grpc)

	zap.L().Info("Successfully registered kms plugin v2")

	go func() {
		if err := grpc.Serve(listener); err != nil {
			zap.L().Fatal("Failed to start kms plugin", zap.Error(err))
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	signal := <-signals

	zap.L().Info("Received signal", zap.Stringer("signal", signal))
	zap.L().Info("Shutting down server")

	grpc.GracefulStop()

	zap.L().Info("Exiting...")

	return nil
}

func (o *Options) validateFlags() error {
	switch {
	case o.VaultAddress == "":
		return errors.New("vault address required")
	case o.VaultToken != "" && o.VaultK8sRole != "":
		return errors.New("cannot use vault-token with vault-k8s-role")
	case o.VaultToken == "" && o.VaultK8sRole == "":
		return errors.New("either vault-token or vault-k8s-role required")
	}

	return nil
}
