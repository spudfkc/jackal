/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

/*
#cgo LDFLAGS: -lidn
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "stringprep.h"

char *nodeprep(char *in) {
	int maxlen = strlen(in)*4 + 1;
	char *buf = (char *)(malloc(maxlen));

	strcpy(buf, in);
	int rc = stringprep_xmpp_nodeprep(buf, maxlen);
	if (rc != 0) {
		free(buf);
		return NULL;
	}
	return buf;
}

char *domainprep(char *str) {
	char *in = stringprep_convert(str, "ASCII", "UTF-8");
	if (in == NULL) {
		return NULL;
	}

	int maxlen = strlen(in)*4 + 1;
	char *buf = (char *)(malloc(maxlen));

	strcpy(buf, in);
	free(in);

	int rc = stringprep_nameprep(buf, maxlen);
	if (rc != 0) {
		free(buf);
		return NULL;
	}
	return buf;
}

char *resourceprep(char *in) {
	int maxlen = strlen(in)*4 + 1;
	char *buf = (char *)(malloc(maxlen));

	strcpy(buf, in);
	int rc = stringprep_xmpp_resourceprep(buf, maxlen);
	if (rc != 0) {
		free(buf);
		return NULL;
	}
	return buf;
}
*/
import "C"
import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

// JIDMatchingOptions represents a matching jid mask.
type JIDMatchingOptions int8

const (
	// JIDMatchesNode indicates that left and right operand has same node value.
	JIDMatchesNode = JIDMatchingOptions(1)

	// JIDMatchesDomain indicates that left and right operand has same domain value.
	JIDMatchesDomain = JIDMatchingOptions(2)

	// JIDMatchesResource indicates that left and right operand has same resource value.
	JIDMatchesResource = JIDMatchingOptions(4)
)

// JID represents an XMPP address (JID).
// A JID is made up of a node (generally a username), a domain, and a resource.
// The node and resource are optional; domain is required.
type JID struct {
	node     string
	domain   string
	resource string
}

// NewJID constructs a JID given a user, domain, and resource.
// This construction allows the caller to specify if stringprep should be applied or not.
func NewJID(node, domain, resource string, skipStringPrep bool) (*JID, error) {
	if skipStringPrep {
		return &JID{
			node:     node,
			domain:   domain,
			resource: resource,
		}, nil
	}
	prepNode, err := nodeprep(node)
	if err != nil {
		return nil, err
	}
	prepDomain, err := domainprep(domain)
	if err != nil {
		return nil, err
	}
	prepResource, err := resourceprep(resource)
	if err != nil {
		return nil, err
	}
	return &JID{
		node:     prepNode,
		domain:   prepDomain,
		resource: prepResource,
	}, nil
}

// NewJIDString constructs a JID from it's string representation.
// This construction allows the caller to specify if stringprep should be applied or not.
func NewJIDString(str string, skipStringPrep bool) (*JID, error) {
	if len(str) == 0 {
		return &JID{}, nil
	}
	var node, domain, resource string

	atIndex := strings.Index(str, "@")
	slashIndex := strings.Index(str, "/")

	// node
	if atIndex > 0 {
		node = str[0:atIndex]
	}

	// domain
	if atIndex+1 == len(str) {
		return nil, errors.New("JID with empty domain not valid")
	}
	if atIndex < 0 {
		if slashIndex > 0 {
			domain = str[0:slashIndex]
		} else {
			domain = str
		}
	} else {
		if slashIndex > 0 {
			domain = str[atIndex+1 : slashIndex]
		} else {
			domain = str[atIndex+1:]
		}
	}

	// resource
	if slashIndex > 0 && slashIndex+1 < len(str) {
		resource = str[slashIndex+1:]
	}
	return NewJID(node, domain, resource, skipStringPrep)
}

// Node returns the node, or empty string if this JID does not contain node information.
func (j *JID) Node() string {
	return j.node
}

// Domain returns the domain.
func (j *JID) Domain() string {
	return j.domain
}

// Resource returns the resource, or empty string if this JID does not contain resource information.
func (j *JID) Resource() string {
	return j.resource
}

// ToBareJID returns the JID equivalent of the bare JID, which is the JID with resource information removed.
func (j *JID) ToBareJID() *JID {
	if len(j.node) == 0 {
		return &JID{node: "", domain: j.domain, resource: ""}
	}
	return &JID{node: j.node, domain: j.domain, resource: ""}
}

// IsServer returns true if instance is a server JID.
func (j *JID) IsServer() bool {
	return len(j.node) == 0
}

// IsBare returns true if instance is a bare JID.
func (j *JID) IsBare() bool {
	return len(j.node) > 0 && len(j.resource) == 0
}

// IsFull returns true if instance is a full JID.
func (j *JID) IsFull() bool {
	return len(j.resource) > 0
}

// IsFullWithServer returns true if instance is a full server JID.
func (j *JID) IsFullWithServer() bool {
	return len(j.node) == 0 && len(j.resource) > 0
}

// IsFullWithUser returns true if instance is a full client JID.
func (j *JID) IsFullWithUser() bool {
	return len(j.node) > 0 && len(j.resource) > 0
}

// Matches returns true if two JID's are equivalent.
func (j *JID) Matches(j2 *JID, options JIDMatchingOptions) bool {
	if (options&JIDMatchesNode) > 0 && j.node != j2.node {
		return false
	}
	if (options&JIDMatchesDomain) > 0 && j.domain != j2.domain {
		return false
	}
	if (options&JIDMatchesResource) > 0 && j.resource != j2.resource {
		return false
	}
	return true
}

// String returns a string representation of the JID.
func (j *JID) String() string {
	buf := bufPool.Get()
	defer bufPool.Put(buf)
	if len(j.node) > 0 {
		buf.WriteString(j.node)
		buf.WriteString("@")
	}
	buf.WriteString(j.domain)
	if len(j.resource) > 0 {
		buf.WriteString("/")
		buf.WriteString(j.resource)
	}
	return buf.String()
}

func nodeprep(in string) (string, error) {
	cin := C.CString(in)
	defer C.free(unsafe.Pointer(cin))

	prep := C.nodeprep(cin)
	if prep == nil {
		return "", fmt.Errorf("input is not a valid JID node part: %v", []byte(in))
	}
	defer C.free(unsafe.Pointer(prep))
	if C.strlen(prep) > 1073 {
		return "", fmt.Errorf("node cannot be larger than 1073. Size is %d bytes", C.strlen(prep))
	}
	return C.GoString(prep), nil
}

func domainprep(in string) (string, error) {
	cin := C.CString(in)
	defer C.free(unsafe.Pointer(cin))

	prep := C.domainprep(cin)
	if prep == nil {
		return "", fmt.Errorf("input is not a valid JID domain part: %v", []byte(in))
	}
	defer C.free(unsafe.Pointer(prep))
	if C.strlen(prep) > 1073 {
		return "", fmt.Errorf("domain cannot be larger than 1073. Size is %d bytes", C.strlen(prep))
	}
	return C.GoString(prep), nil
}

func resourceprep(in string) (string, error) {
	cin := C.CString(in)
	defer C.free(unsafe.Pointer(cin))

	prep := C.resourceprep(cin)
	if prep == nil {
		return "", fmt.Errorf("input is not a valid JID resource part: %v", []byte(in))
	}
	defer C.free(unsafe.Pointer(prep))
	if C.strlen(prep) > 1073 {
		return "", fmt.Errorf("resource cannot be larger than 1073. Size is %d bytes", C.strlen(prep))
	}
	return C.GoString(prep), nil
}
