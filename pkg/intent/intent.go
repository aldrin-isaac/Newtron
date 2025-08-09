// File: pkg/intent/intent.go
// Chat Session: newtron-20250804-01
// Session Timestamp: 2025-08-04T00:00:00Z
package intent

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// --- Top-Level Intent Structs ---

type ResolvedIntent struct {
	DeviceName           string
	Region               string
	Site                 string
	MgmtIP               string
	IsRouter             bool
	IsBridge             bool
	IsBorderRouter       bool
	IsRouteReflector     bool
	GenericAlias         map[string]string
	ReverseAlias         map[string]string
	PrefixLists          map[string][]string
	Communities          map[string]string
	Services             map[string]Service
	VLANPortMapping      map[string][]string
	PrefixServiceMapping map[string][]string
	Permission           map[string][]string
	PolicyRules          map[string]PolicyRule
	ActiveBridgeDomain   *BridgeDomain
	CoreBGP              *CoreBGP
	VPNs                 map[string]VPN
}

type GlobalIntent struct {
	NetworkIntent    *NetworkIntent
	SiteIntent       *SiteIntent
	PlatformProfiles *PlatformProfiles
	DeviceProfiles   map[string]*DeviceProfile
}

// --- Structs for network_intent.yaml ---

type NetworkIntent struct {
	Version              string                 `yaml:"version"`
	LockDir              string                 `yaml:"lock_dir"`
	SuperUsers           []string               `yaml:"super_users"`
	ReverseExclude       []string               `yaml:"reverse_exclude"`
	Permission           map[string][]string    `yaml:"permission"`
	HealthChecks         map[string]HealthCheck `yaml:"health_checks"`
	Authentication       map[string]interface{} `yaml:"authentication"`
	Services             map[string]Service     `yaml:"services"`
	PrefixServiceMapping map[string][]string    `yaml:"prefix_service_mapping"`
	BandwidthShaping     map[string]Bandwidth   `yaml:"bandwidth_shaping"`
	QOSProfiles          map[string]string      `yaml:"qos_profiles"`
	PrefixLists          map[string][]string    `yaml:"prefix_lists"`
	PolicyRuleLists      map[string][]string    `yaml:"policy_rule_lists"`
	PolicyRules          map[string]PolicyRule  `yaml:"policy_rules"`
	CoS                  map[string]CoS         `yaml:"cos"`
	Policers             map[string]Policer     `yaml:"policers"`
	Regions              map[string]Region      `yaml:"regions"`
	VLANs                map[int]VLANInfo       `yaml:"vlans"`
	GenericAlias         map[string]string      `yaml:"generic_alias"`
	Communities          map[string]string      `yaml:"communities"`
	VPNs                 map[string]VPN         `yaml:"vpns"`
}

type HealthCheck struct {
	Method string `yaml:"method"`
}

type Service struct {
	Description     string                 `yaml:"description"`
	ServiceType     string                 `yaml:"service_type"`
	ValidIP         []string               `yaml:"valid_ip"`
	InterfacePolicy map[string]interface{} `yaml:"interface_policy"`
	RoutingBehavior map[string]interface{} `yaml:"routing_behavior"`
	PrefixListAlias map[string][]string    `yaml:"prefix_list_alias"`
}

type Bandwidth struct {
	ValidSpeeds []int `yaml:"valid_speeds"`
}

type PolicyRule struct {
	Application string                 `yaml:"application"`
	Sequence    int                    `yaml:"sequence"`
	Match       map[string]interface{} `yaml:"match"`
	CoS         string                 `yaml:"cos"`
	Policer     string                 `yaml:"policer"`
	Action      string                 `yaml:"action"`
}

type CoS struct {
	Class82 string `yaml:"class82"`
	Color82 string `yaml:"color82"`
	Mark82  string `yaml:"mark82"`
}

type Policer struct {
	Bandwidth string `yaml:"bandwidth"`
	Burst     string `yaml:"burst"`
}

