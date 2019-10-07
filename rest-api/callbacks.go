package restapi

import (
	"context"
	json "encoding/json"
	"errors"
	"math/big"

	fcapi "github.com/carbonfive/go-filecoin-rest-api"
	fcapiTypes "github.com/carbonfive/go-filecoin-rest-api/types"

	"github.com/filecoin-project/go-filecoin/actor/builtin/account"
	"github.com/filecoin-project/go-filecoin/actor/builtin/initactor"
	"github.com/filecoin-project/go-filecoin/actor/builtin/miner"
	"github.com/filecoin-project/go-filecoin/actor/builtin/paymentbroker"
	"github.com/filecoin-project/go-filecoin/actor/builtin/storagemarket"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/commands"
	"github.com/filecoin-project/go-filecoin/types"
)

// NewV1Callbacks creates a facapi.V1Callbacks struct with callbacks
// that our porcelain calls
func NewV1Callbacks(ctx context.Context, porc RESTPorcelain) *fcapi.V1Callbacks {
	return &fcapi.V1Callbacks{
		GetActorByID: MakeActorCallback(ctx, porc),
		GetActors:    MakeActorsCallback(ctx, porc),
		GetBlockByID: MakeActorCallback(),
		GetNode:      MakeNodeCallback(porc),
	}
}

// MakeActorCallback makes an Actor callback function
func MakeActorCallback(ctx context.Context, porc RESTPorcelain) func(string) (fcapiTypes.Actor, error) {
	return func(actorId string) (fcapiTypes.Actor, error) {
		addr, err := address.NewFromString(actorId)
		if err != nil {
			return []byte{}, err
		}
		actor, err := porc.ActorGet(ctx, addr)
		if actor == nil {
			return fcapiTypes.Actor{}, errors.New("no actor found")
		}
		apiActor := fcapiTypes.Actor{
			ActorType: "",
			Address:   "",
			Code:      actor.Code,
			Nonce:     uint64(actor.Nonce),
			Balance:   *actor.Balance.AsBigInt(),
			Head:      actor.Head,
		}
		if err != nil {
			return fcapiTypes.Actor{}, err
		}
		return apiActor, nil
	}
}

// MakeActorsCallback makes an Actors callback function
func MakeActorsCallback(ctx context.Context, porc RESTPorcelain) func() ([]byte, error) {
	return func() ([]byte, error) {
		results, err := porc.ActorLs(ctx)
		if err != nil {
			return []byte{}, err
		}
		actors := []*commands.ActorView{}

		for result := range results {
			if result.Error != nil {
				return []byte{}, result.Error
			}

			var av *commands.ActorView

			switch {
			case result.Actor.Empty(): // empty (balance only) actors have no Code.
				av = commands.MakeActorView(result.Actor, result.Address, nil)
			case result.Actor.Code.Equals(types.AccountActorCodeCid):
				av = commands.MakeActorView(result.Actor, result.Address, &account.Actor{})
			case result.Actor.Code.Equals(types.InitActorCodeCid):
				av = commands.MakeActorView(result.Actor, result.Address, &initactor.Actor{})
			case result.Actor.Code.Equals(types.StorageMarketActorCodeCid):
				av = commands.MakeActorView(result.Actor, result.Address, &storagemarket.Actor{})
			case result.Actor.Code.Equals(types.PaymentBrokerActorCodeCid):
				av = commands.MakeActorView(result.Actor, result.Address, &paymentbroker.Actor{})
			case result.Actor.Code.Equals(types.MinerActorCodeCid):
				av = commands.MakeActorView(result.Actor, result.Address, &miner.Actor{})
			case result.Actor.Code.Equals(types.BootstrapMinerActorCodeCid):
				av = commands.MakeActorView(result.Actor, result.Address, &miner.Actor{})
			default:
				av = commands.MakeActorView(result.Actor, result.Address, nil)
			}
			actors = append(actors, av)
		}
		return json.Marshal(actors)
	}
}

// MakeNodeCallback makes a Node callback function
func MakeNodeCallback(porc RESTPorcelain) func() ([]byte, error) {
	return func() (json []byte, err error) {
		ret, err := porc.ConfigGet("wallet.defaultAddress")
		addr := ret.(address.Address)
		if err != nil {
			undef, _ := address.Undef.MarshalJSON()
			return undef, err
		}

		return addr.MarshalJSON()
	}
}
