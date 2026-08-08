package crypto

func (kp KeyPair) Address() string {
	return kp.PublicKeyHex()
}

func AddressFromPublicKeyHex(publicKeyHex string) string {
	return publicKeyHex
}