type Region struct {
	PeAsNum      int                 `yaml:"pe_as_num"`
	PeAsName     string              `yaml:"pe_as_name"`
	IpMtuTransit int                 `yaml:"ip_mtu_transit"`
	GenericAlias map[string]string   `yaml:"generic_alias"`
	PrefixLists  map[string][]string `yaml:"prefix_lists"`
	Bridge       Bridge              `yaml:"bridge"`
	Management   map[string][]string `yaml:"management"`
}

type Bridge struct {
	Domains map[string]BridgeDomain `yaml:"domains"`
}

type BridgeDomain struct {
	BaselineVLANs []string         `yaml:"baseline_vlans"`
	VLANs         map[int]VLANInfo `yaml:"vlans"`
}

type VLANInfo struct {
	Description string   `yaml:"description"`
	Type        string   `yaml:"type"`
	PrefixList  []string `yaml:"prefix_list"`
}

type VPN struct {
	Description      string   `yaml:"description"`
	ImportTarget     string   `yaml:"import_target"`
	ExportTarget     string   `yaml:"export_target"`
	ExportPrefixList []string `yaml:"export_prefix_list"`
	Permission       []string `yaml:"permission"`
	Service          []string `yaml:"service"`
}

// --- Structs for site_intent.yaml ---
type SiteIntent struct {
	Regions map[string]SiteRegion `yaml:"regions"`
}
type SiteRegion struct {
	Sites map[string]Site `yaml:"sites"`
}
type Site struct {
	RouteReflectors []string `yaml:"route_reflectors"`
	SiteIP          string   `yaml:"site_ip"`
}

// --- Structs for platform.yaml ---
type PlatformProfiles struct {
	Vendors map[string]Vendor `yaml:"vendors"`
}
type Vendor struct {
	Chassis map[string]Chassis `yaml:"chassis"`
}
type Chassis struct {
	ConfigClass  string                            `yaml:"config_class"`
	ChassisClass string                            `yaml:"chassis_class"`
	NodeType     string                            `yaml:"node_type"`
	L2Capable    bool                              `yaml:"l2_capable"`
	Cards        map[string]Card                   `yaml:"cards"`
	AEProfiles   map[string]map[string]AEProfile `yaml:"ae_profiles"`
}

type Card struct {
	Description string   `yaml:"description"`
	PortType    string   `yaml:"port_type"`
	IntfProto   string   `yaml:"intf_proto"`
	IntfEncaps  []string `yaml:"intf_encaps"`
	IntfSymb    string   `yaml:"intf_symb"`
	MaximumMTU  int      `yaml:"maximum_mtu"`
	QueueType   string   `yaml:"queue_type"`
	ValidPorts  []string `yaml:"valid_ports"`
	ValidSpeeds []string `yaml:"valid_speeds"`
	ValidSubifs []string `yaml:"valid_subifs"`
	Bridging    bool     `yaml:"bridging"`
}
type AEProfile struct {
	Description       string   `yaml:"description"`
	BridgeMode        string   `yaml:"bridge_mode"`
	TrunkType         string   `yaml:"trunk_type"`
	DesignatedMembers []string `yaml:"designated_members"`
}

// --- Structs for device-specific profiles ---
type DeviceProfile struct {
	IsManaged        bool                `yaml:"is_managed"`
	Region           string              `yaml:"region"`
	Site             string              `yaml:"site"`
	MgmtIP           string              `yaml:"mgmt_ip"`
	IsRouter         bool                `yaml:"is_router"`
	IsBridge         bool                `yaml:"is_bridge"`
	IsBorderRouter   bool                `yaml:"is_border_router"`
	IsRouteReflector bool                `yaml:"is_route_reflector"`
	Affinity         string              `yaml:"affinity"`
	VLANPortMapping  map[string][]string `yaml:"vlan_port_mapping"`
	BridgeDomain     string              `yaml:"bridge_domain"`
	AEProfile        string              `yaml:"ae_profile"`
	GenericAlias     map[string]string   `yaml:"generic_alias"`
	PrefixLists      map[string][]string `yaml:"prefix_lists"`
	CoreBGP          *CoreBGP            `yaml:"core_bgp"`
}

