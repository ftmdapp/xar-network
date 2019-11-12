package app

import (
	"io"
	"os"

	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/supply"

	"github.com/cosmos/modules/incubator/nft"

	//Public issuance
	"github.com/xar-network/xar-network/x/authority"
	"github.com/xar-network/xar-network/x/interest"
	"github.com/xar-network/xar-network/x/issue"
	"github.com/xar-network/xar-network/x/issuer"
	"github.com/xar-network/xar-network/x/liquidityprovider"

	"github.com/xar-network/xar-network/x/auction"
	"github.com/xar-network/xar-network/x/csdt"
	"github.com/xar-network/xar-network/x/liquidator"
	"github.com/xar-network/xar-network/x/pricefeed"

	//Proof of existence
	"github.com/xar-network/xar-network/x/record"

	//uniswap
	"github.com/xar-network/xar-network/x/uniswap"
)

const appName = "xar"

var (
	// default home directories for xarcli
	DefaultCLIHome = os.ExpandEnv("$HOME/.xarcli")
	// default home directories for xard
	DefaultNodeHome = os.ExpandEnv("$HOME/.xard")

	// The module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		genutil.AppModuleBasic{},
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(paramsclient.ProposalHandler, distr.ProposalHandler),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		supply.AppModuleBasic{},
		issue.AppModuleBasic{},
		nft.AppModuleBasic{},
		auction.AppModuleBasic{},
		csdt.AppModuleBasic{},
		liquidator.AppModuleBasic{},
		pricefeed.AppModuleBasic{},
		record.AppModuleBasic{},
		interest.AppModuleBasic{},
		liquidityprovider.AppModuleBasic{},
		issuer.AppModuleBasic{},
		authority.AppModule{},
		uniswap.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		auth.FeeCollectorName:        nil,
		distr.ModuleName:             nil,
		mint.ModuleName:              {supply.Minter},
		staking.BondedPoolName:       {supply.Burner, supply.Staking},
		staking.NotBondedPoolName:    {supply.Burner, supply.Staking},
		gov.ModuleName:               {supply.Burner},
		liquidityprovider.ModuleName: {supply.Minter, supply.Burner},
		interest.ModuleName:          {supply.Minter},
	}
)

// MakeCodec creates the application codec. The codec is sealed before it is
// returned.
func MakeCodec() *codec.Codec {
	var cdc = codec.New()

	ModuleBasics.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	codec.RegisterEvidences(cdc)

	return cdc.Seal()
}

// xarApp extended ABCI application
type xarApp struct {
	*bam.BaseApp
	cdc *codec.Codec

	invCheckPeriod uint

	// keys to access the substores
	keys  map[string]*sdk.KVStoreKey
	tkeys map[string]*sdk.TransientStoreKey

	// keepers
	accountKeeper  auth.AccountKeeper
	bankKeeper     bank.Keeper
	supplyKeeper   supply.Keeper
	stakingKeeper  staking.Keeper
	slashingKeeper slashing.Keeper
	mintKeeper     mint.Keeper
	distrKeeper    distr.Keeper
	govKeeper      gov.Keeper
	crisisKeeper   crisis.Keeper
	paramsKeeper   params.Keeper

	// app specific keepers
	auctionKeeper    auction.Keeper
	csdtKeeper       csdt.Keeper
	liquidatorKeeper liquidator.Keeper
	pricefeedKeeper  pricefeed.Keeper
	issueKeeper      issue.Keeper
	recordKeeper     record.Keeper

	NFTKeeper nft.Keeper

	interestKeeper  interest.Keeper
	lpKeeper        liquidityprovider.Keeper
	issuerKeeper    issuer.Keeper
	authorityKeeper authority.Keeper

	uniswapKeeper uniswap.Keeper

	// the module manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager
}

