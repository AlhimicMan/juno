package postgresql

import (
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/desmos-labs/juno/config"
	"github.com/desmos-labs/juno/db"
	"github.com/desmos-labs/juno/types"
	_ "github.com/lib/pq" // nolint
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

// type check to ensure interface is properly implemented
var _ db.Database = Database{}

// Database defines a wrapper around a SQL database and implements functionality
// for data aggregation and exporting.
type Database struct {
	Sql   *sql.DB
	Codec *codec.Codec
}

// OpenDB opens a database connection with the given database connection info
// from config. It returns a database connection handle or an error if the
// connection fails.
func Builder(cfg config.PostgreSQLConfig, codec *codec.Codec) (*db.Database, error) {
	sslMode := "disable"
	if cfg.SSLMode != "" {
		sslMode = cfg.SSLMode
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, sslMode,
	)

	if cfg.Password != "" {
		connStr += fmt.Sprintf(" password=%s", cfg.Password)
	}

	postgresDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	var database db.Database = Database{Sql: postgresDb, Codec: codec}
	return &database, nil
}

// LastBlockHeight returns the latest block stored.
func (db Database) LastBlockHeight() (int64, error) {
	var height int64
	err := db.Sql.QueryRow("SELECT coalesce(MAX(height),0) AS height FROM block;").Scan(&height)
	return height, err
}

// HasBlock returns true if a block by height exists. An error should never be
// returned.
func (db Database) HasBlock(height int64) (bool, error) {
	var res bool
	err := db.Sql.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM block WHERE height = $1);",
		height,
	).Scan(&res)

	return res, err
}

// SetBlock stores a block and returns the resulting record ID. An error is
// returned if the operation fails.
func (db Database) SaveBlock(block *tmctypes.ResultBlock, totalGas, preCommits uint64) error {
	var id uint64

	sqlStatement := `
	INSERT INTO block (height, hash, num_txs, total_gas, proposer_address, pre_commits, timestamp)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id;
	`

	return db.Sql.QueryRow(
		sqlStatement,
		block.Block.Height, block.Block.Hash().String(), block.Block.NumTxs,
		totalGas, block.Block.ProposerAddress.String(), preCommits, block.Block.Time,
	).Scan(&id)
}

// SetTx stores a transaction and returns the resulting record ID. An error is
// returned if the operation fails.
func (db Database) SaveTx(tx types.Tx) error {
	var id uint64

	sqlStatement := `
	INSERT INTO transaction (timestamp, gas_wanted, gas_used, height, txhash, events, messages, fee, signatures, memo)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING id;
	`

	stdTx, ok := tx.Tx.(auth.StdTx)
	if !ok {
		return fmt.Errorf("unsupported tx type: %T", tx.Tx)
	}

	eventsBz, err := db.Codec.MarshalJSON(tx.Events)
	if err != nil {
		return fmt.Errorf("failed to JSON encode tx events: %s", err)
	}

	msgsBz, err := db.Codec.MarshalJSON(stdTx.GetMsgs())
	if err != nil {
		return fmt.Errorf("failed to JSON encode tx messages: %s", err)
	}

	feeBz, err := db.Codec.MarshalJSON(stdTx.Fee)
	if err != nil {
		return fmt.Errorf("failed to JSON encode tx fee: %s", err)
	}

	// convert Tendermint signatures into a more human-readable format
	sigs := make([]signature, len(stdTx.GetSignatures()), len(stdTx.GetSignatures()))
	for i, sig := range stdTx.GetSignatures() {
		addr, err := sdk.AccAddressFromHex(sig.Address().String())
		if err != nil {
			return fmt.Errorf("failed to convert account address %s: %s\n", sig.Address(), err)
		}

		pubkey, err := sdk.Bech32ifyAccPub(sig.PubKey) // nolint: typecheck
		if err != nil {
			return fmt.Errorf("failed to convert account public key %X: %s\n", sig.PubKey.Bytes(), err)
		}

		sigs[i] = signature{
			Address:   addr.String(),
			Signature: base64.StdEncoding.EncodeToString(sig.Signature),
			Pubkey:    pubkey,
		}
	}

	sigsBz, err := db.Codec.MarshalJSON(sigs)
	if err != nil {
		return fmt.Errorf("failed to JSON encode tx signatures: %s", err)
	}

	return db.Sql.QueryRow(
		sqlStatement,
		tx.Timestamp, tx.GasWanted, tx.GasUsed, tx.Height, tx.TxHash, string(eventsBz),
		string(msgsBz), string(feeBz), string(sigsBz), stdTx.GetMemo(),
	).Scan(&id)
}

// HasValidator returns true if a given validator by HEX address exists. An
// error should never be returned.
func (db Database) HasValidator(addr string) (bool, error) {
	var res bool
	err := db.Sql.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM validator WHERE address = $1);",
		addr,
	).Scan(&res)

	return res, err
}

// SetValidator stores a validator if it does not already exist. An error is
// returned if the operation fails.
func (db Database) SaveValidator(addr, pk string) error {
	_, err := db.Sql.Exec(
		"INSERT INTO validator (address, consensus_pubkey) VALUES ($1, $2) ON CONFLICT DO NOTHING RETURNING id;",
		addr, pk,
	)

	return err
}

// SetPreCommit stores a validator's pre-commit and returns the resulting record
// ID. An error is returned if the operation fails.
func (db Database) SavePreCommit(pc *tmtypes.CommitSig, votingPower, proposerPriority int64) error {
	var id uint64

	sqlStatement := `
	INSERT INTO pre_commit (height, round, validator_address, timestamp, voting_power, proposer_priority)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id;
	`

	return db.Sql.QueryRow(
		sqlStatement,
		pc.Height, pc.Round, pc.ValidatorAddress.String(), pc.Timestamp, votingPower, proposerPriority,
	).Scan(&id)
}

type signature struct {
	Address   string `json:"address,omitempty"`
	Pubkey    string `json:"pubkey,omitempty"`
	Signature string `json:"signature,omitempty"`
}