type CoreBGP struct {
	PeerGroups map[string][]string `yaml:"peer_groups"`
}

// --- Loading and Resolving Logic ---

// LoadGlobalIntent loads all YAML intent files from a given base path.
func LoadGlobalIntent(basePath string) (*GlobalIntent, error) {
	globalIntent := &GlobalIntent{
		DeviceProfiles: make(map[string]*DeviceProfile),
	}

	// Load network_intent.yaml
	netIntent := &NetworkIntent{}
	if err := loadYAMLFile(filepath.Join(basePath, "network_intent.yaml"), netIntent); err != nil {
		return nil, fmt.Errorf("error loading network intent: %w", err)
	}
	globalIntent.NetworkIntent = netIntent

	// Load site_intent.yaml
	siteIntent := &SiteIntent{}
	if err := loadYAMLFile(filepath.Join(basePath, "site_intent.yaml"), siteIntent); err != nil {
		return nil, fmt.Errorf("error loading site intent: %w", err)
	}
	globalIntent.SiteIntent = siteIntent

	// Load platform.yaml
	platformProfiles := &PlatformProfiles{}
	if err := loadYAMLFile(filepath.Join(basePath, "platform.yaml"), platformProfiles); err != nil {
		return nil, fmt.Errorf("error loading platform profiles: %w", err)
	}
	globalIntent.PlatformProfiles = platformProfiles

	// Load all device profiles from the 'profiles' subdirectory
	profilesPath := filepath.Join(basePath, "profiles")
	files, err := ioutil.ReadDir(profilesPath)
	if err != nil {
		return nil, fmt.Errorf("could not read profiles directory %s: %w", profilesPath, err)
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".yaml") || strings.HasSuffix(f.Name(), ".yml") {
			deviceName := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
			profile := &DeviceProfile{}
			if err := loadYAMLFile(filepath.Join(profilesPath, f.Name()), profile); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not load profile %s: %v\n", f.Name(), err)
				continue
			}
			globalIntent.DeviceProfiles[deviceName] = profile
		}
	}

	return globalIntent, nil
}

