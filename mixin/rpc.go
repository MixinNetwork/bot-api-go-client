package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
)

var httpClient *http.Client

func callRPC(node, method string, params []interface{}) ([]byte, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	body, err := json.Marshal(map[string]interface{}{
		"method": method,
		"params": params,
	})
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", node, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data  interface{} `json:"data"`
		Error interface{} `json:"error"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("ERROR %s", result.Error)
	}

	return json.Marshal(result.Data)
}

type signerInput struct {
	Inputs []struct {
		Hash    crypto.Hash         `json:"hash"`
		Index   int                 `json:"index"`
		Deposit *common.DepositData `json:"deposit,omitempty"`
		Keys    []crypto.Key        `json:"keys"`
		Mask    crypto.Key          `json:"mask"`
	} `json:"inputs"`
	Outputs []struct {
		Type     uint8            `json:"type"`
		Script   common.Script    `json:"script"`
		Accounts []common.Address `json:"accounts,omitempty"`
		Keys     []crypto.Key     `json:"keys,omitempty"`
		Mask     crypto.Key       `json:"mask,omitempty"`
		Amount   common.Integer   `json:"amount"`
	}
	Asset crypto.Hash `json:"asset"`
	Extra string      `json:"extra"`
	Node  string      `json:"-"`
}

func (raw signerInput) ReadUTXO(hash crypto.Hash, index int) (*common.UTXOWithLock, error) {
	utxo := &common.UTXOWithLock{}

	for _, in := range raw.Inputs {
		if in.Hash == hash && in.Index == index && len(in.Keys) > 0 {
			utxo.Keys = in.Keys
			utxo.Mask = in.Mask
			return utxo, nil
		}
	}

	data, err := callRPC(raw.Node, "getutxo", []interface{}{hash.String(), index})
	if err != nil {
		return nil, err
	}
	var out common.UTXOWithLock
	err = json.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}
	if out.Amount.Sign() == 0 {
		return nil, fmt.Errorf("invalid input %s#%d", hash.String(), index)
	}
	utxo.Keys = out.Keys
	utxo.Mask = out.Mask
	return utxo, nil
}

func (raw signerInput) CheckDepositInput(deposit *common.DepositData, tx crypto.Hash) error {
	return nil
}

func (raw signerInput) ReadLastMintDistribution(group string) (*common.MintDistribution, error) {
	return nil, nil
}

func SignTransaction(account common.Address, raw signerInput) (*common.SignedTransaction, error) {
	tx := common.NewTransaction(raw.Asset)
	for _, in := range raw.Inputs {
		if in.Deposit != nil {
			tx.AddDepositInput(in.Deposit)
		} else {
			tx.AddInput(in.Hash, in.Index)
		}
	}

	for _, out := range raw.Outputs {
		if out.Type != common.OutputTypeScript {
			return nil, fmt.Errorf("invalid output type %d", out.Type)
		}

		if out.Accounts != nil {
			tx.AddRandomScriptOutput(out.Accounts, out.Script, out.Amount)
		}
		if out.Keys != nil {
			tx.Outputs = append(tx.Outputs, &common.Output{
				Type:   common.OutputTypeScript,
				Amount: out.Amount,
				Keys:   out.Keys,
				Script: common.NewThresholdScript(1),
				Mask:   out.Mask,
			})
		}
	}

	extra, err := hex.DecodeString(raw.Extra)
	if err != nil {
		return nil, err
	}
	tx.Extra = extra

	signed := &common.SignedTransaction{Transaction: *tx}
	for i, _ := range signed.Inputs {
		err := signed.SignInput(raw, i, []common.Address{account})
		if err != nil {
			return nil, err
		}
	}
	return signed, nil
}

func SignTransactionRaw(node string, account common.Address, rawStr string) (*common.SignedTransaction, error) {
	var raw signerInput
	err := json.Unmarshal([]byte(rawStr), &raw)
	if err != nil {
		return nil, err
	}
	raw.Node = node
	return SignTransaction(account, raw)
}

func SentTransaction(node string, raw string) error {
	content, err := json.Marshal(map[string]interface{}{"method": "sendrawtransaction", "params": []interface{}{raw}})
	if err != nil {
		return err
	}
	log.Println(raw)
	data, err := callRPC(node, "POST", bytes.NewReader(content))
	log.Println(string(data), err)
	if err != nil {
		return err
	}
	var resp struct {
		Data struct {
			Hash string `json:"hash"`
		} `json:"data"`
		Error *string `json:"error,omitempty"`
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return errors.New(*resp.Error)
	}
	log.Println(resp.Data.Hash)
	return nil
}
