// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package healthcheck

import (
	"context"
	"fmt"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func NewIssuerWrapperHealthChecker(inner healthcheck.HealthCheck) *IssuerWrapperHealthChecker {
	return &IssuerWrapperHealthChecker{
		inner: inner,
	}
}

// IssuerWrapperHealthChecker contains all the information for the HealthCheck wrapper
type IssuerWrapperHealthChecker struct {
	logger     logr.Logger
	seedClient client.Client
	inner      healthcheck.HealthCheck
}

// InjectSeedClient injects the seed client
func (healthChecker *IssuerWrapperHealthChecker) InjectSeedClient(seedClient client.Client) {
	healthChecker.seedClient = seedClient
	if itf, ok := healthChecker.inner.(healthcheck.SeedClient); ok {
		itf.InjectSeedClient(seedClient)
	}
}

// SetLoggerSuffix injects the logger
func (healthChecker *IssuerWrapperHealthChecker) SetLoggerSuffix(provider, extension string) {
	healthChecker.logger = log.Log.WithName(fmt.Sprintf("%s-%s-healthcheck-issuer", provider, extension))
	healthChecker.inner.SetLoggerSuffix(provider, extension)
}

// DeepCopy clones the healthCheck struct by making a copy and returning the pointer to that new copy
func (healthChecker *IssuerWrapperHealthChecker) DeepCopy() healthcheck.HealthCheck {
	deepCopy := *healthChecker
	deepCopy.inner = healthChecker.inner.DeepCopy()
	return &deepCopy
}

// Check executes the health check
func (healthChecker *IssuerWrapperHealthChecker) Check(ctx context.Context, request types.NamespacedName) (*healthcheck.SingleCheckResult, error) {
	// first check the inner health
	result, err := healthChecker.inner.Check(ctx, request)
	if err != nil || result.Status == gardencorev1beta1.ConditionFalse {
		return result, err
	}

	list := &certv1alpha1.IssuerList{}
	if err := healthChecker.seedClient.List(ctx, list, client.InNamespace(request.Namespace)); err != nil {
		err := fmt.Errorf("check issuers failed. Unable to retrieve list of issuers in namespace '%s': %v", request.Namespace, err)
		healthChecker.logger.Error(err, "Health check failed")
		return nil, err
	}
	notReady := map[string]string{}
	for _, issuer := range list.Items {
		if issuer.Status.State != certv1alpha1.StateReady {
			msg := fmt.Sprintf("state='%s'", issuer.Status.State)
			if issuer.Status.Message != nil {
				msg += ", message=" + *issuer.Status.Message
			}
			notReady[issuer.Name] = msg
		}
	}
	if len(notReady) > 0 {
		healthChecker.logger.Info("Health check failed: issuers not ready", "issuers", notReady)
		return &healthcheck.SingleCheckResult{
			Status: gardencorev1beta1.ConditionFalse,
			Detail: fmt.Sprintf("%d/%d issuers not ready: %v", len(notReady), len(list.Items), notReady),
			Reason: "issuer not ready",
		}, nil
	}

	return &healthcheck.SingleCheckResult{
		Status: gardencorev1beta1.ConditionTrue,
	}, nil
}
