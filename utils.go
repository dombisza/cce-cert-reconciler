package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type ELBCertificateInfo struct {
	ID         string
	ExpireTime string
}

type certData struct {
	Key        string
	Cert       string
	Domain     string
	Name       string
	ExpireTime string
}

type Config struct {
	Domains []DomainConfig `yaml:"domains"`
}

type DomainConfig struct {
	Domain     string `yaml:"domain"`
	SecretName string `yaml:"secretName"`
	Namespace  string `yaml:"namespace"`
	ELB        bool   `yaml:"elb,omitempty"`
	WafID      string `yaml:"wafId,omitempty"`
}

// TODO add error handling and return
func isReconcileTime(notAfterInSecret, notAfterInProvider string) bool {
	secretTime, err := time.Parse(time.RFC3339, notAfterInSecret)
	if err != nil {
		return false
	}
	providerTime, err := time.Parse(time.RFC3339, notAfterInProvider)
	if err != nil {
		return false
	}
	return secretTime.After(providerTime)
}

func unixToRFC3339(unixTime int64) string {
	t := time.Unix(0, unixTime*int64(time.Millisecond))
	rfc3339Time := t.UTC().Format("2006-01-02T15:04:05Z")

	return rfc3339Time
}

// thank you waf devs
func removeTrailingLineBreak(input string) string {
	return strings.TrimRight(input, "\n\r\t ")
}

func readConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func calculateCertExpireTime(certString string) (string, error) {
	block, _ := pem.Decode([]byte(certString))
	if block == nil {
		return "", fmt.Errorf("failed to pem.Decode certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to x509.ParseCertificate %v", err)
	}

	return cert.NotAfter.UTC().Format(time.RFC3339), nil
}
