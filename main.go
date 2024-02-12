package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/utils"
)

func main() {

	config, err := readConfig("./config.yaml")
	if err != nil {
		log.Fatal("unable to parse config file %v", err)
		os.Exit(1)
	}
	godotenv.Load(".envrc")
	/*
		opts := golangsdk.AuthOptions{
			IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
			Username:         os.Getenv("OS_USERNAME"),
			Password:         os.Getenv("OS_PASSWORD"),
			DomainName:       os.Getenv("OS_DOMAIN_NAME"),
			TenantName:       os.Getenv("OS_SUBPROJECT_NAME"),
		}
	*/
	opts := golangsdk.AKSKAuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		ProjectName:      os.Getenv("OS_SUBPROJECT_NAME"),
		AccessKey:        os.Getenv("OS_AK"),
		SecretKey:        os.Getenv("OS_SK"),
	}
	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		log.Fatalf("%v", err)
	} else {
		log.Printf("Building openstack.AuthenticatedClient was success.")
	}

	elbClient, err := createELBClient(provider, golangsdk.EndpointOpts{
		Region: utils.GetRegionFromAKSK(opts),
	})
	if err != nil {
		log.Fatalf("%v", err)
	} else {
		log.Printf("Building ELBCLient was success")
	}

	wafClient, err := createWAFClient(provider, golangsdk.EndpointOpts{
		Region: utils.GetRegionFromAKSK(opts),
	})
	if err != nil {
		log.Fatalf("%v", err)
	} else {
		log.Printf("Building WAFClient was success")
	}

	var force bool
	flag.BoolVar(&force, "force", false, "Use the force")
	flag.Parse()

	clientset, err := createClientset("/home/a96247585/.kube/config")
	if err != nil {
		log.Fatalf("%v", err)
	} else {
		log.Printf("Kubernetes clientset built")
	}

	for _, domainConfig := range config.Domains {

		log.Printf("[%v] Starting reconcile procedure", domainConfig.Domain)
		log.Printf("[%v] Configuration: secretName: %v namespace: %v ELB: %v wafID: %v",
			domainConfig.Domain,
			domainConfig.SecretName,
			domainConfig.Namespace,
			domainConfig.ELB,
			domainConfig.WafID,
		)

		myKey, myCert, err := getCertFromKubernetes(clientset, domainConfig.SecretName, domainConfig.Namespace)
		if err != nil {
			log.Fatalf("[%v] %v", domainConfig.Domain, err)
		} else {
			log.Printf("[%v] loaded secret '%v' from '%v' namespace", domainConfig.Domain, domainConfig.SecretName, domainConfig.Namespace)
		}
		certName := fmt.Sprintf("managed_by_cce_cm_reconciler-upt-time-%d", time.Now().Unix())
		expireTime, err := calculateCertExpireTime(myCert)
		if err != nil {
			log.Fatalf("[%v] %v", domainConfig.Domain, err)
		} else {
			log.Printf("[%v] certificate expire time in kubernetes secret: %v", domainConfig.Domain, expireTime)
		}
		ci := certData{
			Domain:     domainConfig.Domain,
			Key:        myKey,
			Cert:       myCert,
			Name:       certName,
			ExpireTime: expireTime,
		}
		// its for testing purpose only
		if force {
			ci.ExpireTime = staticNotAfterInSecret
			log.Printf("[%v] FORCE MODE IS ON CERT WILL BE UPDATED IN THE PROVIDER", domainConfig.Domain)
		}
		if domainConfig.ELB {
			elbCertInfo, err := getELBCertID(elbClient, ci.Domain)
			if err != nil {
				log.Fatalf("[%v] %v", domainConfig.Domain, err)
			}
			log.Printf("[%v] ELB certificate ID: %v \n", ci.Domain, elbCertInfo.ID)
			log.Printf("[%v] ELB certificate expire time %v \n", ci.Domain, elbCertInfo.ExpireTime)

			if isReconcileTime(ci.ExpireTime, elbCertInfo.ExpireTime) {
				log.Printf("[%v] certificate's notAfter in kubernetes secret is later than in provider, attempting to reconcile certificate for ELB", ci.Domain)
				err = reconcileELBCert(elbClient, elbCertInfo.ID, ci)
				if err != nil {
					log.Fatalf("[%v] error updating ELB certificate", domainConfig.Domain)
					log.Fatalf("%v", err)
				} else {
					log.Printf("[%v] ELB certificate successfully updated", ci.Domain)
				}
			} else {
				log.Printf("[%v] provider for ELB has %v expire time, Kubernetes secret has %v no reconcile needed", domainConfig.Domain, elbCertInfo.ExpireTime, ci.ExpireTime)
			}
		}

		if domainConfig.Domain != "<nil>" {
			wafNotAfter, err := getWAFCertNotAfter(wafClient, domainConfig.WafID)
			if err != nil {
				log.Fatalf("[%v] %v", domainConfig.Domain, err)
			} else {
				log.Printf("[%v] WAF certificate expire time %v", domainConfig.Domain, wafNotAfter)
			}

			if isReconcileTime(ci.ExpireTime, wafNotAfter) {
				log.Printf("[%v] certificate's notAfter in kubernetes secret is newer than in provider, should reconcile certificate for WAF %v", domainConfig.Domain, domainConfig.WafID)
				wafCertId, err := createWAFCert(wafClient, ci)
				if err != nil {
					log.Fatalf("[%v] %v", domainConfig.Domain, err)
					os.Exit(1)
				} else {
					log.Printf("[%v] created new WAF certificate with ID: %v", ci.Domain, wafCertId)
				}
				err = updateWAFCert(wafClient, domainConfig.WafID, wafCertId)
				if err != nil {
					log.Fatalf("[%v] %v", domainConfig.Domain, err)
					os.Exit(1)
				}
				log.Printf("[%v] WAF certificate has been updated", domainConfig.Domain)
			} else {
				log.Printf("[%v] provider for WAF %v has %v expire time, Kubernetes secret has %v no reconcile needed", domainConfig.Domain, domainConfig.Domain, wafNotAfter, ci.ExpireTime)
			}
		}
	}
}
