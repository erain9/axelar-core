## axelard tx ibc-transfer transfer

Transfer a fungible token through IBC

### Synopsis

Transfer a fungible token through IBC. Timeouts can be specified
as absolute or relative using the "absolute-timeouts" flag. Timeout height can be set by passing in the height string
in the form {revision}-{height} using the "packet-timeout-height" flag. Relative timeouts are added to
the block height and block timestamp queried from the latest consensus state corresponding
to the counterparty channel. Any timeout set to 0 is disabled.

```
axelard tx ibc-transfer transfer [src-port] [src-channel] [receiver] [amount] [flags]
```

### Examples

```
<appd> tx ibc-transfer transfer [src-port] [src-channel] [receiver] [amount]
```

### Options

```
      --absolute-timeouts               Timeout flags are used as absolute timeouts.
  -a, --account-number uint             The account number of the signing account (offline mode only)
  -b, --broadcast-mode string           Transaction broadcasting mode (sync|async|block) (default "block")
      --dry-run                         ignore the --gas flag and perform a simulation of a transaction, but don't broadcast it
      --fee-account string              Fee account pays fees for the transaction instead of deducting from the signer
      --fees string                     Fees to pay along with transaction; eg: 10uatom
      --from string                     Name or address of private key with which to sign
      --gas string                      gas limit to set per-transaction; set to "auto" to calculate sufficient gas automatically (default 200000)
      --gas-adjustment float            adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored  (default 1)
      --gas-prices string               Gas prices in decimal format to determine the transaction fee (e.g. 0.1uatom) (default "0.05uaxl")
      --generate-only                   Build an unsigned transaction and write it to STDOUT (when enabled, the local Keybase is not accessible)
  -h, --help                            help for transfer
      --keyring-backend string          Select keyring's backend (os|file|kwallet|pass|test|memory) (default "file")
      --keyring-dir string              The client Keyring directory; if omitted, the default 'home' directory will be used
      --ledger                          Use a connected Ledger device
      --node string                     <host>:<port> to tendermint rpc interface for this chain (default "tcp://localhost:26657")
      --note string                     Note to add a description to the transaction (previously --memo)
      --offline                         Offline mode (does not allow any online functionality
      --packet-timeout-height string    Packet timeout block height. The timeout is disabled when set to 0-0. (default "0-1000")
      --packet-timeout-timestamp uint   Packet timeout timestamp in nanoseconds. Default is 10 minutes. The timeout is disabled when set to 0. (default 600000000000)
  -s, --sequence uint                   The sequence number of the signing account (offline mode only)
      --sign-mode string                Choose sign mode (direct|amino-json), this is an advanced feature
      --timeout-height uint             Set a block timeout height to prevent the tx from being committed past a certain height
  -y, --yes                             Skip tx broadcasting prompt confirmation (default true)
```

### Options inherited from parent commands

```
      --chain-id string     The network chain ID (default "axelar")
      --home string         directory for config and data (default "$HOME/.axelar")
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic) (default "info")
      --output string       Output format (text|json) (default "text")
      --trace               print out full stack trace on errors
```

### SEE ALSO

- [axelard tx ibc-transfer](axelard_tx_ibc-transfer.md)	 - IBC fungible token transfer transaction subcommands
