package bot

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
