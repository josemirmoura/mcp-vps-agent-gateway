package cloudnode

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
)

func EnsureEnrollment(
	ctx context.Context,
	client *Client,
	statePath, enrollmentToken, nodeName string,
) (Identity, error) {
	if client == nil {
		return Identity{}, errors.New("Cloud client is required")
	}
	if statePath == "" {
		return Identity{}, errors.New("state path is required")
	}

	state, err := LoadState(statePath)
	switch {
	case err == nil:
		if state.Identity.NodeID != "" {
			return state.Identity, nil
		}
		if enrollmentToken != "" && enrollmentToken != state.Enrollment.Token {
			return Identity{}, errors.New("pending enrollment token differs from supplied token")
		}
	case errors.Is(err, os.ErrNotExist):
		if strings.TrimSpace(enrollmentToken) == "" {
			return Identity{}, errors.New("enrollment token is required for first enrollment")
		}
		if strings.TrimSpace(nodeName) == "" {
			return Identity{}, errors.New("node name is required for first enrollment")
		}

		identity, generateErr := GenerateIdentity()
		if generateErr != nil {
			return Identity{}, generateErr
		}
		state = NodeState{
			Identity: identity,
			Enrollment: &PendingEnrollment{
				Token:    strings.TrimSpace(enrollmentToken),
				NodeName: strings.TrimSpace(nodeName),
				Platform: runtime.GOOS,
			},
		}
		if err := SaveState(statePath, state); err != nil {
			return Identity{}, err
		}
	default:
		return Identity{}, err
	}

	response, err := client.Enroll(
		ctx,
		state.Enrollment.Token,
		state.Enrollment.NodeName,
		state.Enrollment.Platform,
		state.Identity.PublicKey,
	)
	if err != nil {
		return Identity{}, err
	}

	state.Identity.NodeID = response.NodeID
	state.Identity.WorkspaceID = response.WorkspaceID
	state.Identity.Fingerprint = response.Fingerprint
	state.Enrollment = nil
	if err := SaveState(statePath, state); err != nil {
		return Identity{}, err
	}
	return state.Identity, nil
}
