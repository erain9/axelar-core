package rest

import (
	"encoding/hex"
	"net/http"

	"github.com/axelarnetwork/axelar-core/x/bitcoin/types"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/gorilla/mux"

	clientUtils "github.com/axelarnetwork/axelar-core/utils"
	tss "github.com/axelarnetwork/axelar-core/x/tss/exported"
)

// rest routes
const (
	TxLink                        = "link"
	TxConfirmTx                   = "confirm"
	TxCreatePendingTransfersTx    = "create-pending-transfers-tx"
	TxCreateMasterConsolidationTx = "create-master-consolidation-tx"
	TxSignTx                      = "sign-tx"
	TxRegisterExternalKey         = "register-external-key"
	TxSubmitExternalSignature     = "submit-external-signature"

	QueryDepositAddress       = "deposit-address"
	QueryConsolidationAddress = "consolidation-address"
	QueryMinOutputAmount      = "min-output-amount"
	QueryNextKeyID            = "next-key-id"
	QueryLatestTx             = "latest-tx"
	QuerySignedTx             = "signed-tx"
)

// RegisterRoutes registers this module's REST routes with the given router
func RegisterRoutes(cliCtx client.Context, r *mux.Router) {
	registerTx := clientUtils.RegisterTxHandlerFn(r, types.RestRoute)
	registerTx(TxHandlerLink(cliCtx), TxLink, clientUtils.PathVarChain)
	registerTx(TxHandlerConfirmTx(cliCtx), TxConfirmTx)
	registerTx(TxHandlerCreatePendingTransfersTx(cliCtx), TxCreatePendingTransfersTx)
	registerTx(TxHandlerCreateMasterConsolidationTx(cliCtx), TxCreateMasterConsolidationTx)
	registerTx(TxHandlerSignTx(cliCtx), TxSignTx)
	registerTx(TxHandlerRegisterExternalKey(cliCtx), TxRegisterExternalKey)
	registerTx(TxHandlerSubmitExternalSignature(cliCtx), TxSubmitExternalSignature)

	registerQuery := clientUtils.RegisterQueryHandlerFn(r, types.RestRoute)
	registerQuery(QueryHandlerDepositAddress(cliCtx), QueryDepositAddress, clientUtils.PathVarChain, clientUtils.PathVarEthereumAddress)
	registerQuery(QueryHandlerConsolidationAddress(cliCtx), QueryConsolidationAddress)
	registerQuery(QueryHandlerNextKeyID(cliCtx), QueryNextKeyID, clientUtils.PathVarKeyRole)
	registerQuery(QueryHandlerMinOutputAmount(cliCtx), QueryMinOutputAmount)
	registerQuery(QueryHandlerLatestTx(cliCtx), QueryLatestTx, clientUtils.PathVarKeyRole)
	registerQuery(QueryHandlerSignedTx(cliCtx), QuerySignedTx, clientUtils.PathVarTxID)
}

// ReqLink represents a request to link a cross-chain address to a Bitcoin address
type ReqLink struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`
	Address string       `json:"address" yaml:"address"`
}

// ReqConfirmOutPoint represents a request to confirm a Bitcoin outpoint
type ReqConfirmOutPoint struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`
	TxInfo  string       `json:"tx_info" yaml:"tx_info"`
}

// ReqCreatePendingTransfersTx represents a request to create a secondary key consolidation transaction handling all pending transfers
type ReqCreatePendingTransfersTx struct {
	BaseReq         rest.BaseReq `json:"base_req" yaml:"base_req"`
	KeyID           string       `json:"key_id" yaml:"key_id"`
	MasterKeyAmount string       `json:"master_key_amount" yaml:"master_key_amount"`
}

// ReqCreateMasterConsolidationTx represents a request to create a master key consolidation transaction
type ReqCreateMasterConsolidationTx struct {
	BaseReq            rest.BaseReq `json:"base_req" yaml:"base_req"`
	KeyID              string       `json:"key_id" yaml:"key_id"`
	SecondaryKeyAmount string       `json:"secondary_key_amount" yaml:"secondary_key_amount"`
}

