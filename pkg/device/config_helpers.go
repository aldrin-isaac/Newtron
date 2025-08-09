// File: pkg/device/config_helpers.go
// Chat Session: newtron-20250706-01
// Session Timestamp: 2025-07-06T15:24:00-07:00
package device

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openconfig/ygot/ygot"
	"newtron/pkg/intent"
	"newtron/pkg/oc"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// applyConfiglet applies a pre-defined, internal OpenConfig JSON payload from the configlets directory.
func (d *Device) applyConfiglet(name string) error {
	fmt.Printf("\n>> [Backend API] Applying internal configlet: %s\n", name)

	payloadFile := fmt.Sprintf("pkg/device/configlets/%s.json", name)

	payload, err := ioutil.ReadFile(payloadFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("configlet file not found: %s", payloadFile)
		}
		return fmt.Errorf("could not read internal payload file %s: %w", payloadFile, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var data oc.Device // Use the generated root struct
	if err := oc.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("failed to unmarshal configlet payload into ygot struct: %w", err)
	}

	return d.gnmiSet(ctx, "/", gpb.UpdateResult_REPLACE, data)
}

// buildAclSet constructs a complete ygot struct for an entire ACL (AclSet).
func buildAclSet(name string, ruleList []interface{}, resolvedIntent *intent.ResolvedIntent) (*oc.OpenconfigAcl_Acl_AclSets_AclSet, error) {
	aclSet := &oc.OpenconfigAcl_Acl_AclSets_AclSet{
		Name: ygot.String(name),
		Type: oc.OpenconfigAcl_ACL_TYPE_ACL_IPV4,
		Config: &oc.OpenconfigAcl_Acl_AclSets_AclSet_Config{
			Name: ygot.String(name),
			Type: oc.OpenconfigAcl_ACL_TYPE_ACL_IPV4,
		},
		AclEntries: &oc.OpenconfigAcl_Acl_AclSets_AclSet_AclEntries{
			AclEntry: make(map[uint32]*oc.OpenconfigAcl_Acl_AclSets_AclSet_AclEntries_AclEntry),
		},
	}

	for _, ruleNameIntf := range ruleList {
		ruleName := ruleNameIntf.(string)
		rule, ok := resolvedIntent.PolicyRules[ruleName]
		if !ok {
			return nil, fmt.Errorf("policy rule definition '%s' not found in intent", ruleName)
		}

		aclEntry, err := buildAclEntry(&rule, resolvedIntent)
		if err != nil {
			return nil, fmt.Errorf("failed to build ACL entry for rule '%s': %w", ruleName, err)
		}
		aclSet.AclEntries.AclEntry[uint32(rule.Sequence)] = aclEntry
	}

	return aclSet, nil
}

// buildAclEntry constructs a single ygot AclEntry struct from an intent PolicyRule.
func buildAclEntry(rule *intent.PolicyRule, resolvedIntent *intent.ResolvedIntent) (*oc.OpenconfigAcl_Acl_AclSets_AclSet_AclEntries_AclEntry, error) {
	entry := &oc.OpenconfigAcl_Acl_AclSets_AclSet_AclEntries_AclEntry{
		SequenceId: ygot.Uint32(uint32(rule.Sequence)),
		Ipv4:       &oc.OpenconfigAcl_Acl_AclSets_AclSet_AclEntries_AclEntry_Ipv4{},
		Actions: &oc.OpenconfigAcl_Acl_AclSets_AclSet_AclEntries_AclEntry_Actions{
			Config: &oc.OpenconfigAcl_Acl_AclSets_AclSet_AclEntries_AclEntry_Actions_Config{},
		},
	}

	ipv4Conf := &oc.OpenconfigAcl_Acl_AclSets_AclSet_AclEntries_AclEntry_Ipv4_Config{}
	if rule.Match != nil {
		if proto, ok := rule.Match["protocol"].(string); ok {
			switch strings.ToUpper(proto) {
			case "TCP":
				ipv4Conf.Protocol = oc.OpenconfigPacketMatchTypes_IP_PROTOCOL_TCP
			case "UDP":
				ipv4Conf.Protocol = oc.OpenconfigPacketMatchTypes_IP_PROTOCOL_UDP
			case "ICMP":
				ipv4Conf.Protocol = oc.OpenconfigPacketMatchTypes_IP_PROTOCOL_ICMP
			}
		}
		if srcList, ok := rule.Match["source_list"].([]interface{}); ok && len(srcList) > 0 {
			ipv4Conf.SourceAddress = ygot.String(srcList[0].(string))
		}
	}
	entry.Ipv4.Config = ipv4Conf

	if rule.Action == "discard" {
		entry.Actions.Config.ForwardingAction = oc.OpenconfigAcl_FORWARDING_ACTION_DROP
	} else {
		entry.Actions.Config.ForwardingAction = oc.OpenconfigAcl_FORWARDING_ACTION_ACCEPT
	}

	return entry, nil
}

// buildFirewallAclBinding creates a ygot struct for binding ACLs to an interface.
func buildFirewallAclBinding(inFilter, outFilter string) (ygot.GoStruct, error) {
	acl := &oc.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Acl{
		Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Acl_Config{},
	}
	if inFilter != "" {
		acl.Config.IngressAclSet = ygot.String(inFilter)
	}
	if outFilter != "" {
		acl.Config.EgressAclSet = ygot.String(outFilter)
	}
	return acl, nil
}

