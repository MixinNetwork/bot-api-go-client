package main

type SafeUser struct {
	UserID            string `json:"user_id"`
	AppID             string `json:"app_id"`
	SessionID         string `json:"session_id"`
	ServerPublicKey   string `json:"server_public_key"`
	SessionPrivateKey string `json:"session_private_key"`
}
