package bot

type Authenticator struct {
	Uid        string
	Sid        string
	PrivateKey string
}

func NewAuthenticator(uid, sid, privateKey string) *Authenticator {
	return &Authenticator{
		Uid:        uid,
		Sid:        sid,
		PrivateKey: privateKey,
	}
}

func (a *Authenticator) BuildJWT(method, uri, body string) (string, error) {
	user := &SafeUser{
		UserId:            uid,
		SessionId:         sid,
		SessionPrivateKey: a.PrivateKey,
	}
	return SignAuthenticationToken(method, uri, body, user)
}
