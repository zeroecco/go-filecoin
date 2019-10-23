package restapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"testing"

	apiTypes "github.com/filecoin-project/go-http-api/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/address"
	. "github.com/filecoin-project/go-filecoin/rest-api"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/testhelpers"
	"github.com/filecoin-project/go-filecoin/types"
)

func TestLaunchHappyPath(t *testing.T) {
	actor1 := testhelpers.RequireNewAccountActor(t, types.NewAttoFILFromFIL(12))
	defaultAddr := address.TestAddress
	pid, err := testhelpers.RandPeerID()
	require.NoError(t, err)
	porc := TestPorcelain{
		actors:     []*actor.Actor{actor1},
		nodeID:     pid,
		walletAddr: defaultAddr,
	}

	port, err := testhelpers.GetFreePort()
	require.NoError(t, err)
	api := Launch(context.Background(), &porc, port)
	defer func() {
		err := api.Shutdown()
		if err != nil {
			t.Log(err)
		}
	}()

	t.Run("actor endpoint returns actor", func(t *testing.T) {
		path := fmt.Sprintf("actors/%s", defaultAddr.String())
		resp := RequireGetResponseBody(t, port, path)
		var act apiTypes.Actor
		require.NoError(t, json.Unmarshal(resp, &act))

		assert.True(t, actor1.Code.Equals(act.Code))
		assert.True(t, actor1.Head.Equals(act.Head))
		assert.Equal(t, new(big.Int).SetUint64(uint64(actor1.Nonce)), act.Nonce)
		assert.Equal(t, actor1.Balance.AsBigInt(), act.Balance)
	})

	t.Run("node endpoint returns correct value", func(t *testing.T) {
		resp := RequireGetResponseBody(t, port, "control/node")
		var node apiTypes.Node

		require.NoError(t, json.Unmarshal(resp, &node))
		assert.Equal(t, node.Kind, "node")
		assert.Equal(t, node.Id, pid.String())
		assert.Len(t, node.Addresses, 1)
		assert.Equal(t, node.Addresses[0], defaultAddr.String())
	})

}

type TestPorcelain struct {
	nodeID                      peer.ID
	walletAddr                  address.Address
	actors                      []*actor.Actor
	failActorGet, failConfigGet bool
}

// ActorGet returns error if the porcelain is configured to fail, or if there are no actors.
// Otherwise it returns just the first actor.
func (tp *TestPorcelain) ActorGet(ctx context.Context, addr address.Address) (*actor.Actor, error) {
	if tp.failActorGet {
		return nil, errors.New("actorGet failed")
	}
	if len(tp.actors) == 0 {
		return nil, errors.New("no actors")
	}
	return tp.actors[0], nil
}

// ActorLs returns all actors as a channel
func (tp *TestPorcelain) ActorLs(ctx context.Context) (<-chan state.GetAllActorsResult, error) {
	out := make(chan state.GetAllActorsResult)
	defer close(out)
	for _, testActor := range tp.actors {
		select {
		case <-ctx.Done():
			out <- state.GetAllActorsResult{
				Error: ctx.Err(),
			}
			return out, ctx.Err()
		default:
			out <- state.GetAllActorsResult{
				Address: address.TestAddress.String(),
				Actor:   testActor,
			}
		}
	}
	return out, nil
}

func (tp *TestPorcelain) ConfigGet(config string) (interface{}, error) {
	if tp.failConfigGet {
		return "", errors.New("ConfigGet failed")
	}
	if config == "wallet.defaultAddress" {
		return tp.walletAddr, nil
	}
	return "", errors.New("bad config call")
}

func (tp *TestPorcelain) NetworkGetPeerID() peer.ID {
	return tp.nodeID
}
func (tp *TestPorcelain) WalletAddresses() []address.Address {
	return []address.Address{tp.walletAddr}
}
func RequireGetResponseBody(t *testing.T, port int, path string) []byte {
	uri := fmt.Sprintf("http://localhost:%d/api/filecoin/v1/%s", port, path)
	resp, err := http.Get(uri)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	return body
}