// ReqSignTx represents a request to sign a consolidation transaction
type ReqSignTx struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`
	KeyRole string       `json:"key_role" yaml:"key_role"`
}

// ReqRegisterExternalKey represents a request to register an external key
type ReqRegisterExternalKey struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`
	KeyID   string       `json:"key_id" yaml:"key_id"`
	PubKey  string       `json:"pub_key" yaml:"pub_key"`
}

// ReqSubmitExternalSignature represents a request to submit a signature from an external key
type ReqSubmitExternalSignature struct {
	BaseReq   rest.BaseReq `json:"base_req" yaml:"base_req"`
	KeyID     string       `json:"key_id" yaml:"key_id"`
	Signature string       `json:"signature" yaml:"signature"`
	SigHash   string       `json:"sig_hash" yaml:"sig_hash"`
}

// TxHandlerLink returns the handler to link a Bitcoin address to a cross-chain address
func TxHandlerLink(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReqLink
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddr, ok := clientUtils.ExtractReqSender(w, req.BaseReq)
		if !ok {
			return
		}

		msg := types.NewLinkRequest(fromAddr, req.Address, mux.Vars(r)[clientUtils.PathVarChain])
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

// TxHandlerConfirmTx returns the handler to confirm a tx outpoint
func TxHandlerConfirmTx(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReqConfirmOutPoint
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddr, ok := clientUtils.ExtractReqSender(w, req.BaseReq)
		if !ok {
			return
		}

		var out types.OutPointInfo
		if err := cliCtx.LegacyAmino.UnmarshalJSON([]byte(req.TxInfo), &out); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := types.NewConfirmOutpointRequest(fromAddr, out)

		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

// TxHandlerCreatePendingTransfersTx returns the handler to create a secondary key consolidation transaction handling all pending transfers
func TxHandlerCreatePendingTransfersTx(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReqCreatePendingTransfersTx
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddr, ok := clientUtils.ExtractReqSender(w, req.BaseReq)
		if !ok {
			return
		}

		masterKeyAmount, err := types.ParseSatoshi(req.MasterKeyAmount)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := types.NewCreatePendingTransfersTxRequest(fromAddr, req.KeyID, btcutil.Amount(masterKeyAmount.Amount.Int64()))
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

// TxHandlerCreateMasterConsolidationTx returns the handler to create a master key consolidation transaction
func TxHandlerCreateMasterConsolidationTx(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReqCreateMasterConsolidationTx
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddr, ok := clientUtils.ExtractReqSender(w, req.BaseReq)
		if !ok {
			return
		}

		secondaryKeyAmount, err := types.ParseSatoshi(req.SecondaryKeyAmount)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := types.NewCreateMasterTxRequest(fromAddr, req.KeyID, btcutil.Amount(secondaryKeyAmount.Amount.Int64()))
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

// TxHandlerSignTx returns the handler to sign a consolidation transaction
func TxHandlerSignTx(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReqSignTx
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddr, ok := clientUtils.ExtractReqSender(w, req.BaseReq)
		if !ok {
			return
		}

		keyRole, err := tss.KeyRoleFromSimpleStr(req.KeyRole)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := types.NewSignTxRequest(fromAddr, keyRole)
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

// TxHandlerRegisterExternalKey returns the handler to register an external key
func TxHandlerRegisterExternalKey(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReqRegisterExternalKey
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddr, ok := clientUtils.ExtractReqSender(w, req.BaseReq)
		if !ok {
			return
		}

		pubKeyBytes, err := hex.DecodeString(req.PubKey)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		pubKey, err := btcec.ParsePubKey(pubKeyBytes, btcec.S256())
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := types.NewRegisterExternalKeyRequest(fromAddr, req.KeyID, pubKey)
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

// TxHandlerSubmitExternalSignature returns the handler to submit a signature from an external key
func TxHandlerSubmitExternalSignature(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReqSubmitExternalSignature
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddr, ok := clientUtils.ExtractReqSender(w, req.BaseReq)
		if !ok {
			return
		}

		signature, err := hex.DecodeString(req.Signature)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		sigHash, err := hex.DecodeString(req.SigHash)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := types.NewSubmitExternalSignatureRequest(fromAddr, req.KeyID, signature, sigHash)
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}
