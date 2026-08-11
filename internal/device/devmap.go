package device

type Fingerprint struct {
	Reg   uint16
	Value byte
}

type DevMap struct {
	Device      string
	Vendor      string
	Addresses   []uint16
	Fingerprint []Fingerprint
}
