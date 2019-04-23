/*
 * Copyright (c) 2019 Manabu Sonoda
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dtap

import (
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/miekg/dns"
)

type DnstapFlatT struct {
	Timestamp       time.Time `json:timestamp`
	QueryTime       time.Time `json:"query_time,omitempty"`
	QueryAddress    string    `json:"query_address,omitempty"`
	QueryPort       uint32    `json:"query_port,omitempty"`
	ResponseTime    time.Time `json:"response_time,omitempty"`
	ResponseAddress string    `json:"response_address,omitempty"`
	ResponsePort    uint32    `json:"response_port,omitempty"`
	ResponseZone    string    `json:"response_zone,omitempty"`
	Identity        string    `json:"identity,omitempty"`
	Type            string    `json:"type"`
	SocketFamily    string    `json:"socket_family"`
	SocketProtocol  string    `json:"socket_protocol"`
	Version         string    `json:"version"`
	Extra           string    `json:"extra"`
	Names           []string  `json:"names"`
	Qname           string    `json:"qname"`
	Qclass          string    `json:"qclass"`
	Qtype           string    `json:"qtype"`
	MessageSize     int       `json:"message_size"`
	Txid            uint16    `json:"txid"`
	Rcode           string    `json:"rcode"`
	AA              bool      `json:"aa"`
	TC              bool      `json:"tc"`
	RD              bool      `json:"rd"`
	RA              bool      `json:"ra"`
	AD              bool      `json:"ad"`
	CD              bool      `json:"cd"`
}

func FlatDnstap(dt *dnstap.Dnstap, ipv4Mask net.IPMask, ipv6Mask net.IPMask) (map[string]interface{}, error) {
	var names = map[int]string{
		2: "tld",
		3: "2ld",
		4: "3ld",
		5: "4ld",
	}

	var dnsMessage []byte
	var data = make(map[string]interface{})
	msg := dt.GetMessage()
	if msg.GetQueryMessage() != nil {
		dnsMessage = msg.GetQueryMessage()
	} else {
		dnsMessage = msg.GetResponseMessage()
	}

	data["query_time"] = time.Unix(int64(msg.GetQueryTimeSec()), int64(msg.GetQueryTimeNsec())).Format(time.RFC3339Nano)
	data["response_time"] = time.Unix(int64(msg.GetResponseTimeSec()), int64(msg.GetResponseTimeNsec())).Format(time.RFC3339Nano)
	if len(msg.GetQueryAddress()) == 4 {
		data["query_address"] = net.IP(msg.GetQueryAddress()).Mask(ipv4Mask).String()
	} else {
		data["query_address"] = net.IP(msg.GetQueryAddress()).Mask(ipv6Mask).String()
	}
	data["query_port"] = msg.GetQueryPort()
	if len(msg.GetResponseAddress()) == 4 {
		data["response_address"] = net.IP(msg.GetResponseAddress()).Mask(ipv4Mask).String()
	} else {
		data["response_address"] = net.IP(msg.GetResponseAddress()).Mask(ipv6Mask).String()
	}

	data["response_port"] = msg.GetResponsePort()
	data["response_zone"] = msg.GetQueryZone()
	data["identity"] = dt.GetIdentity()
	if data["identity"] == nil {
		data["identity"] = hostname
	} else {
		if identity, ok := data["identity"].([]byte); ok {
			if string(identity) == "" {
				data["identity"] = hostname
			}
		}
	}
	data["type"] = msg.GetType().String()
	data["socket_family"] = msg.GetSocketFamily().String()
	data["socket_protocol"] = msg.GetSocketProtocol().String()
	data["version"] = dt.GetVersion()
	data["extra"] = dt.GetExtra()
	dnsMsg := dns.Msg{}
	if err := dnsMsg.Unpack(dnsMessage); err != nil {
		return nil, errors.Wrapf(err, "can't parse dns message() failed: %s\n", err)
	}

	if len(dnsMsg.Question) > 0 {
		data["qname"] = dnsMsg.Question[0].Name
		data["qclass"] = dns.ClassToString[dnsMsg.Question[0].Qclass]
		data["qtype"] = dns.TypeToString[dnsMsg.Question[0].Qtype]
		labels := strings.Split(dnsMsg.Question[0].Name, ".")
		labelsLen := len(labels)
		for i, n := range names {
			if labelsLen-i >= 0 {
				data[n] = strings.Join(labels[labelsLen-i:labelsLen-1], ".")
			} else {
				data[n] = dnsMsg.Question[0].Name
			}
		}
		data["message_size"] = len(dnsMessage)
		data["txid"] = dnsMsg.MsgHdr.Id
	}
	data["rcode"] = dns.RcodeToString[dnsMsg.Rcode]
	data["aa"] = dnsMsg.Authoritative
	data["tc"] = dnsMsg.Truncated
	data["rd"] = dnsMsg.RecursionDesired
	data["ra"] = dnsMsg.RecursionAvailable
	data["ad"] = dnsMsg.AuthenticatedData
	data["cd"] = dnsMsg.CheckingDisabled

	switch msg.GetType() {
	case dnstap.Message_AUTH_QUERY, dnstap.Message_RESOLVER_QUERY,
		dnstap.Message_CLIENT_QUERY, dnstap.Message_FORWARDER_QUERY,
		dnstap.Message_STUB_QUERY, dnstap.Message_TOOL_QUERY:
		data["@timestamp"] = data["query_time"]
	case dnstap.Message_AUTH_RESPONSE, dnstap.Message_RESOLVER_RESPONSE,
		dnstap.Message_CLIENT_RESPONSE, dnstap.Message_FORWARDER_RESPONSE,
		dnstap.Message_STUB_RESPONSE, dnstap.Message_TOOL_RESPONSE:
		data["@timestamp"] = data["response_time"]
	}

	return data, nil
}
