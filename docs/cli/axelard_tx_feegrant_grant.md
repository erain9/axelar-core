## axelard tx feegrant grant

Grant Fee allowance to an address

### Synopsis

Grant authorization to pay fees from your address. Note, the'--from' flag is
ignored as it is implied from \[granter\].

Examples:
<appd> tx feegrant grant cosmos1skjw... cosmos1skjw... --spend-limit 100stake --expiration 2022-01-30T15:04:05Z or
<appd> tx feegrant grant cosmos1skjw... cosmos1skjw... --spend-limit 100stake --period 3600 --period-limit 10stake --expiration 36000 or
<appd> tx feegrant grant cosmos1skjw... cosmos1skjw... --spend-limit 100stake --expiration 2022-01-30T15:04:05Z
--allowed-messages "/cosmos.gov.v1beta1.MsgSubmitProposal,/cosmos.gov.v1beta1.MsgVote"

```
axelard tx feegrant grant [granter_key_or_address] [grantee] [flags]
```

### Options

```
  -a, --account-number uint        The account number of the signing account (offline mode only)
      --allowed-messages strings   Set of allowed messages for fee allowance
  -b, --broadcast-mode string      Transaction broadcasting mode (sync|async|block) (default "block")
      --dry-run                    ignore the --gas flag and perform a simulation of a transaction, but don't broadcast it
      --expiration string          The RFC 3339 timestamp after which the grant expires for the user
      --fee-account string         Fee account pays fees for the transaction instead of deducting from the signer
      --fees string                Fees to pay along with transaction; eg: 10uatom
      --from string                Name or address of private key with which to sign
      --gas string                 gas limit to set per-transaction; set to "auto" to calculate sufficient gas automatically (default 200000)
      --gas-adjustment float       adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored  (default 1)
      --gas-prices string          Gas prices in decimal format to determine the transaction fee (e.g. 0.1uatom) (default "0.05uaxl")
      --generate-only              Build an unsigned transaction and write it to STDOUT (when enabled, the local Keybase is not accessible)
  -h, --help                       help for grant
      --keyring-backend string     Select keyring's backend (os|file|kwallet|pass|test|memory) (default "file")
      --keyring-dir string         The client Keyring directory; if omitted, the default 'home' directory will be used
      --ledger                     Use a connected Ledger device
      --node string                <host>:<port> to tendermint rpc interface for this chain (default "tcp://localhost:26657")
      --note string                Note to add a description to the transaction (previously --memo)
      --offline                    Offline mode (does not allow any online functionality
      --period int                 period specifies the time duration in which period_spend_limit coins can be spent before that allowance is reset
      --period-limit string        period limit specifies the maximum number of coins that can be spent in the period
  -s, --sequence uint              The sequence number of the signing account (offline mode only)
      --sign-mode string           Choose sign mode (direct|amino-json), this is an advanced feature
      --spend-limit string         Spend limit specifies the max limit can be used, if not mentioned there is no limit
      --timeout-height uint        Set a block timeout height to prevent the tx from being committed past a certain height
  -y, --yes                        Skip tx broadcasting prompt confirmation (default true)
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

- [axelard tx feegrant](axelard_tx_feegrant.md)	 - Feegrant transactions subcommands