// Resolve combines the global intent with a specific device profile to create a fully resolved intent.
func (gi *GlobalIntent) Resolve(deviceName string) (*ResolvedIntent, error) {
	fmt.Printf("  [Intent API] Resolving intent for device: %s\n", deviceName)

	deviceProfile, ok := gi.DeviceProfiles[deviceName]
	if !ok {
		return nil, fmt.Errorf("no profile found for device: %s", deviceName)
	}

	// --- 1. Hierarchically Merge Aliases and Prefix Lists ---
	resolvedAliases := make(map[string]string)
	mergeStringMaps(resolvedAliases, gi.NetworkIntent.GenericAlias)

	resolvedPrefixLists := make(map[string][]string)
	mergeStringSliceMaps(resolvedPrefixLists, gi.NetworkIntent.PrefixLists)

	regionName := deviceProfile.Region
	regionData, regionExists := gi.NetworkIntent.Regions[regionName]
	if !regionExists {
		return nil, fmt.Errorf("region '%s' not found in network intent", regionName)
	}
	regionAliases := map[string]string{
		"pe-asnum":  fmt.Sprintf("%d", regionData.PeAsNum),
		"pe-asname": regionData.PeAsName,
		"PE-ASNAME": strings.ToUpper(regionData.PeAsName),
	}
	mergeStringMaps(resolvedAliases, regionAliases)
	mergeStringMaps(resolvedAliases, regionData.GenericAlias)
	mergeStringSliceMaps(resolvedPrefixLists, regionData.PrefixLists)

	mergeStringMaps(resolvedAliases, deviceProfile.GenericAlias)
	mergeStringSliceMaps(resolvedPrefixLists, deviceProfile.PrefixLists)

	// --- 2. Create the Resolver Engine ---
	resolver := newResolver(resolvedAliases, gi.NetworkIntent.ReverseExclude)

	// --- 3. Resolve Communities and VPNs ---
	resolvedCommunities := resolver.resolveMap(gi.NetworkIntent.Communities)
	resolvedVPNs := make(map[string]VPN)
	for name, vpn := range gi.NetworkIntent.VPNs {
		resolvedVPNs[name] = VPN{
			Description:      vpn.Description,
			ImportTarget:     resolver.resolveString(vpn.ImportTarget),
			ExportTarget:     resolver.resolveString(vpn.ExportTarget),
			ExportPrefixList: vpn.ExportPrefixList,
			Permission:       vpn.Permission,
			Service:          vpn.Service,
		}
	}

	// --- 4. Resolve the Active Bridge Domain ---
	var activeBridgeDomain *BridgeDomain
	if regionExists && deviceProfile.BridgeDomain != "" {
		if domain, ok := regionData.Bridge.Domains[deviceProfile.BridgeDomain]; ok {
			activeBridgeDomain = &domain
		}
	}

	// --- 5. Assemble the final ResolvedIntent struct ---
	resolvedIntent := &ResolvedIntent{
		DeviceName:           deviceName,
		Region:               deviceProfile.Region,
		Site:                 deviceProfile.Site,
		MgmtIP:               deviceProfile.MgmtIP,
		IsRouter:             deviceProfile.IsRouter,
		IsBridge:             deviceProfile.IsBridge,
		IsBorderRouter:       deviceProfile.IsBorderRouter,
		IsRouteReflector:     deviceProfile.IsRouteReflector,
		GenericAlias:         resolvedAliases,
		ReverseAlias:         resolver.reverseAliases,
		PrefixLists:          resolvedPrefixLists,
		Communities:          resolvedCommunities,
		Services:             gi.NetworkIntent.Services,
		VPNs:                 resolvedVPNs,
		PrefixServiceMapping: gi.NetworkIntent.PrefixServiceMapping,
		Permission:           gi.NetworkIntent.Permission,
		VLANPortMapping:      deviceProfile.VLANPortMapping,
		PolicyRules:          gi.NetworkIntent.PolicyRules,
		ActiveBridgeDomain:   activeBridgeDomain,
		CoreBGP:              deviceProfile.CoreBGP,
	}

	fmt.Printf("  [Intent API] Intent resolved successfully for %s.\n", deviceName)
	return resolvedIntent, nil
}

// --- Resolver Engine ---

type resolver struct {
	aliases        map[string]string
	reverseAliases map[string]string
	variableRegex  *regexp.Regexp
}

// newResolver initializes a new resolver engine with the final, merged alias map.
func newResolver(aliases map[string]string, reverseExclude []string) *resolver {
	r := &resolver{
		aliases:        aliases,
		reverseAliases: make(map[string]string),
		variableRegex:  regexp.MustCompile(`[<]?([a-zA-Z0-9_-]+)[>]?`),
	}

	excludeSet := make(map[string]bool)
	for _, item := range reverseExclude {
		excludeSet[item] = true
	}

	for key, val := range aliases {
		if !excludeSet[key] {
			r.reverseAliases[val] = key
		}
	}
	return r
}

// resolveString recursively substitutes variables in a string.
func (r *resolver) resolveString(input string) string {
	return r.variableRegex.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.Trim(match, "<>")
		if val, ok := r.aliases[key]; ok {
			return r.resolveString(val)
		}
		return match
	})
}

// resolveMap applies resolveString to all keys and values of a map.
func (r *resolver) resolveMap(m map[string]string) map[string]string {
	resolved := make(map[string]string)
	for key, val := range m {
		resolvedKey := r.resolveString(key)
		resolved[resolvedKey] = r.resolveString(val)
	}
	return resolved
}

// --- Helper Functions ---

func loadYAMLFile(path string, out interface{}) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

func mergeStringMaps(dest, source map[string]string) {
	for k, v := range source {
		dest[k] = v
	}
}

func mergeStringSliceMaps(dest, source map[string][]string) {
	for k, v := range source {
		dest[k] = v
	}
}