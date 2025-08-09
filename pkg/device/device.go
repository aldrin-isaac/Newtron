// File: pkg/device/device.go
// Chat Session: newtron-20250706-01
// Session Timestamp: 2025-07-06T15:24:00-07:00
package device

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"newtron/pkg/intent"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// Device is the primary struct representing a network device and its connection.
type Device struct {
	Node       *Node
	gnmiClient gpb.GNMIClient
	conn       *grpc.ClientConn
	Intent     *intent.ResolvedIntent
}

// --- Primary Device State Structs ---
// ... (structs are unchanged)

type Node struct {
	Name        string
	Version     string
	Chassis     string
	ConfigClass string
	Cards       map[string]*Card
}

type Card struct {
	ID          string
	Model       string
	Description string
	PortType    string
	Ports       []*Port
	ParentNode  *Node
	ValidSpeeds []string
	IntfProto   string
	IntfEncaps  []string
	IntfSymb    string
	MaximumMTU  int
	QueueType   string
	Bridging    bool
}

type Port struct {
	ID            string
	ParentCard    *Card
	IfName        string
	OperStatus    string
	AdminStatus   string
	Mode          string
	Speed         string
	Bridging      bool
	SubInterfaces []*SubInterface
	LAG           *LAG
}

type SubInterface struct {
	ID          int
	ParentPort  *Port
	IfName      string
	Description string
	IPAddress   string
	Service     string
	AdminStatus string
	Bandwidth   string
	FilterName  string
	VRF         *VRF
}

type VRF struct {
	Name               string
	RouteDistinguisher string
	ImportRouteTargets []string
	ExportRouteTargets []string
	AssociatedVPNs     map[string]*VPN
}

type LAG struct {
	LACPEnabled bool
	LACPMode    string
	LACPRate    string
	Members     []*Port
}

type VPN struct {
	Name        string
	Description string
}


// --- Device Methods ---

// NewDevice creates and initializes a new Device instance.
func NewDevice(name string, resolvedIntent *intent.ResolvedIntent) (*Device, error) {
	node := &Node{
		Name:  name,
		Cards: make(map[string]*Card),
	}
	return &Device{
		Node:   node,
		Intent: resolvedIntent,
	}, nil
}

// Connect establishes a real gNMI connection to the device.
func (d *Device) Connect(user, pass string) error {
	fmt.Printf("  [Backend API] Connecting to %s...\n", d.Intent.MgmtIP)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(newAuthCreds(user, pass)),
	}

	conn, err := grpc.DialContext(ctx, d.Intent.MgmtIP, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC server for %s: %w", d.Intent.MgmtIP, err)
	}

	d.conn = conn
	d.gnmiClient = gpb.NewGNMIClient(conn)

	if err := d.loadInitialState(); err != nil {
		d.Close()
		return fmt.Errorf("failed to load initial device state: %w", err)
	}

	fmt.Printf("  [Backend API] Connection successful.\n")
	return nil
}

// Close gracefully terminates the gNMI connection.
func (d *Device) Close() {
	if d.conn != nil {
		fmt.Printf("  [Backend API] Closing connection to %s...\n", d.Node.Name)
		d.conn.Close()
	}
}

// gnmiSet is a generic helper to perform gNMI Set operations (Update, Replace, Delete).
func (d *Device) gnmiSet(ctx context.Context, pathStr string, op gpb.UpdateResult_Operation, s ygot.GoStruct) error {
	path, err := StringToPath(pathStr)
	if err != nil {
		return err
	}

	var setReq *gpb.SetRequest
	switch op {
	case gpb.UpdateResult_DELETE:
		setReq = &gpb.SetRequest{Delete: []*gpb.Path{path}}
	default: // UPDATE and REPLACE
		jsonVal, err := ygot.EmitJSON(s, &ygot.EmitJSONConfig{
			Format: ygot.RFC7951,
			RFC7951Config: &ygot.RFC7951JSONConfig{
				AppendModuleName: true,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to marshal config struct to JSON: %w", err)
		}
		update := &gpb.Update{
			Path: path,
			Val:  &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(jsonVal)}},
		}
		if op == gpb.UpdateResult_REPLACE {
			setReq = &gpb.SetRequest{Replace: []*gpb.Update{update}}
		} else {
			setReq = &gpb.SetRequest{Update: []*gpb.Update{update}}
		}
	}

	_, err = d.gnmiClient.Set(ctx, setReq)
	if err != nil {
		return fmt.Errorf("gNMI Set failed for path %s (op: %s): %w", pathStr, op, err)
	}

	fmt.Printf("  [Backend API] Successfully applied config for path %s (op: %s)\n", pathStr, op)
	return nil
}

// --- gNMI Path Helper ---
var (
	pathElementRe = regexp.MustCompile(`([a-zA-Z0-9\-:_]+)(\[([a-zA-Z0-9\-:_]+)='?([^']*)'?\])?`)
)

// StringToPath converts a string like "/interfaces/interface[name=et-0/0/0]" into a gNMI Path object.
func StringToPath(p string) (*gpb.Path, error) {
	path := &gpb.Path{Origin: "openconfig"}
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return path, nil
	}
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		matches := pathElementRe.FindStringSubmatch(part)
		if matches == nil {
			return nil, fmt.Errorf("invalid path element format: %s", part)
		}
		elem := &gpb.PathElem{Name: matches[1]}
		if len(matches) > 3 && matches[3] != "" && matches[4] != "" {
			elem.Key = map[string]string{matches[3]: matches[4]}
		}
		path.Elem = append(path.Elem, elem)
	}
	return path, nil
}

// --- Authentication Helper ---
type authCreds struct {
	username, password string
}

func newAuthCreds(username, password string) *authCreds {
	return &authCreds{username: username, password: password}
}

func (c *authCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"username": c.username,
		"password": c.password,
	}, nil
}

func (c *authCreds) RequireTransportSecurity() bool {
	return false
}

