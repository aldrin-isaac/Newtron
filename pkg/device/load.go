// File: pkg/device/load.go
// Chat Session: newtron-20250804-01
// Session Timestamp: 2025-08-04T00:00:00Z
package device

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// --- Implemented Load Functions ---

// loadInitialState is called by Connect to get basic device info.
func (d *Device) loadInitialState() error {
	fmt.Println("  [Backend API] Loading initial device state (version, chassis)...")
	type SystemState struct {
		Hostname        string `json:"hostname"`
		Platform        string `json:"platform"`
		SoftwareVersion string `json:"software-version"`
	}
	type SystemContainer struct {
		State SystemState `json:"state"`
	}
	type OpenconfigSystem struct {
		System SystemContainer `json:"openconfig-system:system"`
	}

	path, err := StringToPath("/system/state")
	if err != nil {
		return fmt.Errorf("failed to create gNMI path: %w", err)
	}

	req := &gpb.GetRequest{
		Path:     []*gpb.Path{path},
		Type:     gpb.GetRequest_STATE,
		Encoding: gpb.Encoding_JSON_IETF,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := d.gnmiClient.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("gNMI Get for system state failed: %w", err)
	}

	if len(res.GetNotification()) > 0 && len(res.GetNotification()[0].GetUpdate()) > 0 {
		val := res.GetNotification()[0].GetUpdate()[0].GetVal()
		jsonVal := val.GetJsonIetfVal()

		var data OpenconfigSystem
		if err := json.Unmarshal(jsonVal, &data); err != nil {
			return fmt.Errorf("failed to unmarshal system state JSON: %w", err)
		}
		if data.System.State.Hostname == "" {
			return fmt.Errorf("incomplete system state data received")
		}

		d.Node.Name = data.System.State.Hostname
		d.Node.Version = data.System.State.SoftwareVersion
		d.Node.Chassis = data.System.State.Platform
		d.Node.ConfigClass = d.Intent.GenericAlias["config-class"]
	} else {
		return fmt.Errorf("no system state data received from device")
	}

	return nil
}

// LoadCards fetches hardware component information from the device using gNMI.
func (d *Device) LoadCards() error {
	fmt.Println("  [Backend API] Loading cards via gNMI...")
	type ComponentState struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
	}
	type Component struct {
		Name  string         `json:"name"`
		State ComponentState `json:"state"`
	}
	type ComponentsContainer struct {
		Component []Component `json:"component"`
	}
	type OpenconfigPlatform struct {
		Components ComponentsContainer `json:"openconfig-platform:components"`
	}

	path, err := StringToPath("/components")
	if err != nil {
		return fmt.Errorf("failed to create gNMI path: %w", err)
	}

	req := &gpb.GetRequest{
		Path:     []*gpb.Path{path},
		Type:     gpb.GetRequest_STATE,
		Encoding: gpb.Encoding_JSON_IETF,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res, err := d.gnmiClient.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("gNMI Get for components failed: %w", err)
	}

	d.Node.Cards = make(map[string]*Card) // Reset card list
	if len(res.GetNotification()) > 0 && len(res.GetNotification()[0].GetUpdate()) > 0 {
		val := res.GetNotification()[0].GetUpdate()[0].GetVal()
		jsonVal := val.GetJsonIetfVal()

		var data OpenconfigPlatform
		if err := json.Unmarshal(jsonVal, &data); err != nil {
			return fmt.Errorf("failed to unmarshal components JSON: %w", err)
		}

		re := regexp.MustCompile(`FPC-(\d+)`)
		for _, comp := range data.Components.Component {
			if strings.Contains(comp.State.Type, "FPC") {
				matches := re.FindStringSubmatch(comp.Name)
				if len(matches) > 1 {
					cardID := matches[1]
					card := &Card{
						ID:          cardID,
						Model:       comp.Name,
						Description: comp.State.Description,
						ParentNode:  d.Node,
					}
					d.Node.Cards[cardID] = card
				}
			}
		}
	}

	d.Node.Cards["ae"] = &Card{ID: "ae", Model: "Aggregated-Ethernet", Description: "Aggregated Ethernet", PortType: "ae", ParentNode: d.Node}
	d.Node.Cards["irb"] = &Card{ID: "irb", Model: "IRB", Description: "Integrated Routing and Bridging", PortType: "irb", ParentNode: d.Node}

	return nil
}