// buildInterfaceConfig creates a ygot struct for basic interface properties.
func buildInterfaceConfig(ifName string, enabled bool, description string, speed uint32) (ygot.GoStruct, error) {
	iface := &oc.OpenconfigInterfaces_Interfaces_Interface{
		Name: ygot.String(ifName),
		Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Config{
			Name:        ygot.String(ifName),
			Enabled:     ygot.Bool(enabled),
			Description: ygot.String(description),
		},
	}
	if speed > 0 {
		iface.Ethernet = &oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet{
			Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet_Config{
				PortSpeed: oc.E_OpenconfigIfEthernet_SPEED(speed),
			},
		}
	}
	return iface, nil
}

// buildLACPConfig creates a ygot struct for LACP configuration on an AE interface.
func buildLACPConfig(ifName string, lacpMode oc.E_OpenconfigLacp_LacpActivityType) (ygot.GoStruct, error) {
	iface := &oc.OpenconfigInterfaces_Interfaces_Interface{
		Name: ygot.String(ifName),
		Aggregation: &oc.OpenconfigInterfaces_Interfaces_Interface_Aggregation{
			Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Aggregation_Config{
				LagType: oc.OpenconfigIfAggregate_AggregationType_LACP,
			},
			Lacp: &oc.OpenconfigInterfaces_Interfaces_Interface_Aggregation_Lacp{
				Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Aggregation_Lacp_Config{
					LacpMode: lacpMode,
				},
			},
		},
	}
	return iface, nil
}

// buildSubinterfaceIPConfig creates a ygot struct for an L3 subinterface.
func buildSubinterfaceIPConfig(ifName string, subifIndex int, ipAddressCIDR string) (ygot.GoStruct, error) {
	parts := strings.Split(ipAddressCIDR, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid CIDR format: %s", ipAddressCIDR)
	}
	ip := parts[0]
	prefixLen, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid prefix length: %s", parts[1])
	}
	subif := &oc.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface{
		Index: ygot.Uint32(uint32(subifIndex)),
		Ipv4: &oc.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4{
			Addresses: &oc.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses{
				Address: map[string]*oc.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address{
					ip: {
						Ip: ygot.String(ip),
						Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Config{
							Ip:           ygot.String(ip),
							PrefixLength: ygot.Uint8(uint8(prefixLen)),
						},
					},
				},
			},
		},
	}
	return subif, nil
}

// buildSwitchedVlanConfig creates a ygot struct for L2 switched-vlan configuration.
func buildSwitchedVlanConfig(mode string, vlans []string) (*oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan, error) {
	config := &oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config{}
	switch mode {
	case "access":
		if len(vlans) != 1 {
			return nil, fmt.Errorf("access mode requires exactly one VLAN, got %d", len(vlans))
		}
		vlanID, err := strconv.Atoi(vlans[0])
		if err != nil {
			return nil, fmt.Errorf("invalid VLAN ID for access mode: %s", vlans[0])
		}
		config.InterfaceMode = oc.OpenconfigVlan_VlanModeType_ACCESS
		config.AccessVlan = ygot.Uint16(uint16(vlanID))
	case "trunk":
		var trunkVlans []oc.OpenconfigVlan_VlanTypes_VlanId_Union
		for _, v := range vlans {
			vlanID, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid VLAN ID for trunk mode: %s", v)
			}
			trunkVlans = append(trunkVlans, oc.UnionUint16(uint16(vlanID)))
		}
		config.InterfaceMode = oc.OpenconfigVlan_VlanModeType_TRUNK
		config.TrunkVlans = trunkVlans
	default:
		return nil, fmt.Errorf("unsupported interface mode: %s", mode)
	}
	return &oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan{
		Config: config,
	}, nil
}

// buildVrfAfiSafiConfig creates a ygot struct for VRF import/export route-targets.
func buildVrfAfiSafiConfig(importRTs, exportRTs []string) (ygot.GoStruct, error) {
	var importUnion []oc.OpenconfigRoutingPolicy_RoutingPolicy_BgpConditions_MatchSetOptions_Config_MatchSetOptions_Union
	for _, rt := range importRTs {
		importUnion = append(importUnion, oc.UnionString(rt))
	}
	var exportUnion []oc.OpenconfigRoutingPolicy_RoutingPolicy_BgpConditions_MatchSetOptions_Config_MatchSetOptions_Union
	for _, rt := range exportRTs {
		exportUnion = append(exportUnion, oc.UnionString(rt))
	}
	ni := &oc.OpenconfigNetworkInstance_NetworkInstances_NetworkInstance_AfiSafis_AfiSafi{
		AfiSafiName: oc.OpenconfigBgpTypes_AFI_SAFI_TYPE_IPV4_UNICAST,
		Config: &oc.OpenconfigNetworkInstance_NetworkInstances_NetworkInstance_AfiSafis_AfiSafi_Config{
			AfiSafiName: oc.OpenconfigBgpTypes_AFI_SAFI_TYPE_IPV4_UNICAST,
		},
		ImportExportPolicy: &oc.OpenconfigNetworkInstance_NetworkInstances_NetworkInstance_AfiSafis_AfiSafi_ImportExportPolicy{
			Config: &oc.OpenconfigNetworkInstance_NetworkInstances_NetworkInstance_AfiSafis_AfiSafi_ImportExportPolicy_Config{
				ImportRouteTarget: importUnion,
				ExportRouteTarget: exportUnion,
			},
		},
	}
	return ni, nil
}

// modifyRTList is a helper to add or remove an item from a slice of strings.
func modifyRTList(list []string, item, action string) []string {
	set := make(map[string]bool)
	for _, s := range list {
		set[s] = true
	}
	if action == "add" {
		set[item] = true
	} else { // "delete"
		delete(set, item)
	}
	newList := make([]string, 0, len(set))
	for s := range set {
		newList = append(newList, s)
	}
	return newList
}

