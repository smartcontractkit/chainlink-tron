package gauntlet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"
	gauntletgen "github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/gauntlet/gauntletgen"
	"golang.org/x/exp/slog"
)

type ProviderRawRPCClient interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

type ProviderGauntletClient interface {
	PostExecute(ctx context.Context, body io.Reader) (GauntletOutput, *gauntletgen.Error, error)
	PostQuery(ctx context.Context, body io.Reader) (GauntletOutput, error)
}

// TODO: when we move it for a specific package, we should standardize
// how to use mockery

//go:generate mockery --name=Provider
type Provider interface {
	ProviderRawRPCClient
	ProviderGauntletClient

	Init(config *Config) error
	Close()
}

// TODO: Changes this to TRON Client in the future
type provider struct {
	rpcHTTP      *rpc.Client
	gauntletHTTP *gauntletgen.ClientWithResponses
}

// Ensure that provider implements the Provider interface at compile-time.
var _ Provider = (*provider)(nil)

func NewProvider(ctx context.Context, config *Config) Provider {
	return &provider{}
}

func (p *provider) Init(config *Config) error {
	slog.Debug("provider: initializing")

	if config.GauntletHTTP != "" {
		slog.Debug("provider: creating Gauntlet HTTP client")
		gauntletHTTP, err := gauntletgen.NewClientWithResponses(config.GauntletHTTP)
		if err != nil {
			return err
		}
		p.gauntletHTTP = gauntletHTTP
	}

	if config.SolidityNode != "" {
		slog.Debug("provider: creating HTTP RPC client")
		rpcHTTP, err := rpc.DialHTTP(config.SolidityNode)
		if err != nil {
			return err
		}
		p.rpcHTTP = rpcHTTP
	}

	return nil
}

func (p *provider) Close() {
	slog.Debug("provider: closing all opened clients")
	// TODO: Implement teardown logic

}

// Raw RPC client implementations.

// CallContext wraps https://pkg.go.dev/github.com/ethereum/go-ethereum/rpc#Client.CallContext
func (p *provider) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	stringArgs := fmt.Sprintf("%+v", args)
	slog.Debug("provider.CallContext(): calling with method (" + method + ") the args (" + stringArgs + ")")

	if p.rpcHTTP == nil {
		return errors.New("the raw RPC HTTP client is not initialized")
	}
	return p.rpcHTTP.CallContext(ctx, result, method, args...)
}

// ----------- G++ client implementation ------------

const contentType = `application/json`

type (
	GauntletOutput *interface{}
)

func (p *provider) PostExecute(ctx context.Context, body io.Reader) (GauntletOutput, *gauntletgen.Error, error) {
	slog.Debug("provider.PostExecute(): executing")

	if p.gauntletHTTP == nil {
		return nil, nil, errors.New("the Gauntlet HTTP client is not initialized")
	}

	response, err := p.gauntletHTTP.PostExecuteWithBodyWithResponse(ctx, contentType, body)
	if err != nil {
		return nil, nil, fmt.Errorf("provider.PostExecute() error: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		fmt.Println("response.error()", response.Body)
		return nil, nil, fmt.Errorf("expecting receive an HTTP StatusCode 200, but got: %d", response.StatusCode())
	}

	if response.JSON200 == nil {
		return nil, nil, fmt.Errorf("expecting an non-nil response.JSON200")
	}

	slog.Debug("p.gauntletHTTP.PostExecuteWithBodyWithResponse()",
		slog.String("response.Status()", response.Status()),
		slog.String("response.JSON200.Output", fmt.Sprintf("%+v", response.JSON200.Output)),
		slog.String("response.JSON200.Error", fmt.Sprintf("%+v", response.JSON200.Error)),
		slog.String("response.Body", fmt.Sprintf("%+v", string(response.Body))),
	)

	return response.JSON200.Output, response.JSON200.Error, nil
}

func (p *provider) PostQuery(ctx context.Context, body io.Reader) (GauntletOutput, error) {
	slog.Debug("provider.PostQuery(): calling")

	if p.gauntletHTTP == nil {
		return nil, errors.New("the Gauntlet HTTP client is not initialized")
	}

	response, err := p.gauntletHTTP.PostQueryWithBodyWithResponse(ctx, contentType, body)
	if err != nil {
		return nil, fmt.Errorf("provider.PostQuery() error: %w", err)
	}

	slog.Debug("p.gauntletHTTP.PostQueryWithBodyWithResponse()",
		slog.String("response.Status()", response.Status()),
		slog.String("response.JSON200.Output", fmt.Sprintf("%+v", response.JSON200.Output)),
		slog.String("response.JSON200.Error", fmt.Sprintf("%+v", response.JSON200.Error)),
		slog.String("response.Body", fmt.Sprintf("%+v", string(response.Body))),
	)

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("expecting receive an HTTP StatusCode 200, but got: %d", response.StatusCode())
	}

	if response.JSON200 == nil {
		return nil, fmt.Errorf("expecting an non-nil response.JSON200")
	}

	if response.JSON200.Output == nil {
		return nil, fmt.Errorf("expecting an non-nil response.JSON200.Output")
	}

	output := response.JSON200.Output
	mOutput, ok := (*output).(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expecting an map[string]interface{} as output but got %T", output)
	}

	data := mOutput["data"]

	return &data, nil
}
