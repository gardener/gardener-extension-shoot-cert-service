// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shared

const (
	// EnvLeaderElectionNamespace is the environment variable name set in the deployment for providing the pod namespace.
	EnvLeaderElectionNamespace = "LEADER_ELECTION_NAMESPACE"
	// FinalizerSuffix is the finalizer suffix for the shoot cert service controller.
	FinalizerSuffix = "shoot-cert-service"
)
