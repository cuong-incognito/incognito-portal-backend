package main

type API_add_portal_shielding_request struct {
	IncAddress string
	BTCAddress string
}

type API_respond struct {
	Result interface{}
	Error  *string
}