// LoadPorts fetches physical interface information for a specific card using gNMI.
func (d *Device) LoadPorts(card *Card) error {
	fmt.Printf("  [Backend API] Loading ports for card %s via gNMI...\n", card.ID)
	type InterfaceState struct {
		Name        string `json:"name"`
		AdminStatus string `json:"admin-status"`
		OperStatus  string `json:"oper-status"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}
	type Interface struct {
		Name  string         `json:"name"`
		State InterfaceState `json:"state"`
	}
	type InterfacesContainer struct {
		Interface []Interface `json:"interface"`
	}
	type OpenconfigInterfaces struct {
		Interfaces InterfacesContainer `json:"openconfig-interfaces:interfaces"`
	}

	path, err := StringToPath("/interfaces")
	if err != nil {
		return fmt.Errorf("failed to create gNMI path: %w", err)
	}

	req := &gpb.GetRequest{
		Path:     []*gpb.Path{path},
		Type:     gpb.GetRequest_STATE,
		Encoding: gpb.Encoding_JSON_IETF,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res, err := d.gnmiClient.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("gNMI Get for interfaces failed: %w", err)
	}

	card.Ports = []*Port{} // Clear previous data
	if len(res.GetNotification()) > 0 && len(res.GetNotification()[0].GetUpdate()) > 0 {
		val := res.GetNotification()[0].GetUpdate()[0].GetVal()
		jsonVal := val.GetJsonIetfVal()

		var data OpenconfigInterfaces
		if err := json.Unmarshal(jsonVal, &data); err != nil {
			return fmt.Errorf("failed to unmarshal interfaces JSON: %w", err)
		}

		re := regexp.MustCompile(`\w+-(\d+)/\d+/(\d+)`)
		for _, iface := range data.Interfaces.Interface {
			matches := re.FindStringSubmatch(iface.Name)
			if len(matches) == 3 && matches[1] == card.ID {
				portID := matches[2]
				port := &Port{
					ID:          portID,
					ParentCard:  card,
					IfName:      iface.Name,
					OperStatus:  iface.State.OperStatus,
					AdminStatus: iface.State.AdminStatus,
					Bridging:    card.Bridging,
				}
				card.Ports = append(card.Ports, port)
			}
		}
	}

	return nil
}

// LoadPortDetail fetches detailed state for a single port, including subinterfaces.
func (d *Device) LoadPortDetail(port *Port) error {
	fmt.Printf("  [Backend API] Loading details for port %s via gNMI...\n", port.IfName)

	type SubifState struct {
		Index       uint32 `json:"index"`
		Description string `json:"description"`
		AdminStatus string `json:"admin-status"`
	}
	type SubifAddressState struct {
		IP           string `json:"ip"`
		PrefixLength uint8  `json:"prefix-length"`
	}
	type SubifAddress struct {
		State SubifAddressState `json:"state"`
	}
	type SubifAddresses struct {
		Address map[string]SubifAddress `json:"address"`
	}
	type SubifIPv4 struct {
		Addresses SubifAddresses `json:"addresses"`
	}
	type Subinterface struct {
		Index uint32      `json:"index"`
		State SubifState  `json:"state"`
		IPv4  SubifIPv4   `json:"openconfig-if-ip:ipv4"`
	}
	type SubinterfacesContainer struct {
		Subinterface []Subinterface `json:"subinterface"`
	}
	type Interface struct {
		Subinterfaces SubinterfacesContainer `json:"subinterfaces"`
	}
	type InterfacesContainer struct {
		Interface []Interface `json:"interface"`
	}
	type OpenconfigInterfaces struct {
		Interfaces InterfacesContainer `json:"openconfig-interfaces:interfaces"`
	}

	path, err := StringToPath(fmt.Sprintf("/interfaces/interface[name=%s]", port.IfName))
	if err != nil {
		return fmt.Errorf("failed to create gNMI path for port detail: %w", err)
	}

	req := &gpb.GetRequest{Path: []*gpb.Path{path}, Type: gpb.GetRequest_STATE, Encoding: gpb.Encoding_JSON_IETF}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := d.gnmiClient.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("gNMI Get for port detail failed: %w", err)
	}

	port.SubInterfaces = []*SubInterface{} // Clear previous data
	if len(res.GetNotification()) > 0 && len(res.GetNotification()[0].GetUpdate()) > 0 {
		val := res.GetNotification()[0].GetUpdate()[0].GetVal()
		jsonVal := val.GetJsonIetfVal()

		var data OpenconfigInterfaces
		if err := json.Unmarshal(jsonVal, &data); err != nil {
			return fmt.Errorf("failed to unmarshal port detail JSON: %w", err)
		}

		if len(data.Interfaces.Interface) > 0 {
			for _, subif := range data.Interfaces.Interface[0].Subinterfaces.Subinterface {
				newSub := &SubInterface{
					ID:          int(subif.Index),
					ParentPort:  port,
					IfName:      fmt.Sprintf("%s.%d", port.IfName, subif.Index),
					Description: subif.State.Description,
					AdminStatus: subif.State.AdminStatus,
				}
				for ip, addr := range subif.IPv4.Addresses.Address {
					newSub.IPAddress = fmt.Sprintf("%s/%d", ip, addr.State.PrefixLength)
					break // Assuming one IP per subinterface for now
				}
				// Set Service by matching IP to intent (simple prefix match for now)
				for prefix, services := range d.Intent.PrefixServiceMapping {
					if strings.HasPrefix(newSub.IPAddress, prefix) {
						newSub.Service = services[0] // Assume first match
						break
					}
				}
				// Set FilterName (query ACL binding; stub for now or add gNMI Get for /acl/interfaces/interface[...])
				newSub.FilterName = "stub-filter" // Temp
				port.SubInterfaces = append(port.SubInterfaces, newSub)
			}
		}
	}
	return nil
}

// LoadLAGDetail fetches LAG details for an AE port.
func (d *Device) LoadLAGDetail(aePort *Port) error {
	fmt.Printf("  [Backend API] Loading LAG details for %s via gNMI...\n", aePort.IfName)

	type LagState struct {
		LACPMode string `json:"lacp-mode"`
	}
	type LagMemberState struct {
		Interface string `json:"interface"`
	}
	type LagMember struct {
		State LagMemberState `json:"state"`
	}
	type AggregationState struct {
		LagType string   `json:"lag-type"`
		Members []string `json:"member"`
	}
	type Aggregation struct {
		State   AggregationState   `json:"state"`
		Members map[string]LagMember `json:"members"`
	}
	type Interface struct {
		Aggregation Aggregation `json:"openconfig-if-aggregate:aggregation"`
	}
	type InterfacesContainer struct {
		Interface []Interface `json:"interface"`
	}
	type OpenconfigInterfaces struct {
		Interfaces InterfacesContainer `json:"openconfig-interfaces:interfaces"`
	}

	path, err := StringToPath(fmt.Sprintf("/interfaces/interface[name=%s]", aePort.IfName))
	if err != nil {
		return fmt.Errorf("failed to create gNMI path for LAG detail: %w", err)
	}

	req := &gpb.GetRequest{Path: []*gpb.Path{path}, Type: gpb.GetRequest_STATE, Encoding: gpb.Encoding_JSON_IETF}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := d.gnmiClient.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("gNMI Get for LAG detail failed: %w", err)
	}

	aePort.LAG = &LAG{} // Initialize LAG struct
	if len(res.GetNotification()) > 0 && len(res.GetNotification()[0].GetUpdate()) > 0 {
		val := res.GetNotification()[0].GetUpdate()[0].GetVal()
		jsonVal := val.GetJsonIetfVal()

		var data OpenconfigInterfaces
		if err := json.Unmarshal(jsonVal, &data); err != nil {
			return fmt.Errorf("failed to unmarshal LAG detail JSON: %w", err)
		}

		if len(data.Interfaces.Interface) > 0 {
			agg := data.Interfaces.Interface[0].Aggregation
			aePort.LAG.LACPEnabled = agg.State.LagType == "LACP"
			aePort.LAG.LACPMode = agg.State.LACPMode
			for _, memberName := range agg.State.Members {
				aePort.LAG.Members = append(aePort.LAG.Members, &Port{IfName: memberName})
			}
		}
	}
	return nil
}

// LoadVRFDetail fetches details for a specific VRF.
func (d *Device) LoadVRFDetail(vrf *VRF) error {
	fmt.Printf("  [Backend API] Loading details for VRF %s via gNMI...\n", vrf.Name)

	type VrfState struct {
		RouteDistinguisher string `json:"route-distinguisher"`
	}
	type VrfAfiSafiState struct {
		ImportRouteTarget []string `json:"import-route-target"`
		ExportRouteTarget []string `json:"export-route-target"`
	}
	type VrfAfiSafi struct {
		State VrfAfiSafiState `json:"state"`
	}
	type VrfAfiSafis struct {
		AfiSafi map[string]VrfAfiSafi `json:"afi-safi"`
	}
	type NetworkInstance struct {
		State    VrfState    `json:"state"`
		AfiSafis VrfAfiSafis `json:"afi-safis"`
	}
	type NetworkInstancesContainer struct {
		NetworkInstance []NetworkInstance `json:"network-instance"`
	}
	type OpenconfigNetworkInstance struct {
		NetworkInstances NetworkInstancesContainer `json:"openconfig-network-instance:network-instances"`
	}

	path, err := StringToPath(fmt.Sprintf("/network-instances/network-instance[name=%s]", vrf.Name))
	if err != nil {
		return fmt.Errorf("failed to create gNMI path for VRF detail: %w", err)
	}

	req := &gpb.GetRequest{Path: []*gpb.Path{path}, Type: gpb.GetRequest_STATE, Encoding: gpb.Encoding_JSON_IETF}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := d.gnmiClient.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("gNMI Get for VRF detail failed: %w", err)
	}

	if len(res.GetNotification()) > 0 && len(res.GetNotification()[0].GetUpdate()) > 0 {
		val := res.GetNotification()[0].GetUpdate()[0].GetVal()
		jsonVal := val.GetJsonIetfVal()

		var data OpenconfigNetworkInstance
		if err := json.Unmarshal(jsonVal, &data); err != nil {
			return fmt.Errorf("failed to unmarshal VRF detail JSON: %w", err)
		}

		if len(data.NetworkInstances.NetworkInstance) > 0 {
			ni := data.NetworkInstances.NetworkInstance[0]
			vrf.RouteDistinguisher = ni.State.RouteDistinguisher
			if ipv4, ok := ni.AfiSafis.AfiSafi["IPV4_UNICAST"]; ok {
				vrf.ImportRouteTargets = ipv4.State.ImportRouteTarget
				vrf.ExportRouteTargets = ipv4.State.ExportRouteTarget
			}
			vrf.AssociatedVPNs = make(map[string]*VPN)
			for vpnName, vpnIntent := range d.Intent.VPNs {
				for _, rt := range vrf.ImportRouteTargets {
					if rt == vpnIntent.ImportTarget {
						vrf.AssociatedVPNs[vpnName] = &VPN{Name: vpnName, Description: vpnIntent.Description}
						break
					}
				}
			}
		}
	}
	return nil
}

// --- State vs. Intent Lookup Functions ---

// GetAvailableVPNs compares the configured state with the global intent to find
// VPNs that can be added to a sub-interface's VRF.
func (d *Device) GetAvailableVPNs(subInterface *SubInterface) ([]*VPN, error) {
	fmt.Println("  [Backend API] Getting available VPNs...")

	if subInterface.VRF == nil {
		return nil, fmt.Errorf("subinterface %s is not associated with a VRF", subInterface.IfName)
	}

	// 1. Ensure the VRF's current state is loaded from the device.
	if err := d.LoadVRFDetail(subInterface.VRF); err != nil {
		return nil, fmt.Errorf("could not load current VRF state: %w", err)
	}

	// 2. Get the list of all possible VPNs from the global intent.
	allPossibleVPNs := d.Intent.Services[subInterface.Service].RoutingBehavior["vrf_default_vpn"].([]interface{})

	// 3. Get the list of already configured VPNs from the live device state.
	configuredVPNs := subInterface.VRF.AssociatedVPNs

	// 4. Calculate the difference.
	availableVPNs := []*VPN{}
	for _, vpnNameIntf := range allPossibleVPNs {
		vpnName := vpnNameIntf.(string)
		if _, isConfigured := configuredVPNs[vpnName]; !isConfigured {
			// This VPN is a candidate to be added.
			if vpnIntent, ok := d.Intent.VPNs[vpnName]; ok {
				availableVPNs = append(availableVPNs, &VPN{
					Name:        vpnName,
					Description: vpnIntent.Description,
				})
			}
		}
	}

	return availableVPNs, nil
}

// GetNextAvailableAEInterface finds the next available (unconfigured) Aggregated Ethernet interface ID.
func (d *Device) GetNextAvailableAEInterface() (*Port, error) {
	fmt.Println("  [Backend API] Finding next available AE interface...")

	aeCard, ok := d.Node.Cards["ae"]
	if !ok {
		return nil, fmt.Errorf("AE card definition not found in node's card list")
	}

	// 1. Get all possible AE interfaces from intent
	validPorts, err := expandPortRanges(aeCard.ValidPorts)
	if err != nil {
		return nil, fmt.Errorf("could not parse valid AE port ranges: %w", err)
	}

	// 2. Get all configured AE interfaces from the device
	if err := d.LoadPorts(aeCard); err != nil {
		return nil, fmt.Errorf("could not load configured AE ports: %w", err)
	}

	configuredPorts := make(map[string]bool)
	for _, port := range aeCard.Ports {
		configuredPorts[port.ID] = true
	}

	// 3. Find the first valid port that is not configured
	for _, portID := range validPorts {
		if !configuredPorts[portID] {
			return &Port{
				ID:         portID,
				ParentCard: aeCard,
				IfName:     fmt.Sprintf("ae%s", portID),
			}, nil
		}
	}

	return nil, fmt.Errorf("no available AE interfaces found")
}

// GetAvailableVLANs finds VLANs that are allowed by intent but not yet configured on a port.
func (d *Device) GetAvailableVLANs(port *Port) ([]string, error) {
	fmt.Printf("  [Backend API] Getting available VLANs for port %s...\n", port.IfName)

	// 1. Get VLANs allowed by intent for this port
	allowedVLANs := d.Intent.VLANPortMapping["default"] // Placeholder

	// 2. Get VLANs currently configured on the port from the live device state
	if err := d.LoadPortDetail(port); err != nil {
		return nil, fmt.Errorf("could not load port details to check current VLANs: %w", err)
	}

	configuredVLANs := make(map[string]bool)
	for _, sub := range port.SubInterfaces {
		vlanID := strconv.Itoa(sub.ID)
		configuredVLANs[vlanID] = true
	}

	// 3. Calculate the difference
	availableVLANs := []string{}
	for _, vlan := range allowedVLANs {
		if !configuredVLANs[vlan] {
			availableVLANs = append(availableVLANs, vlan)
		}
	}

	return availableVLANs, nil
}