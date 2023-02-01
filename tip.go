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

func tipBodyForRawTransactionCreate(assetId string, opponentKey string, opponentReceivers []string, opponentThreshold int64, amount number.Decimal, traceId, memo string) []byte {
	body := assetId + opponentKey
	for _, or := range opponentReceivers {
		body = body + or
	}
	body = body + fmt.Sprint(opponentThreshold)
	body = body + amount.Persist()
	body = body + traceId + memo
	return tipBody(TIPRawTransactionCreate + body)
}

func tipBodyForWithdrawalCreate(addressId string, amount, fee number.Decimal, traceId, memo string) []byte {
	body := addressId + amount.Persist() + fee.Persist()
	body = body + traceId + memo
	return tipBody(TIPWithdrawalCreate + body)
}

func tipBodyForTransfer(assetId string, counterUserId string, amount number.Decimal, traceId, memo string) []byte {
	body := assetId + counterUserId + amount.Persist()
	body = body + traceId + memo
	return tipBody(TIPTransferCreate + body)
}

func tipBodyForPhoneNumberUpdate(verificationId, code string) []byte {
	body := verificationId + code
	return tipBody(TIPPhoneNumberUpdate + body)
}

func tipBodyForEmergencyContactCreate(verificationId, code string) []byte {
	body := verificationId + code
	return tipBody(TIPEmergencyContactCreate + body)
}

func tipBodyForAddressAdd(assetId string, publicKey, keyTag, name string) []byte {
	body := assetId + publicKey + keyTag + name
	return tipBody(TIPAddressAdd + body)
}

func tipBodyForProvisioningUpdate(deviceId string, secret string) []byte {
	body := deviceId + secret
	return tipBody(TIPProvisioningUpdate + body)
}

func tipBody(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
