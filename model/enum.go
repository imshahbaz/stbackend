package model

type OTPType string

const (
	OTPRegister OTPType = "REGISTER"
	OTPUpdate   OTPType = "UPDATE"
)

type UserCacheType string

const (
	Truecaller UserCacheType = "TRUECALLER"
	Signup     UserCacheType = "SIGNUP"
	CredUpdate UserCacheType = "CRED_UPDATE"
)
