package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/MixinNetwork/go-number"
)

const (
	TIPVerify                   = "TIP:VERIFY:"
	TIPAddressAdd               = "TIP:ADDRESS:ADD:"
	TIPAddressRemove            = "TIP:ADDRESS:REMOVE:"
	TIPUserDeactivate           = "TIP:USER:DEACTIVATE:"
	TIPEmergencyContactCreate   = "TIP:EMERGENCY:CONTACT:CREATE:"
	TIPEmergencyContactRead     = "TIP:EMERGENCY:CONTACT:READ:"
	TIPEmergencyContactRemove   = "TIP:EMERGENCY:CONTACT:REMOVE:"
	TIPPhoneNumberUpdate        = "TIP:PHONE:NUMBER:UPDATE:"
	TIPMultisigRequestSign      = "TIP:MULTISIG:REQUEST:SIGN:"
	TIPMultisigRequestUnlock    = "TIP:MULTISIG:REQUEST:UNLOCK:"
	TIPCollectibleRequestSign   = "TIP:COLLECTIBLE:REQUEST:SIGN:"
	TIPCollectibleRequestUnlock = "TIP:COLLECTIBLE:REQUEST:UNLOCK:"
	TIPTransferCreate           = "TIP:TRANSFER:CREATE:"
	TIPWithdrawalCreate         = "TIP:WITHDRAWAL:CREATE:"
	TIPRawTransactionCreate     = "TIP:TRANSACTION:CREATE:"
	TIPOAuthApprove             = "TIP:OAUTH:APPROVE:"
	TIPProvisioningUpdate       = "TIP:PROVISIONING:UPDATE:"
	TIPOwnershipTransfer        = "TIP:APP:OWNERSHIP:TRANSFER:"
	TIPSequencerRegister        = "SEQUENCER:REGISTER:"
)

type TipNodeData struct {
	Commitments []string `json:"commitments"`
	Identity    string   `json:"identity"`
}

func GetTipNodeByPathWithRequestId(ctx context.Context, path, requestId string) (*TipNodeData, error) {
	url := fmt.Sprintf("/external/tip/%s", path)
	body, err := RequestWithId(ctx, "GET", url, nil, "", requestId)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *TipNodeData `json:"data"`
		Error Error        `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func GetTipNodeByPath(ctx context.Context, path string) (*TipNodeData, error) {
	return GetTipNodeByPathWithRequestId(ctx, path, UuidNewV4().String())
}

func TIPMigrateBody(pub ed25519.PublicKey) string {
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, 1)
	pub = append(pub, counter...)
	return hex.EncodeToString(pub)
}

func TIPBodyForVerify(timestamp int64) []byte {
	return []byte(fmt.Sprintf("%s%032d", TIPVerify, timestamp))
}

func TipBodyForRawTransactionCreate(assetId string, opponentKey string, opponentReceivers []string, opponentThreshold int64, amount number.Decimal, traceId, memo string) []byte {
	body := assetId + opponentKey
	for _, or := range opponentReceivers {
		body = body + or
	}
	body = body + fmt.Sprint(opponentThreshold)
	body = body + amount.Persist()
	body = body + traceId + memo
	return TipBody(TIPRawTransactionCreate + body)
}

func TipBodyForWithdrawalCreate(addressId string, amount, fee number.Decimal, traceId, memo string) []byte {
	body := addressId + amount.Persist() + fee.Persist()
	body = body + traceId + memo
	return TipBody(TIPWithdrawalCreate + body)
}

func TipBodyForTransfer(assetId string, counterUserId string, amount number.Decimal, traceId, memo string) []byte {
	body := assetId + counterUserId + amount.Persist()
	body = body + traceId + memo
	return TipBody(TIPTransferCreate + body)
}

func TipBodyForPhoneNumberUpdate(verificationId, code string) []byte {
	body := verificationId + code
	return TipBody(TIPPhoneNumberUpdate + body)
}

func TipBodyForEmergencyContactCreate(verificationId, code string) []byte {
	body := verificationId + code
	return TipBody(TIPEmergencyContactCreate + body)
}

func TipBodyForAddressAdd(assetId string, publicKey, keyTag, name string) []byte {
	body := assetId + publicKey + keyTag + name
	return TipBody(TIPAddressAdd + body)
}

func TipBodyForProvisioningUpdate(deviceId string, secret string) []byte {
	body := deviceId + secret
	return TipBody(TIPProvisioningUpdate + body)
}

func TipBodyForOwnershipTransfer(userId string) []byte {
	return TipBody(TIPOwnershipTransfer + userId)
}

func TIPBodyForSequencerRegister(userId, publicKey string) []byte {
	return TipBody(TIPSequencerRegister + userId + publicKey)
}

func TipBody(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
