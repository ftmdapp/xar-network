package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tdb "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/codec"

	abci "github.com/tendermint/tendermint/abci/types"
)

func TestXardGeneric(t *testing.T) {
	db := tdb.NewMemDB()
	mkdb := tdb.NewMemDB()
	gapp := NewXarApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, mkdb, nil, true, 0)
	setGenesis(gapp)

	modAccPerms := GetMaccPerms()
	require.Equal(t, 11, len(modAccPerms))
}

func TestXardExport(t *testing.T) {
	db := tdb.NewMemDB()
	mkdb := tdb.NewMemDB()
	gapp := NewXarApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, mkdb, nil, true, 0)
	setGenesis(gapp)

	// Making a new app object with the db, so that initchain hasn't been called
	newGapp := NewXarApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, mkdb, nil, true, 0)
	_, _, err := newGapp.ExportAppStateAndValidators(false, []string{})
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}

func TestXardExportZeroHeight(t *testing.T) {
	db := tdb.NewMemDB()
	mkdb := tdb.NewMemDB()
	gapp := NewXarApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, mkdb, nil, true, 0)
	setGenesis(gapp)

	// Making a new app object with the db, so that initchain hasn't been called
	newGapp := NewXarApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, mkdb, nil, true, 0)
	_, _, err := newGapp.ExportAppStateAndValidators(true, []string{})
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}

// ensure that black listed addresses are properly set in bank keeper
func TestBlackListedAddrs(t *testing.T) {
	db := tdb.NewMemDB()
	mkdb := tdb.NewMemDB()
	gapp := NewXarApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, mkdb, nil, true, 0)

	for acc := range maccPerms {
		require.True(t, gapp.bankKeeper.BlacklistedAddr(gapp.supplyKeeper.GetModuleAddress(acc)))
	}
}

func setGenesis(gapp *XarApp) error {
	//genesisState := simapp.NewDefaultGenesisState()
	genesisState := ModuleBasics.DefaultGenesis()
	stateBytes, err := codec.MarshalJSONIndent(gapp.cdc, genesisState)
	if err != nil {
		return err
	}

	// Initialize the chain
	gapp.InitChain(
		abci.RequestInitChain{
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)

	gapp.Commit()
	return nil
}
