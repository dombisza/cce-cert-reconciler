package main

import (
	"fmt"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	elbcert "github.com/opentelekomcloud/gophertelekomcloud/openstack/elb/v3/certificates"

	
)

func createELBClient(provider *golangsdk.ProviderClient, opts golangsdk.EndpointOpts) (*golangsdk.ServiceClient, error) {
	elbClient, err := openstack.NewELBV3(provider, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create golangsdk ELB serviceClient %v", err)
	}
	return elbClient, nil
}

// get certinfo for a choosen domain

func getELBCertID(elbClient *golangsdk.ServiceClient, domain string) (ELBCertificateInfo, error) {
	certOpts := elbcert.ListOpts{
		Domain: []string{domain},
	}

	pager, err := elbcert.List(elbClient, certOpts).AllPages()
	if err != nil {
		return ELBCertificateInfo{}, fmt.Errorf("failed to list certificates for domain %v %v", domain, err)
	}
	allCerts, err := elbcert.ExtractCertificates(pager)
	if err != nil {
		return ELBCertificateInfo{}, fmt.Errorf("failed to extract certificate for domain %v %v", domain, err)
	}
	if len(allCerts) == 0 {
		return ELBCertificateInfo{}, fmt.Errorf("no certificates found for domain %v", domain)
	}

	// we expect only one cert for a domain
	certInfo := ELBCertificateInfo{
		ID:         allCerts[0].ID,
		ExpireTime: allCerts[0].ExpireTime,
	}
	return certInfo, nil
}

func reconcileELBCert(elbClient *golangsdk.ServiceClient, id string, c certData) error {
	desc := "This certificate is managed by cce-cm-reconciler"
	uo := elbcert.UpdateOpts{
		Name:        c.Name,
		Description: &desc,
		Domain:      c.Domain,
		PrivateKey:  c.Key,
		Certificate: c.Cert,
	}
	_, err := elbcert.Update(elbClient, id, uo).Extract()
	if err != nil {
		return fmt.Errorf("failed to update certificate for domain %v %v", c.Domain, err)
	}
	return nil
}


