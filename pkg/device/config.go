// File: pkg/device/config.go
// Chat Session: newtron-20250804-01
// Session Timestamp: 2025-08-04T00:00:00Z
package device

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/openconfig/ygot/ygot"
	"newtron/pkg/oc"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// ConfigBaseline is the high-level orchestrator for applying Day-1 configuration.
// This function will be called by the NewtronBaseline application.
func (d *Device) ConfigBaseline() error {
	fmt.Printf("\n>> [Backend API] Applying baseline configuration to %s...\n", d.Node.Name)

	// The orchestrator calls a series of helper functions and applies internal configlets
	// in a specific order to ensure the device is configured correctly.
	
	// Example of the orchestration flow:
	if err := d.applyConfiglet("system_common"); err != nil {
		return fmt.Errorf("failed to apply system_common configlet: %w", err)
	}
	if err := d.applyConfiglet("qos_common"); err != nil {
		return fmt.Errorf("failed to apply qos_common configlet: %w", err)
	}
	if err := d.applyConfiglet("routing_common"); err != nil {
		return fmt.Errorf("failed to apply routing_common configlet: %w", err)
	}
	
	// In a real implementation, we would add more calls here for management config, etc.
	fmt.Printf(">> [Backend API] Baseline configuration successfully applied.\n")
	return nil
}


