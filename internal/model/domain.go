package model

import "time"

// DomainInfo 表示域名查询结果
type DomainInfo struct {
	DomainName       string     `json:"domain_name"`
	ROID             string     `json:"roid,omitempty"`
	QueryProtocol    string     `json:"query_protocol"`
	QueryTime        time.Time  `json:"query_time"`
	QueryDuration    int64      `json:"query_duration_ms"`
	DataSource       string     `json:"data_source"`
	RegistrarName    string     `json:"registrar_name,omitempty"`
	RegistrarURL     string     `json:"registrar_url,omitempty"`
	RegistrarIANAID  string     `json:"registrar_iana_id,omitempty"`
	RegistrantName   string     `json:"registrant_name,omitempty"`
	RegistrationDate *time.Time `json:"registration_date,omitempty"`
	ExpirationDate   *time.Time `json:"expiration_date,omitempty"`
	LastUpdated      *time.Time `json:"last_updated,omitempty"`
	Status           []string   `json:"status"`
	NameServers      []string   `json:"name_servers"`
	DNSSEC           DNSSECInfo `json:"dnssec"`
	RawResponse      *string    `json:"raw_response,omitempty"`
}

// DNSSECInfo 表示 DNSSEC 信息
type DNSSECInfo struct {
	Signed           *bool `json:"signed,omitempty"`
	DelegationSigned *bool `json:"delegation_signed,omitempty"`
}
