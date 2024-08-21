package cmd

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	customHTTP "github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/http"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/logging"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/plugin"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/probes"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/socket"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/utils"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// auth
	AuthMethod string `env:"AUTH_METHOD"`

	// token auth
	Token string `env:"TOKEN"`

	// approle auth
	AppRoleRoleID       string `env:"APPROLE_ROLE_ID"`
	AppRoleRoleSecretID string `env:"APPROLE_SECRET_ID"`
	AppRoleMount        string `env:"APPROLE_MOUNT"     envDefault:"approle"`

	// transit
	TransitKey   string `env:"TRANSIT_KEY"   envDefault:"kms"`
	TransitMount string `env:"TRANSIT_MOUNT" envDefault:"transit"`

	// healthz check
	HealthPort string `env:"HEALTH_PORT" envDefault:":8080"`

	// Disable KMSv1 Plugin
	DisableV1 bool `env:"DISABLE_V1" envDefault:"false"`

	Version bool
}

// NewPlugin instantiates the plugin.
// nolint: funlen, cyclop
func NewPlugin(version string) error {
	opts := &Options{}

	// first parse any env vars
	if err := utils.ParseEnvs("VAULT_KMS_", opts); err != nil {
		return fmt.Errorf("error parsing env vars: %w", err)
	}

	flag := flag.FlagSet{}
	// then flags, since they have precedence over env vars
	flag.StringVar(&opts.Socket, "socket", opts.Socket, "Destination path of the socket (required)")

	flag.BoolVar(&opts.Debug, "debug", opts.Debug, "Enable debug logs")

	flag.StringVar(&opts.VaultAddress, "vault-address", opts.VaultAddress, "Vault API address (required)")
	flag.StringVar(&opts.VaultNamespace, "vault-namespace", opts.VaultNamespace, "Vault Namespace (only when Vault Enterprise)")

	flag.StringVar(&opts.AuthMethod, "auth-method", opts.AuthMethod, "Auth Method. Supported: token, approle, k8s")

	flag.StringVar(&opts.Token, "token", opts.Token, "Vault Token (when Token auth)")

	flag.StringVar(&opts.AppRoleMount, "approle-mount", opts.AppRoleMount, "Vault Approle mount name (when approle auth)")
	flag.StringVar(&opts.AppRoleRoleID, "approle-role-id", opts.AppRoleRoleID, "Vault Approle role ID (when approle auth)")
	flag.StringVar(&opts.AppRoleRoleSecretID, "approle-secret-id", opts.AppRoleRoleSecretID, "Vault Approle Secret ID (when approle auth)")

	flag.StringVar(&opts.TransitMount, "transit-mount", opts.TransitMount, "Vault Transit mount name")
	flag.StringVar(&opts.TransitKey, "transit-key", opts.TransitKey, "Vault Transit key name")

	flag.StringVar(&opts.HealthPort, "health-port", opts.HealthPort, "Health Check Port")

	flag.BoolVar(&opts.DisableV1, "disable-v1", opts.DisableV1, "disable the v1 kms plugin")

	flag.BoolVar(&opts.Version, "version", opts.Version, "prints out the plugins version")

	if err := flag.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	if opts.Version {
		fmt.Fprintf(os.Stdout, "vault-kubernetes-kms v%s\n", version)

		os.Exit(0)
	}

	if err := opts.validateFlags(); err != nil {
		return fmt.Errorf("error validating args: %w", err)
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

	var (
		authMethod   vault.Option
		logFields    []zapcore.Field
		healthChecks = []probes.Prober{}
	)

	logFields = append(logFields,
		zap.String("auth-method", opts.AuthMethod),
		zap.String("socket", opts.Socket),
		zap.Bool("debug", opts.Debug),
		zap.String("vault-address", opts.VaultAddress),
		zap.String("vault-namespace", opts.VaultNamespace),
		zap.String("transit-engine", opts.TransitMount),
		zap.String("transit-key", opts.TransitKey),
		zap.String("health-port", opts.HealthPort),
		zap.Bool("disable-v1", opts.DisableV1),
	)

	switch strings.ToLower(opts.AuthMethod) {
	case "token":
		authMethod = vault.WithTokenAuth(opts.Token)
	case "approle":
		authMethod = vault.WithAppRoleAuth(opts.AppRoleMount, opts.AppRoleRoleID, opts.AppRoleRoleSecretID)
		logFields = append(logFields,
			zap.String("approle-mount", opts.AppRoleMount),
			zap.String("approle-role-id", opts.AppRoleRoleID))
	default:
		return fmt.Errorf("invalid auth method: %s", opts.AuthMethod)
	}

	zap.L().Info("starting kms plugin", logFields...)

	vc, err := vault.NewClient(
		customHTTP.NewHTTPClient().Client,
		vault.WithVaultAddress(opts.VaultAddress),
		vault.WithVaultNamespace(opts.VaultNamespace),
		vault.WithTransit(opts.TransitMount, opts.TransitKey),
		authMethod,
	)
	if err != nil {
		zap.L().Fatal("Failed to create vault client", zap.Error(err))
	}

	zap.L().Info("Successfully authenticated to vault")

	listener, err := socket.Listen(opts.Socket)
	if err != nil {
		zap.L().Fatal("Cannot listen on socket", zap.Error(err))
	}

	zap.L().Info("Successfully dialed to unix domain socket", zap.String("socket", opts.Socket))

	grpc := grpc.NewServer()

	if !opts.DisableV1 {
		pluginV1 := plugin.NewPluginV1(vc)
		pluginV1.Register(grpc)

		healthChecks = append(healthChecks, pluginV1)

		zap.L().Info("Successfully registered kms plugin v1")
	}

	pluginV2 := plugin.NewPluginV2(vc)
	pluginV2.Register(grpc)
	healthChecks = append(healthChecks, pluginV2)

	zap.L().Info("Successfully registered kms plugin v2")

	go func() {
		if err := grpc.Serve(listener); err != nil {
			zap.L().Fatal("Failed to start kms plugin", zap.Error(err))
		}
	}()

	go func() {
		promReg := metrics.RegisterPrometheusMetrics()

		http.HandleFunc("GET /healthz", probes.HealthZ(healthChecks))
		http.HandleFunc("GET /livez", probes.HealthZ(healthChecks))
		http.HandleFunc("GET /metrics", promhttp.HandlerFor(promReg,
			promhttp.HandlerOpts{
				EnableOpenMetrics: false,
			}).ServeHTTP)

		//nolint:gosec
		if err := http.ListenAndServe(opts.HealthPort, nil); err != nil {
			zap.L().Fatal("Failed to start health check handlers", zap.Error(err))
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

// nolint: cyclop
func (o *Options) validateFlags() error {
	switch {
	case o.VaultAddress == "":
		return errors.New("vault address required")
	// check auth method
	case !slices.Contains([]string{"token", "approle"}, o.AuthMethod):
		return errors.New("invalid auth method. Supported: token, approle")

	// validate token auth
	case o.AuthMethod == "token" && o.Token == "":
		return errors.New("token required when using token auth")

	// validate approle auth
	case o.AuthMethod == "approle" && (o.AppRoleRoleID == "" || o.AppRoleRoleSecretID == ""):
		return errors.New("approle role id and secret id required when using approle auth")
	}

	return nil
}