// ConfigInterfaceAdminStatus enables or disables a physical interface.
func (d *Device) ConfigInterfaceAdminStatus(port *Port, action string) error {
	fmt.Printf("\n>> [Backend API] Setting admin state of %s to '%s'\n", port.IfName, action)
	enabled := action == "enable"

	iface := &oc.OpenconfigInterfaces_Interfaces_Interface{
		Name: ygot.String(port.IfName),
		Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Config{
			Name:    ygot.String(port.IfName),
			Enabled: ygot.Bool(enabled),
		},
	}

	path := fmt.Sprintf("/interfaces/interface[name=%s]/config/enabled", port.IfName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return d.gnmiSet(ctx, path, gpb.UpdateResult_UPDATE, iface)
}

// ConfigInterfaceSpeed sets the speed on a physical Ethernet interface.
func (d *Device) ConfigInterfaceSpeed(port *Port, speed string) error {
	fmt.Printf("\n>> [Backend API] Setting speed of %s to '%s'\n", port.IfName, speed)

	speedVal, err := strconv.ParseUint(speed, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid speed value: %s", speed)
	}
	ocSpeed := oc.E_OpenconfigIfEthernet_SPEED(speedVal)

	iface := &oc.OpenconfigInterfaces_Interfaces_Interface{
		Name: ygot.String(port.IfName),
		Ethernet: &oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet{
			Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet_Config{
				PortSpeed: ocSpeed,
			},
		},
	}

	path := fmt.Sprintf("/interfaces/interface[name=%s]/ethernet/config/port-speed", port.IfName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return d.gnmiSet(ctx, path, gpb.UpdateResult_UPDATE, iface)
}

// ConfigAEInterfaceMember adds or removes a member from a LAG.
func (d *Device) ConfigAEInterfaceMember(aePort, memberPort *Port, action string) error {
	fmt.Printf("\n>> [Backend API] %s member %s %s bundle %s\n", action, memberPort.IfName, map[string]string{"add": "to", "remove": "from"}[action], aePort.IfName)

	if action == "add" {
		memberIface := &oc.OpenconfigInterfaces_Interfaces_Interface{
			Name: ygot.String(memberPort.IfName),
			Ethernet: &oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet{
				Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet_Config{
					AggregateId: ygot.String(aePort.IfName),
				},
			},
		}
		path := fmt.Sprintf("/interfaces/interface[name=%s]/ethernet/config/aggregate-id", memberPort.IfName)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return d.gnmiSet(ctx, path, gpb.UpdateResult_UPDATE, memberIface)
	} else { // "remove"
		path := fmt.Sprintf("/interfaces/interface[name=%s]/ethernet/config/aggregate-id", memberPort.IfName)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return d.gnmiSet(ctx, path, gpb.UpdateResult_DELETE, nil)
	}
}

// ConfigAEInterfaceLACP enables or disables LACP on a bundle.
func (d *Device) ConfigAEInterfaceLACP(aePort *Port, action string) error {
	fmt.Printf("\n>> [Backend API] Setting LACP on %s to '%s'\n", aePort.IfName, action)

	var lagType oc.E_OpenconfigIfAggregate_AggregationType
	if action == "enable" {
		lagType = oc.OpenconfigIfAggregate_AggregationType_LACP
	} else { // "disable"
		lagType = oc.OpenconfigIfAggregate_AggregationType_STATIC
	}

	iface := &oc.OpenconfigInterfaces_Interfaces_Interface{
		Name: ygot.String(aePort.IfName),
		Aggregation: &oc.OpenconfigInterfaces_Interfaces_Interface_Aggregation{
			Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Aggregation_Config{
				LagType: lagType,
			},
		},
	}

	path := fmt.Sprintf("/interfaces/interface[name=%s]/aggregation/config/lag-type", aePort.IfName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return d.gnmiSet(ctx, path, gpb.UpdateResult_UPDATE, iface)
}

// ConfigBridgeInterface configures L2 properties on a physical port.
func (d *Device) ConfigBridgeInterface(port *Port, action, mode string, vlans []string) error {
	fmt.Printf("\n>> [Backend API] %s bridge interface on %s (Mode: %s, VLANs: %s)\n", action, port.IfName, mode, strings.Join(vlans, ","))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	path := fmt.Sprintf("/interfaces/interface[name=%s]/ethernet/switched-vlan", port.IfName)

	if action == "delete" {
		return d.gnmiSet(ctx, path, gpb.UpdateResult_DELETE, nil)
	}

	swvlanConfig, err := buildSwitchedVlanConfig(mode, vlans)
	if err != nil {
		return err
	}

	iface := &oc.OpenconfigInterfaces_Interfaces_Interface{
		Name: ygot.String(port.IfName),
		Ethernet: &oc.OpenconfigInterfaces_Interfaces_Interface_Ethernet{
			SwitchedVlan: swvlanConfig,
		},
	}

	return d.gnmiSet(ctx, path, gpb.UpdateResult_REPLACE, iface)
}

// ConfigSubInterface adds or deletes a subinterface, including its L3 configuration.
func (d *Device) ConfigSubInterface(port *Port, subIfID int, ip, service, action string) error {
	fmt.Printf("\n>> [Backend API] %s sub-interface %s.%d with IP %s\n", action, port.IfName, subIfID, ip)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	path := fmt.Sprintf("/interfaces/interface[name=%s]/subinterfaces/subinterface[index=%d]", port.IfName, subIfID)

	if action == "delete" {
		return d.gnmiSet(ctx, path, gpb.UpdateResult_DELETE, nil)
	}

	subif, err := buildSubinterfaceIPConfig(port.IfName, subIfID, ip)
	if err != nil {
		return err
	}

	return d.gnmiSet(ctx, path, gpb.UpdateResult_UPDATE, subif)
}

// ConfigSubInterfaceIPVPN adds or removes a VPN's route-targets from a VRF.
func (d *Device) ConfigSubInterfaceIPVPN(subInterface *SubInterface, vpn *VPN, action string) error {
	fmt.Printf("\n>> [Backend API] %s VPN %s %s VRF %s\n",
		strings.Title(action),
		vpn.Name,
		map[string]string{"add": "to", "delete": "from"}[action],
		subInterface.VRF.Name)

	vpnIntent, ok := d.Intent.VPNs[vpn.Name]
	if !ok {
		return fmt.Errorf("VPN intent for '%s' not found", vpn.Name)
	}
	importRT := vpnIntent.ImportTarget
	exportRT := vpnIntent.ExportTarget

	if err := d.LoadVRFDetail(subInterface.VRF); err != nil {
		return fmt.Errorf("could not load current VRF state: %w", err)
	}

	newImportRTs := modifyRTList(subInterface.VRF.ImportRouteTargets, importRT, action)
	newExportRTs := modifyRTList(subInterface.VRF.ExportRouteTargets, exportRT, action)

	ni, err := buildVrfAfiSafiConfig(newImportRTs, newExportRTs)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/network-instances/network-instance[name=%s]/afi-safis/afi-safi[afi-safi-name=IPV4_UNICAST]/config", subInterface.VRF.Name)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return d.gnmiSet(ctx, path, gpb.UpdateResult_REPLACE, ni)
}

// ConfigFirewallPolicy generates and applies the correct ingress and egress firewall filters to a subinterface.
func (d *Device) ConfigFirewallPolicy(subInterface *SubInterface) error {
	fmt.Printf("\n>> [Backend API] Applying firewall filter to %s\n", subInterface.IfName)

	serviceIntent, ok := d.Intent.Services[subInterface.Service]
	if !ok {
		return fmt.Errorf("service intent for '%s' not found", subInterface.Service)
	}

	policyMap, ok := serviceIntent.InterfacePolicy["all"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("could not find 'all' interface policy for service '%s'", subInterface.Service)
	}

	inFilterName, ok := policyMap["in_name"].(string)
	if !ok {
		return fmt.Errorf("invalid 'in_name' in policy map for service '%s'", subInterface.Service)
	}
	outFilterName, ok := policyMap["out_name"].(string)
	if !ok {
		return fmt.Errorf("invalid 'out_name' in policy map for service '%s'", subInterface.Service)
	}
	inRuleList, ok := policyMap["in_rule"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid 'in_rule' in policy map for service '%s'", subInterface.Service)
	}
	outRuleList, ok := policyMap["out_rule"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid 'out_rule' in policy map for service '%s'", subInterface.Service)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Build and apply the Ingress ACL
	if inFilterName != "" && len(inRuleList) > 0 {
		fmt.Printf("  [Backend API] Building Ingress ACL: %s\n", inFilterName)
		inAclSet, err := buildAclSet(inFilterName, inRuleList, d.Intent)
		if err != nil {
			return fmt.Errorf("could not build ingress ACL ygot struct: %w", err)
		}
		aclPath := fmt.Sprintf("/acl/acl-sets/acl-set[name=%s][type=ACL_IPV4]", inFilterName)
		if err := d.gnmiSet(ctx, aclPath, gpb.UpdateResult_REPLACE, inAclSet); err != nil {
			return fmt.Errorf("failed to apply ingress ACL set: %w", err)
		}
	}

	// 2. Build and apply the Egress ACL
	if outFilterName != "" && len(outRuleList) > 0 {
		fmt.Printf("  [Backend API] Building Egress ACL: %s\n", outFilterName)
		outAclSet, err := buildAclSet(outFilterName, outRuleList, d.Intent)
		if err != nil {
			return fmt.Errorf("could not build egress ACL ygot struct: %w", err)
		}
		aclPath := fmt.Sprintf("/acl/acl-sets/acl-set[name=%s][type=ACL_IPV4]", outFilterName)
		if err := d.gnmiSet(ctx, aclPath, gpb.UpdateResult_REPLACE, outAclSet); err != nil {
			return fmt.Errorf("failed to apply egress ACL set: %w", err)
		}
	}

	// 3. Bind the ACLs to the interface
	fmt.Printf("  [Backend API] Binding ACLs to interface %s\n", subInterface.IfName)
	aclBinding, err := buildFirewallAclBinding(inFilterName, outFilterName)
	if err != nil {
		return err
	}
	bindingPath := fmt.Sprintf("/interfaces/interface[name=%s]/subinterfaces/subinterface[index=%d]/ipv4/acl", subInterface.ParentPort.IfName, subInterface.ID)

	return d.gnmiSet(ctx, bindingPath, gpb.UpdateResult_REPLACE, aclBinding)
}

// ConfigSubInterfaceIPAdminStatus enables or disables a sub-interface.
func (d *Device) ConfigSubInterfaceIPAdminStatus(subInterface *SubInterface, action string) error {
    fmt.Printf("\n>> [Backend API] Setting admin state of %s to '%s'\n", subInterface.IfName, action)
    enabled := action == "enable"

    subif := &oc.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface{
        Index: ygot.Uint32(uint32(subInterface.ID)),
        Config: &oc.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Config{
            Enabled: ygot.Bool(enabled),
        },
    }

    path := fmt.Sprintf("/interfaces/interface[name=%s]/subinterfaces/subinterface[index=%d]/config/enabled", subInterface.ParentPort.IfName, subInterface.ID)
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    return d.gnmiSet(ctx, path, gpb.UpdateResult_UPDATE, subif)
}

// ConfigSubInterfaceBandwidth sets shaping on a sub-interface (stub for now; implement later).
func (d *Device) ConfigSubInterfaceBandwidth(subInterface *SubInterface, bw string) error {
    fmt.Printf("\n>> [Backend API] Setting bandwidth on %s to '%s' (stubbed)\n", subInterface.IfName, bw)
    // TODO: Build ygot struct for QoS/shaping and apply via gNMI.
    return nil // Stub: No-op for now to allow compilation.
}