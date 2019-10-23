package restapi

import (
	"context"

	server "github.com/filecoin-project/go-http-api"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/state"
)

// RESTPorcelain is the subset of porcelain and plumbing commands needed for the
// REST API callbacks
type RESTPorcelain interface {
	ActorGet(context.Context, address.Address) (*actor.Actor, error)
	ActorLs(context.Context) (<-chan state.GetAllActorsResult, error)
	ConfigGet(string) (interface{}, error)
	NetworkGetPeerID() peer.ID
	WalletAddresses() []address.Address
}

// Launch creates and launches the REST API, serving HTTP from the given
// port
func Launch(ctx context.Context, porc RESTPorcelain, port int) *server.HTTPAPI {
	cb := NewV1Callbacks(ctx, porc)
	config := server.Config{
		Port: port,
		//TLSCertPath: os.Getenv("TLS_CERT_PATH"),
		//TLSKeyPath:  os.Getenv("TLS_KEY_PATH"),
	}
	api := server.NewHTTPAPI(ctx, cb, config)
	api.Run()
	return api
}
