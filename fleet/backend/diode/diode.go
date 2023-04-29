/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package diode

import (
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/opentracing/opentracing-go"
	"github.com/orb-community/orb/fleet"
	"github.com/orb-community/orb/fleet/backend"
)

var _ backend.Backend = (*diodeBackend)(nil)

type diodeBackend struct {
	auth        mainflux.AuthServiceClient
	agentRepo   fleet.AgentRepository
	Backend     string
	Description string
}

type BackendTaps struct {
	Name             string
	InputType        string
	ConfigPredefined []string
	TotalAgents      uint64
}

func (p diodeBackend) Metadata() interface{} {
	return struct {
		Backend       string `json:"backend"`
		Description   string `json:"description"`
		SchemaVersion string `json:"schema_version"`
	}{
		Backend:       p.Backend,
		Description:   p.Description,
		SchemaVersion: CurrentSchemaVersion,
	}
}

func (p diodeBackend) MakeHandler(tracer opentracing.Tracer, opts []kithttp.ServerOption, r *bone.Mux) {
	MakeDiodeHandler(tracer, p, opts, r)
}

func Register(auth mainflux.AuthServiceClient, agentRepo fleet.AgentRepository) bool {
	backend.Register("diode", &diodeBackend{
		Backend:     "diode",
		Description: "diode observability agent from diode.dev",
		auth:        auth,
		agentRepo:   agentRepo,
	})
	return true
}
