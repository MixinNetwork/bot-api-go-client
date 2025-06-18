package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	InscriptionModeInstant = 1
	InscriptionModeDone    = 2
)

type InscriptionDeploy struct {
	// the version must be 1
	Version uint8 `json:"version"`

	// 1 distribute tokens per inscription
	// 2 distribute tokens after inscription progress done
	Mode uint8 `json:"mode"`

	// supply is the total supply of all tokens
	// unit is the amount of tokens per inscription
	//
	// if supply is 1,000,000,000 and unit is 1,000,000,
	// then there should be 1,000 inscription operations
	// 1 inscription represents 1 collectible, so there will be 1,000 NFTs
	Unit   string `json:"unit"`
	Supply string `json:"supply"`

	// the token symbol and name are required and must be valid UTF8
	Symbol string `json:"symbol"`
	Name   string `json:"name"`

	// the icon must be in valid data URI scheme
	// e.g. image/webp;base64,IVVB===
	Icon string `json:"icon"`

	// only needed if the deployer wants to limit the NFT contents of all
	// inscriptions, base64 of all NFT blake3 checksums, and all checksums
	// must be different from each other
	Checksum string `json:"checksum,omitempty"`

	// ratio of each inscribed tokens will be kept in treasury
	// the treasury tokens will be distributed to the recipient MIX address
	// at the same time  as defined by the mode
	//
	// For MAO, the ratio will be 0.9, and each collectible will only cost
	// 10% of the unit tokens, so only the inscribers have NFTs, but not
	// the treasury tokens, however they can occupy a vacant NFT.
	Treasury *struct {
		Ratio     string `json:"ratio"`
		Recipient string `json:"recipient"`
	} `json:"treasury,omitempty"`
}

type InscriptionInscribe struct {
	// operation must be inscribe
	Operation string `json:"operation"`

	// Recipient can only be MIX address, not ghost keys, because
	// otherwise the keys may be used by others, then redeemed invalid
	Recipient string `json:"recipient"`

	// data URI scheme
	// application/octet-stream;key=fingerprint;base64,iVBO==
	// image/webp;trait=one;base64,iVBO==
	// text/plain;charset=UTF-8,cedric.mao
	// text/plain;charset=UTF-8;base64,iii==
	Content string `json:"content,omitempty"`
}

type InscriptionDistribute struct {
	// operation must be distribute
	Operation string `json:"distribute"`

	// sequence must monoticially increase from 0
	Sequence uint64 `json:"sequence"`
}

type InscriptionOccupy struct {
	// operation must be occupy
	Operation string `json:"operation"`

	// the integer sequence number of the NFT inscription
	Sequence uint64 `json:"sequence"`
}

// Collection represents the API response structure for collection data
type Collection struct {
	AssetKey       string                 `json:"asset_key"`
	CollectionHash string                 `json:"collection_hash"`
	KernelAssetID  string                 `json:"kernel_asset_id"`
	Name           string                 `json:"name"`
	Symbol         string                 `json:"symbol"`
	Description    string                 `json:"description"`
	IconURL        string                 `json:"icon_url"`
	Supply         string                 `json:"supply"`
	Unit           string                 `json:"unit"`
	MinimumPrice   string                 `json:"minimum_price"`
	Treasury       map[string]interface{} `json:"treasury"`
	Type           string                 `json:"type"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// Inscription represents the API response structure for inscription data
type Inscription struct {
	InscriptionHash string    `json:"inscription_hash"`
	CollectionHash  string    `json:"collection_hash"`
	ContentType     string    `json:"content_type"`
	ContentURL      string    `json:"content_url"`
	CreatedAt       time.Time `json:"created_at"`
	OccupiedBy      string    `json:"occupied_by"`
	Owner           string    `json:"owner"`
	Recipient       string    `json:"recipient"`
	Sequence        int64     `json:"sequence"`
	Type            string    `json:"type"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ReadCollection reads collection information from Mixin API
func ReadCollection(ctx context.Context, collectionHash string) (*Collection, error) {
	body, err := Request(ctx, "GET", "/safe/inscriptions/collections/"+collectionHash, nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *Collection `json:"data"`
		Error Error       `json:"error"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error.Code > 0 {
		return nil, resp.Error
	}

	return resp.Data, nil
}

// ReadInscription reads inscription information from Mixin API
func ReadInscription(ctx context.Context, inscriptionHash string) (*Inscription, error) {
	body, err := Request(ctx, "GET", "/safe/inscriptions/items/"+inscriptionHash, nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *Inscription `json:"data"`
		Error Error        `json:"error"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error.Code > 0 {
		return nil, resp.Error
	}

	return resp.Data, nil
}

// ReadCollectionItems reads all items in a collection from Mixin API
func ReadCollectionItems(ctx context.Context, collectionHash string) ([]*Inscription, error) {
	path := fmt.Sprintf("/safe/inscriptions/collections/%s/items", collectionHash)
	body, err := Request(ctx, "GET", path, nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  []*Inscription `json:"data"`
		Error Error          `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}

	return resp.Data, nil
}
