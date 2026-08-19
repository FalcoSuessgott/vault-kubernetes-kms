package vault

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

const spiffeJWTTokenFetchTimeout = 10 * time.Second

type jwtTokenSource interface {
	Token(ctx context.Context) (string, error)
}

type fileJWTTokenSource struct {
	path string
}

func (s fileJWTTokenSource) Token(_ context.Context) (string, error) {
	jwtBytes, err := os.ReadFile(s.path)
	if err != nil {
		return "", fmt.Errorf("error reading jwt token file %q: %w", s.path, err)
	}

	return strings.TrimSpace(string(jwtBytes)), nil
}

type spiffeJWTTokenSource struct {
	endpoint string
	audience string
	subject  spiffeid.ID
	fetch    func(context.Context, jwtsvid.Params, ...workloadapi.ClientOption) (*jwtsvid.SVID, error)
}

func (s spiffeJWTTokenSource) Token(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, spiffeJWTTokenFetchTimeout)
	defer cancel()

	var options []workloadapi.ClientOption
	if s.endpoint != "" {
		options = append(options, workloadapi.WithAddr(s.endpoint))
	}

	fetch := s.fetch
	if fetch == nil {
		fetch = workloadapi.FetchJWTSVID
	}

	svid, err := fetch(ctx, jwtsvid.Params{
		Audience: s.audience,
		Subject:  s.subject,
	}, options...)
	if err != nil {
		return "", fmt.Errorf("error fetching SPIFFE JWT-SVID: %w", err)
	}

	return svid.Marshal(), nil
}

// WithJWTAuth performs a JWT auth login using a token file. The file is
// re-read on every login so that rotated tokens are picked up automatically.
func WithJWTAuth(mount, role, tokenPath string) Option {
	return withJWTTokenSource(mount, role, fileJWTTokenSource{path: tokenPath})
}

// WithSPIFFEJWTAuth performs a JWT auth login using a JWT-SVID fetched from
// the SPIFFE Workload API. A fresh JWT-SVID is fetched for every login.
func WithSPIFFEJWTAuth(mount, role, endpoint, audience, subject string) Option {
	return func(c *Client) error {
		subjectID, err := spiffeid.FromString(subject)
		if err != nil {
			return fmt.Errorf("invalid SPIFFE ID %q: %w", subject, err)
		}

		return withJWTTokenSource(mount, role, spiffeJWTTokenSource{
			endpoint: endpoint,
			audience: audience,
			subject:  subjectID,
		})(c)
	}
}

func withJWTTokenSource(mount, role string, source jwtTokenSource) Option {
	return func(c *Client) error {
		c.JWTMount = mount
		c.JWTRole = role

		jwtToken, err := source.Token(context.Background())
		if err != nil {
			return err
		}

		secret, err := c.Logical().Write(fmt.Sprintf(certAuthLoginPath, mount), map[string]any{
			"role": role,
			"jwt":  jwtToken,
		})
		if err != nil {
			return fmt.Errorf("error performing jwt auth: %w", err)
		}

		if secret == nil || secret.Auth == nil {
			return errors.New("jwt auth: empty auth response from vault")
		}

		c.SetToken(secret.Auth.ClientToken)

		if c.AuthMethodFunc == nil {
			c.AuthMethodFunc = withJWTTokenSource(mount, role, source)
		}

		return nil
	}
}
