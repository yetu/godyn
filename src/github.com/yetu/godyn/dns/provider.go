package dns

type Provider interface {
	UpdateARecord(zone, fqdn, ip string, force bool) (result bool, err error)
}
