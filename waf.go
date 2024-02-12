package main

import (
	"fmt"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	wafcert "github.com/opentelekomcloud/gophertelekomcloud/openstack/waf/v1/certificates"
	wafdomain "github.com/opentelekomcloud/gophertelekomcloud/openstack/waf/v1/domains"
)

func createWAFClient(provider *golangsdk.ProviderClient, opts golangsdk.EndpointOpts) (*golangsdk.ServiceClient, error) {
	wafClient, err := openstack.NewWAFV1(provider, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create golangsdk WAF serviceClient %v", err)
	}
	return wafClient, nil
}

func createWAFCert(wafClient *golangsdk.ServiceClient, c certData) (string, error) {
	uo := wafcert.CreateOpts{
		Name:    c.Name,
		Content: c.Cert,
		Key:     c.Key,
	}
	r, err := wafcert.Create(wafClient, uo).Extract()
	if err != nil {
		return "", err
	}
	return r.Id, nil
}

func updateWAFCert(wafClient *golangsdk.ServiceClient, wafId string, certId string) error {
	uo := wafdomain.UpdateOpts{
		CertificateId: certId,
	}
	_, err := wafdomain.Update(wafClient, wafId, uo).Extract()
	if err != nil {
		return err
	}
	return nil
}

// TODO delete old waf cert

func getWAFCertNotAfter(wafClient *golangsdk.ServiceClient, wafId string) (string, error) {
	r, err := wafdomain.Get(wafClient, wafId).Extract()
	if err != nil {
		return "", fmt.Errorf("failed to get WAF instance %v %v", wafId, err)
	}
	re, err := wafcert.Get(wafClient, r.CertificateId).Extract()
	if err != nil {
		return "", fmt.Errorf("failed to get WAF certificate for WAF: %v with CertificateId: %v %v", wafId, r.CertificateId, err)
	}
	return unixToRFC3339(int64(re.ExpireTime)), nil
}
