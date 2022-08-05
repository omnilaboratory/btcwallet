package legacyrpc

import (
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcwallet/wallet"
	"io/ioutil"
	"net/http"
)

func gotoRemote(interface{}, *wallet.Wallet) (interface{}, error){
	return nil,nil
}
var pubRpcHandlers = map[string]struct {
	handler          requestHandler
	handlerWithChain requestHandlerChainRequired

	// Function variables cannot be compared against anything but nil, so
	// use a boolean to record whether help generation is necessary.  This
	// is used by the tests to ensure that help can be generated for every
	// implemented method.
	//
	// A single map and this bool is here is used rather than several maps
	// for the unimplemented handlers so every method has exactly one
	// handler function.
	noHelp bool
}{
	// Reference implementation wallet methods (implemented)
	"addmultisigaddress":     {handler: unsupported},
	"createmultisig":         {handler: unsupported},
	"dumpprivkey":            {handler: unsupported},
	"getaccount":             {handler: unsupported},
	"getaccountaddress":      {handler: unsupported},
	"getaddressesbyaccount":  {handler: unsupported},
	"getbalance":             {handler: unsupported},
	"getbestblockhash":       {handler: gotoRemote},
	"getblockcount":          {handler: gotoRemote},
	//"getinfo":                {handlerWithChain: getInfo},
	"getinfo":                {handler: gotoRemote},
	"getnewaddress":          {handler: unsupported},
	"getrawchangeaddress":    {handler: unsupported},
	"getreceivedbyaccount":   {handler: unsupported},
	"getreceivedbyaddress":   {handler: unsupported},
	"gettransaction":         {handler: gotoRemote},
	"help":                   {handler: gotoRemote},
	"importprivkey":          {handler: unsupported},
	"keypoolrefill":          {handler: unsupported},
	"listaccounts":           {handler: unsupported},
	"listlockunspent":        {handler: unsupported},
	"listreceivedbyaccount":  {handler: unsupported},
	"listreceivedbyaddress":  {handler: unsupported},
	"listsinceblock":         {handler: unsupported},
	"listtransactions":       {handler: unsupported},
	"listunspent":            {handler: unsupported},
	"lockunspent":            {handler: unsupported},
	"sendfrom":               {handler: unsupported},
	"sendmany":               {handler: unsupported},
	"sendtoaddress":          {handler: unsupported},
	"settxfee":               {handler: unsupported},
	"signmessage":            {handler: unsupported},
	"signrawtransaction":     {handler: unsupported},
	"validateaddress":        {handler: unsupported},
	"verifymessage":          {handler: unsupported},
	"walletlock":             {handler: unsupported},
	"walletpassphrase":       {handler: unsupported},
	"walletpassphrasechange": {handler: unsupported},


	// Reference implementation methods (still unimplemented)
	"backupwallet":         {handler: unimplemented, noHelp: true},
	"dumpwallet":           {handler: unimplemented, noHelp: true},
	"getwalletinfo":        {handler: unimplemented, noHelp: true},
	"importwallet":         {handler: unimplemented, noHelp: true},
	"listaddressgroupings": {handler: unimplemented, noHelp: true},
	// Reference methods which can't be implemented by btcwallet due to
	// design decision differences
	"encryptwallet": {handler: unsupported, noHelp: true},
	"move":          {handler: unsupported, noHelp: true},
	"setaccount":    {handler: unsupported, noHelp: true},




	// Extensions to the reference client JSON-RPC API
	"createnewaccount": {handler: unsupported},
	"getbestblock":     {handler: gotoRemote},
	// This was an extension but the reference implementation added it as
	// well, but with a different API (no account parameter).  It's listed
	// here because it hasn't been update to use the reference
	// implemenation's API.
	"getunconfirmedbalance":   {handler: unsupported},
	"listaddresstransactions": {handler: unsupported},
	"listalltransactions":     {handler: unsupported},
	"renameaccount":           {handler: unsupported},
	"walletislocked":          {handler: unsupported},
}


/*bod update wxf
 */
//modify from  func lazyApplyHandler
//only export chain public api, not include any wallet function
func lazyApplyHandlerOnlyPub(request *btcjson.Request, chainClient *rpcclient.Client ) lazyHandler {
	handlerData, ok := pubRpcHandlers[request.Method]
	//var  w *wallet.Wallet
	if ok && handlerData.handler != nil && fmt.Sprintf("%v",handlerData.handler)!=fmt.Sprintf("%v",gotoRemote) {
		return func() (interface{}, *btcjson.RPCError) {
			cmd, err := btcjson.UnmarshalCmd(request)
			if err != nil {
				return nil, btcjson.ErrRPCInvalidRequest
			}
			resp, err := handlerData.handler(cmd, nil)
			if err != nil {
				return nil, jsonError(err)
			}
			return resp, nil
		}
	}

	// Fallback to RPC passthrough
	return func() (interface{}, *btcjson.RPCError) {
		if chainClient == nil {
			return nil, &btcjson.RPCError{
				Code:    -1,
				Message: "Chain RPC is inactive",
			}
		}
		resp, err := chainClient.RawRequest(request.Method,
			request.Params)
		if err != nil {
			return nil, jsonError(err)
		}
		return &resp, nil
	}
}

func PublicHttpHandler(w http.ResponseWriter, r *http.Request,chainClient *rpcclient.Client) {
	body := http.MaxBytesReader(w, r.Body, maxRequestSize)
	rpcRequest, err := ioutil.ReadAll(body)
	if err != nil {
		// TODO: what if the underlying reader errored?
		http.Error(w, "413 Request Too Large.",
			http.StatusRequestEntityTooLarge)
		return
	}

	// First check whether wallet has a handler for this request's method.
	// If unfound, the request is sent to the chain server for further
	// processing.  While checking the methods, disallow authenticate
	// requests, as they are invalid for HTTP POST clients.
	var req btcjson.Request
	err = json.Unmarshal(rpcRequest, &req)
	if err != nil {
		resp, err := btcjson.MarshalResponse(
			btcjson.RpcVersion1, req.ID, nil,
			btcjson.ErrRPCInvalidRequest,
		)
		if err != nil {
			log.Errorf("Unable to marshal response: %v", err)
			http.Error(w, "500 Internal Server Error",
				http.StatusInternalServerError)
			return
		}
		_, err = w.Write(resp)
		if err != nil {
			log.Warnf("Cannot write invalid request request to "+
				"client: %v", err)
		}
		return
	}
	// Create the response and error from the request.  Two special cases
	// are handled for the authenticate and stop request methods.
	var res interface{}
	var jsonErr *btcjson.RPCError
	res, jsonErr =lazyApplyHandlerOnlyPub(&req,chainClient)()
	// Marshal and send.
	mresp, err := btcjson.MarshalResponse(
		btcjson.RpcVersion1, req.ID, res, jsonErr,
	)
	if err != nil {
		log.Errorf("Unable to marshal response: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	_, err = w.Write(mresp)
	if err != nil {
		log.Warnf("Unable to respond to client: %v", err)
	}
}