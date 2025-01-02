// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package sniconfig

import (
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/resourcemanager/apis/config"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	// HandlerName is the name of the webhook handler.
	HandlerName = "sni-config"
	// WebhookPath is the path at which the handler should be registered.
	WebhookPath = "/webhooks/sni-config"
)

func AddToManager(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	handler := &Handler{
		Logger:       mgr.GetLogger().WithName("webhook").WithName(HandlerName),
		TargetClient: mgr.GetClient(),
		Config:       config.HighAvailabilityConfigWebhookConfig{},
		Decoder:      admission.NewDecoder(mgr.GetScheme()),
	}
	return &extensionswebhook.Webhook{
		Name:              HandlerName,
		Provider:          "",
		Action:            extensionswebhook.ActionMutating,
		NamespaceSelector: nil,
		ObjectSelector:    nil,
		Path:              WebhookPath,
		Target:            extensionswebhook.TargetSeed,
		Webhook:           &admission.Webhook{Handler: handler, RecoverPanic: ptr.To(true)},
		Types: []extensionswebhook.Type{
			{Obj: &appsv1.Deployment{}},
		},
	}, nil
}
