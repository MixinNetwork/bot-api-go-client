## Installation

```
go get github.com/MixinNetwork/bot-api-go-client/v3
```

## User-Agent

Set a User-Agent that identifies your application during startup, before making
requests:

```go
import bot "github.com/MixinNetwork/bot-api-go-client/v3"

bot.SetUserAgent("my-app/1.0 (+https://example.com)")
```

If the User-Agent is empty or still uses the SDK default, the SDK logs a warning
once when it sends an HTTP request. `SetUserAgent` configures the whole process;
call it before starting concurrent requests.

For more info, see [api](https://developers.mixin.one/docs/api/guide).