// NewXarApp returns a reference to an initialized xarApp.
func NewXarApp(
	logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool,
	invCheckPeriod uint, baseAppOptions ...func(*bam.BaseApp),
) *xarApp {

	cdc := MakeCodec()

	bApp := bam.NewBaseApp(appName, logger, db, auth.DefaultTxDecoder(cdc), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetAppVersion(version.Version)

	keys := sdk.NewKVStoreKeys(bam.MainStoreKey, auth.StoreKey, staking.StoreKey,
		supply.StoreKey, mint.StoreKey, distr.StoreKey, slashing.StoreKey,
		gov.StoreKey, params.StoreKey, issue.StoreKey, pricefeed.StoreKey,
		auction.StoreKey, csdt.StoreKey, liquidator.StoreKey, nft.StoreKey,
		interest.StoreKey, authority.StoreKey, issuer.StoreKey,
		record.StoreKey, uniswap.ModuleName,
	)

	tkeys := sdk.NewTransientStoreKeys(staking.TStoreKey, params.TStoreKey)

	app := &xarApp{
		BaseApp:        bApp,
		cdc:            cdc,
		invCheckPeriod: invCheckPeriod,
		keys:           keys,
		tkeys:          tkeys,
	}

	// init params keeper and subspaces
	app.paramsKeeper = params.NewKeeper(app.cdc, keys[params.StoreKey], tkeys[params.TStoreKey], params.DefaultCodespace)
	authSubspace := app.paramsKeeper.Subspace(auth.DefaultParamspace)
	bankSubspace := app.paramsKeeper.Subspace(bank.DefaultParamspace)
	stakingSubspace := app.paramsKeeper.Subspace(staking.DefaultParamspace)
	mintSubspace := app.paramsKeeper.Subspace(mint.DefaultParamspace)
	distrSubspace := app.paramsKeeper.Subspace(distr.DefaultParamspace)
	slashingSubspace := app.paramsKeeper.Subspace(slashing.DefaultParamspace)
	govSubspace := app.paramsKeeper.Subspace(gov.DefaultParamspace).WithKeyTable(gov.ParamKeyTable())
	crisisSubspace := app.paramsKeeper.Subspace(crisis.DefaultParamspace)

	issueSubspace := app.paramsKeeper.Subspace(issue.DefaultParamspace)
	csdtSubspace := app.paramsKeeper.Subspace(csdt.DefaultParamspace)
	liquidatorSubspace := app.paramsKeeper.Subspace(liquidator.DefaultParamspace)
	recordSubspace := app.paramsKeeper.Subspace(record.DefaultParamspace)
	interestSubspace := app.paramsKeeper.Subspace(interest.DefaultParamspace)

	uniswapSubspace := app.paramsKeeper.Subspace(uniswap.DefaultParamSpace)

	// add keepers
	app.accountKeeper = auth.NewAccountKeeper(app.cdc, keys[auth.StoreKey], authSubspace, auth.ProtoBaseAccount)
	bankKeeper := bank.NewBaseKeeper(app.accountKeeper, bankSubspace, bank.DefaultCodespace, app.ModuleAccountAddrs())
	app.supplyKeeper = supply.NewKeeper(app.cdc, keys[supply.StoreKey], app.accountKeeper, app.bankKeeper, maccPerms)
	stakingKeeper := staking.NewKeeper(app.cdc, keys[staking.StoreKey], app.supplyKeeper, stakingSubspace, staking.DefaultCodespace)
	app.mintKeeper = mint.NewKeeper(app.cdc, keys[mint.StoreKey], mintSubspace, &stakingKeeper, app.supplyKeeper, auth.FeeCollectorName)
	app.distrKeeper = distr.NewKeeper(app.cdc, keys[distr.StoreKey], distrSubspace, &stakingKeeper,
		app.supplyKeeper, distr.DefaultCodespace, auth.FeeCollectorName, app.ModuleAccountAddrs())
	app.slashingKeeper = slashing.NewKeeper(app.cdc, keys[slashing.StoreKey], &stakingKeeper, slashingSubspace, slashing.DefaultCodespace)
	app.crisisKeeper = crisis.NewKeeper(crisisSubspace, invCheckPeriod, app.supplyKeeper, auth.FeeCollectorName)

	app.NFTKeeper = nft.NewKeeper(app.cdc, keys[nft.StoreKey])
	app.issueKeeper = issue.NewKeeper(keys[issue.StoreKey], issueSubspace, app.bankKeeper, issue.DefaultCodespace)
	app.pricefeedKeeper = pricefeed.NewKeeper(keys[pricefeed.StoreKey], app.cdc, pricefeed.DefaultCodespace)
	app.recordKeeper = record.NewKeeper(app.cdc, keys[record.StoreKey], recordSubspace, record.DefaultCodespace)
	app.csdtKeeper = csdt.NewKeeper(app.cdc, keys[csdt.StoreKey], csdtSubspace, app.pricefeedKeeper, app.bankKeeper)
	app.auctionKeeper = auction.NewKeeper(app.cdc, app.csdtKeeper, keys[auction.StoreKey])
	app.liquidatorKeeper = liquidator.NewKeeper(app.cdc, keys[liquidator.StoreKey], liquidatorSubspace, app.csdtKeeper, app.auctionKeeper, app.csdtKeeper)

	app.interestKeeper = interest.NewKeeper(app.cdc, keys[interest.StoreKey], interestSubspace, app.supplyKeeper, auth.FeeCollectorName)
	app.lpKeeper = liquidityprovider.NewKeeper(app.accountKeeper, app.supplyKeeper)
	app.issuerKeeper = issuer.NewKeeper(keys[issuer.StoreKey], app.lpKeeper, app.interestKeeper)
	app.authorityKeeper = authority.NewKeeper(keys[authority.StoreKey], app.issuerKeeper)

	app.uniswapKeeper = uniswap.NewKeeper(app.cdc, keys[uniswap.ModuleName], uniswapSubspace)

	// register the proposal types
	govRouter := gov.NewRouter()
	govRouter.AddRoute(gov.RouterKey, gov.ProposalHandler).
		AddRoute(params.RouterKey, params.NewParamChangeProposalHandler(app.paramsKeeper)).
		AddRoute(distr.RouterKey, distr.NewCommunityPoolSpendProposalHandler(app.distrKeeper))
	app.govKeeper = gov.NewKeeper(app.cdc, keys[gov.StoreKey], govSubspace,
		app.supplyKeeper, &stakingKeeper, gov.DefaultCodespace, govRouter)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.stakingKeeper = *stakingKeeper.SetHooks(
		staking.NewMultiStakingHooks(app.distrKeeper.Hooks(), app.slashingKeeper.Hooks()),
	)

	app.bankKeeper = *bankKeeper.SetHooks(
		NewBankHooks(app.boxKeeper.Hooks(), app.issueKeeper.Hooks(), app.accMustMemoKeeper.Hooks()),
	)

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		genutil.NewAppModule(app.accountKeeper, app.stakingKeeper, app.BaseApp.DeliverTx),
		auth.NewAppModule(app.accountKeeper),
		bank.NewAppModule(app.bankKeeper, app.accountKeeper),
		crisis.NewAppModule(&app.crisisKeeper),
		supply.NewAppModule(app.supplyKeeper, app.accountKeeper),
		distr.NewAppModule(app.distrKeeper, app.supplyKeeper),
		gov.NewAppModule(app.govKeeper, app.supplyKeeper),
		mint.NewAppModule(app.mintKeeper),
		slashing.NewAppModule(app.slashingKeeper, app.stakingKeeper),
		staking.NewAppModule(app.stakingKeeper, app.accountKeeper, app.supplyKeeper),

		nft.NewAppModule(app.NFTKeeper),
		issue.NewAppModule(app.issueKeeper, app.accountKeeper),
		auction.NewAppModule(app.auctionKeeper),
		csdt.NewAppModule(app.csdtKeeper),
		liquidator.NewAppModule(app.liquidatorKeeper),
		pricefeed.NewAppModule(app.pricefeedKeeper),
		record.NewAppModule(app.recordKeeper),

		interest.NewAppModule(app.interestKeeper),
		liquidityprovider.NewAppModule(app.lpKeeper),
		issuer.NewAppModule(app.issuerKeeper),
		authority.NewAppModule(app.authorityKeeper),
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	app.mm.SetOrderBeginBlockers(
		interest.ModuleName,
		mint.ModuleName,
		distr.ModuleName,
		slashing.ModuleName,
	)

	app.mm.SetOrderEndBlockers(
		crisis.ModuleName,
		gov.ModuleName,
		staking.ModuleName,
		pricefeed.ModuleName,
		authority.ModuleName,
		interest.ModuleName,
		issue.ModuleName,
		authority.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	app.mm.SetOrderInitGenesis(
		distr.ModuleName, staking.ModuleName, auth.ModuleName, bank.ModuleName,
		slashing.ModuleName, gov.ModuleName, mint.ModuleName, supply.ModuleName,
		crisis.ModuleName, issue.ModuleName,
		auction.ModuleName, csdt.ModuleName, liquidator.ModuleName, pricefeed.ModuleName,
		interest.ModuleName, authority.ModuleName, liquidityprovider.ModuleName, issuer.ModuleName,
		nft.ModuleName, record.ModuleName, genutil.ModuleName,
	)

	app.mm.RegisterInvariants(&app.crisisKeeper)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter())

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: This is not required for apps that don't use the simulator for fuzz testing
	// transactions.
	app.sm = module.NewSimulationManager(
		auth.NewAppModule(app.accountKeeper),
		bank.NewAppModule(app.bankKeeper, app.accountKeeper),
		supply.NewAppModule(app.supplyKeeper, app.accountKeeper),
		gov.NewAppModule(app.govKeeper, app.supplyKeeper),
		mint.NewAppModule(app.mintKeeper),
		distr.NewAppModule(app.distrKeeper, app.supplyKeeper),
		staking.NewAppModule(app.stakingKeeper, app.accountKeeper, app.supplyKeeper),
		slashing.NewAppModule(app.slashingKeeper, app.stakingKeeper),

		issue.NewAppModule(app.issueKeeper, app.accountKeeper),

		// TODO: Add simulation keepers
		/*

			record.NewAppModule(app.recordKeeper),
			auction.NewAppModule(app.auctionKeeper),
			csdt.NewAppModule(app.csdtKeeper),
			liquidator.NewAppModule(app.liquidatorKeeper),
			pricefeed.NewAppModule(app.pricefeedKeeper),
			nft.NewAppModule(app.NFTKeeper),

		*/
	)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetAnteHandler(auth.NewAnteHandler(app.accountKeeper, app.supplyKeeper, auth.DefaultSigVerificationGasConsumer))
	app.SetEndBlocker(app.EndBlocker)

	if loadLatest {
		err := app.LoadLatestVersion(app.keys[bam.MainStoreKey])
		if err != nil {
			cmn.Exit(err.Error())
		}
	}
	return app
}

// application updates every begin block
func (app *xarApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// application updates every end block
func (app *xarApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// application update at chain initialization
func (app *xarApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	app.cdc.MustUnmarshalJSON(req.AppStateBytes, &genesisState)
	return app.mm.InitGenesis(ctx, genesisState)
}

// load a particular height
func (app *xarApp) LoadHeight(height int64) error {
	return app.LoadVersion(height, app.keys[bam.MainStoreKey])
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *xarApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[supply.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// Codec returns the application's sealed codec.
func (app *xarApp) Codec() *codec.Codec {
	return app.cdc
}

// GetMaccPerms returns a mapping of the application's module account permissions.
func GetMaccPerms() map[string][]string {
	modAccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		modAccPerms[k] = v
	}
	return modAccPerms
}
