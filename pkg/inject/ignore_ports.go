package inject

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

// IgnorePorts holds the list of ports to ignore for both incoming and outgoing traffic.
type IgnorePorts struct {
	Incoming []int
	Outgoing []int
}

// IsEmpty reports whether or not IgnorePorts struct is empty.
func (ip IgnorePorts) IsEmpty() bool {
	return ip.Incoming == nil && ip.Outgoing == nil
}

// Validate returns an error if the ports are not valid.
func (ip IgnorePorts) Validate() error {
	validate := func(ports []int) error {
		for _, p := range ports {
			if !(p > 0 && p <= 65535) {
				return fmt.Errorf("'%d' is not a valid port", p)
			}
		}

		return nil
	}

	if err := validate(ip.Incoming); err != nil {
		return fmt.Errorf("incoming ignore ports are not valid: %w", err)
	}
	if err := validate(ip.Outgoing); err != nil {
		return fmt.Errorf("outgoing ignore ports are not valid: %w", err)
	}

	return nil
}

// GetIgnorePorts returns the ignored ports whether specified via arg or annotation.
func GetIgnorePorts(annotations map[string]string, ports IgnorePorts) (IgnorePorts, error) {
	annotatedIgnorePorts, err := GetIgnorePortsFromAnnotations(annotations)
	if err != nil {
		return IgnorePorts{}, err
	}

	if ports.IsEmpty() && annotatedIgnorePorts.IsEmpty() {
		return IgnorePorts{}, nil
	}
	if annotatedIgnorePorts.IsEmpty() {
		return ports, ports.Validate()
	}
	if ports.IsEmpty() {
		return annotatedIgnorePorts, annotatedIgnorePorts.Validate()
	}
	if !reflect.DeepEqual(annotatedIgnorePorts, ports) {
		return IgnorePorts{}, errors.New("ignore ports in annotation do not match those provided as arguments")
	}

	return ports, ports.Validate()
}

// GetIgnorePortsFromAnnotations returns the ignored ports that are set in the Pod annotations.
func GetIgnorePortsFromAnnotations(annotations map[string]string) (IgnorePorts, error) {
	convert := func(portsList []string) ([]int, error) {
		ports := make([]int, 0, len(portsList))
		for _, port := range portsList {
			p, err := strconv.Atoi(port)
			if err != nil {
				return []int{}, fmt.Errorf("error converting port to integer: %w", err)
			}
			ports = append(ports, p)
		}

		return ports, nil
	}

	ignPorts := IgnorePorts{}
	if val, ok := annotations[mesh.IgnoreOutgoingPortsAnnotation]; ok {
		annotationPorts := strings.Split(val, ",")
		ports, err := convert(annotationPorts)
		if err != nil {
			return IgnorePorts{}, err
		}

		ignPorts.Outgoing = ports
	}

	if val, ok := annotations[mesh.IgnoreIncomingPortsAnnotation]; ok {
		annotationPorts := strings.Split(val, ",")
		ports, err := convert(annotationPorts)
		if err != nil {
			return IgnorePorts{}, err
		}

		ignPorts.Incoming = ports
	}

	return ignPorts, nil
}
