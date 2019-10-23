package restapi

import (
	"context"
	"errors"
	"math/big"
	"reflect"
	"strings"

	v1h "github.com/filecoin-project/go-http-api/handlers/v1"
	apiTypes "github.com/filecoin-project/go-http-api/types"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/actor/builtin/account"
	"github.com/filecoin-project/go-filecoin/actor/builtin/initactor"
	"github.com/filecoin-project/go-filecoin/actor/builtin/miner"
	"github.com/filecoin-project/go-filecoin/actor/builtin/paymentbroker"
	"github.com/filecoin-project/go-filecoin/actor/builtin/storagemarket"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/exec"
	"github.com/filecoin-project/go-filecoin/types"
)

// NewV1Callbacks creates a v1h.V1Callbacks struct with callbacks
// that our porcelain calls
func NewV1Callbacks(ctx context.Context, porc RESTPorcelain) *v1h.Callbacks {
	cb := v1h.Callbacks{
		GetActorByID: MakeActorCallback(ctx, porc),
		GetActors:    MakeActorsCallback(ctx, porc),
		GetNode:      MakeNodeCallback(porc),
	}
	return &cb
}

// MakeActorCallback makes an Actor callback function
func MakeActorCallback(ctx context.Context, porc RESTPorcelain) func(string) (*apiTypes.Actor, error) {
	emptyActor := &apiTypes.Actor{}

	return func(actorId string) (*apiTypes.Actor, error) {
		addr, err := address.NewFromString(actorId)
		if err != nil {
			return emptyActor, err
		}
		act, err := porc.ActorGet(ctx, addr)
		if act == nil {
			return emptyActor, errors.New("no actor found")
		}

		return apiActorFromActor(act, addr.String()), nil
	}
}

// MakeActorsCallback makes an Actors callback function
func MakeActorsCallback(ctx context.Context, porc RESTPorcelain) func() ([]*apiTypes.Actor, error) {
	var emptySet []*apiTypes.Actor
	return func() ([]*apiTypes.Actor, error) {
		results, err := porc.ActorLs(ctx)
		if err != nil {
			return emptySet, err
		}
		var actors []*apiTypes.Actor

		for result := range results {
			if result.Error != nil {
				return emptySet, result.Error
			}

			av := apiActorFromActor(result.Actor, result.Address)
			actors = append(actors, av)
		}
		return actors, nil
	}
}

// MakeNodeCallback makes a Node callback function
func MakeNodeCallback(porc RESTPorcelain) func() (*apiTypes.Node, error) {
	return func() (node *apiTypes.Node, err error) {
		peerID := porc.NetworkGetPeerID()
		addrs := porc.WalletAddresses()

		nd := apiTypes.Node{
			Id:           peerID.String(),
			Addresses:    *addrsAsStrings(&addrs),
			Version:      "",
			Commit:       "",
			Protocol:     apiTypes.Protocol{},
			BitswapStats: apiTypes.BitswapStats{},
		}
		return &nd, nil
	}
}

func addrsAsStrings(addrs *[]address.Address) *[]string {
	res := make([]string, len(*addrs))
	for i, el := range *addrs {
		res[i] = el.String()
	}
	return &res
}

func apiActorFromActor(act *actor.Actor, addr string) *apiTypes.Actor {
	var apiActor *apiTypes.Actor
	switch {
	case act.Empty(): // empty (balance only) actors have no Code.
		apiActor = makeActorView(act, addr, nil)
	case act.Code.Equals(types.AccountActorCodeCid):
		apiActor = makeActorView(act, addr, &account.Actor{})
	case act.Code.Equals(types.InitActorCodeCid):
		apiActor = makeActorView(act, addr, &initactor.Actor{})
	case act.Code.Equals(types.StorageMarketActorCodeCid):
		apiActor = makeActorView(act, addr, &storagemarket.Actor{})
	case act.Code.Equals(types.PaymentBrokerActorCodeCid):
		apiActor = makeActorView(act, addr, &paymentbroker.Actor{})
	case act.Code.Equals(types.MinerActorCodeCid):
		apiActor = makeActorView(act, addr, &miner.Actor{})
	case act.Code.Equals(types.BootstrapMinerActorCodeCid):
		apiActor = makeActorView(act, addr, &miner.Actor{})
	default:
		apiActor = makeActorView(act, addr, nil)
	}
	return apiActor
}

func makeActorView(act *actor.Actor, addr string, actType exec.ExecutableActor) *apiTypes.Actor {
	var actorType string
	var exports map[string]apiTypes.ReadableFunctionSignature
	if actType == nil {
		actorType = "UnknownActor"
	} else {
		actorType = getActorType(actType)
		exports = presentExports(actType.Exports())
	}

	return &apiTypes.Actor{
		ActorType: actorType,
		Address:   addr,
		Code:      act.Code,
		Nonce:     new(big.Int).SetUint64(uint64(act.Nonce)),
		Balance:   act.Balance.AsBigInt(),
		Exports:   exports,
		Head:      act.Head,
	}
}
func makeReadable(f *exec.FunctionSignature) *apiTypes.ReadableFunctionSignature {
	rfs := &apiTypes.ReadableFunctionSignature{
		Params: make([]string, len(f.Params)),
		Return: make([]string, len(f.Return)),
	}
	for i, p := range f.Params {
		rfs.Params[i] = p.String()
	}
	for i, r := range f.Return {
		rfs.Return[i] = r.String()
	}
	return rfs
}

func presentExports(e exec.Exports) map[string]apiTypes.ReadableFunctionSignature {
	rdx := make(map[string]apiTypes.ReadableFunctionSignature)
	for k, v := range e {
		rdx[k] = *makeReadable(v)
	}
	return rdx
}

func getActorType(actType exec.ExecutableActor) string {
	t := reflect.TypeOf(actType).Elem()
	prefixes := strings.Split(t.PkgPath(), "/")
	pkg := prefixes[len(prefixes)-1]

	// strip actor suffix required if package would otherwise be a reserved word
	pkg = strings.TrimSuffix(pkg, "actor")

	return strings.Title(pkg) + t.Name()
}
