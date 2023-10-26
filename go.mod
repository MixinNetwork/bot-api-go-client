module github.com/MixinNetwork/bot-api-go-client

go 1.21

toolchain go1.21.0

replace github.com/vmihailenco/msgpack/v4 => github.com/MixinNetwork/msgpack/v4 v4.3.14

require (
	filippo.io/edwards25519 v1.0.0
	github.com/MixinNetwork/go-number v0.1.1
	github.com/MixinNetwork/mixin v0.17.1
	github.com/gofrs/uuid/v5 v5.0.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/gorilla/websocket v1.5.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.4
	github.com/vmihailenco/msgpack/v4 v4.3.13
	golang.org/x/crypto v0.14.0
	gopkg.in/urfave/cli.v1 v1.20.0
	nhooyr.io/websocket v1.8.10
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/vmihailenco/tagparser v0.1.2 // indirect
	github.com/zeebo/blake3 v0.2.3 // indirect
	golang.org/x/sys v0.13.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
