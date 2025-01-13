// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package garden

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"time"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/logger"
	. "github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const namespace = "garden"

var (
	parentCtx     context.Context
	runtimeClient client.Client
)

var _ = BeforeSuite(func() {
	Expect(os.Getenv("KUBECONFIG")).NotTo(BeEmpty(), "KUBECONFIG must be set")
	Expect(os.Getenv("REPO_ROOT")).NotTo(BeEmpty(), "REPO_ROOT must be set")

	logf.SetLogger(logger.MustNewZapLogger(logger.InfoLevel, logger.FormatJSON, zap.WriteTo(GinkgoWriter)))

	restConfig, err := kubernetes.RESTConfigFromClientConnectionConfiguration(&componentbaseconfigv1alpha1.ClientConnectionConfiguration{Kubeconfig: os.Getenv("KUBECONFIG")}, nil, kubernetes.AuthTokenFile, kubernetes.AuthClientCertificate)
	Expect(err).NotTo(HaveOccurred())

	scheme := runtime.NewScheme()
	Expect(kubernetesscheme.AddToScheme(scheme)).To(Succeed())
	Expect(operatorv1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(extensionsv1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(certv1alpha1.AddToScheme(scheme)).To(Succeed())
	runtimeClient, err = client.New(restConfig, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
})

var _ = BeforeEach(func() {
	parentCtx = context.Background()
})

func waitForGardenToBeReconciled(ctx context.Context, garden *operatorv1alpha1.Garden) {
	CEventually(ctx, func(g Gomega) gardencorev1beta1.LastOperationState {
		g.Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(garden), garden)).To(Succeed())
		if garden.Status.LastOperation == nil || garden.Status.ObservedGeneration != garden.Generation {
			return ""
		}
		return garden.Status.LastOperation.State
	}).WithPolling(2 * time.Second).Should(Equal(gardencorev1beta1.LastOperationStateSucceeded))
}

func waitForOperatorExtensionToBeReconciled(
	ctx context.Context,
	extension *operatorv1alpha1.Extension,
	expectedRuntimeStatus, expectedVirtualStatus gardencorev1beta1.ConditionStatus,
) {
	CEventually(ctx, func(g Gomega) []gardencorev1beta1.Condition {
		g.Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(extension), extension)).To(Succeed())
		if extension.Status.ObservedGeneration != extension.Generation {
			return nil
		}

		return extension.Status.Conditions
	}).WithPolling(2 * time.Second).Should(ConsistOf(MatchFields(IgnoreExtras, Fields{
		"Type":   Equal(operatorv1alpha1.ExtensionInstalled),
		"Status": Equal(gardencorev1beta1.ConditionTrue),
	}), MatchFields(IgnoreExtras, Fields{
		"Type":   Equal(operatorv1alpha1.ExtensionRequiredRuntime),
		"Status": Equal(expectedRuntimeStatus),
	}), MatchFields(IgnoreExtras, Fields{
		"Type":   Equal(operatorv1alpha1.ExtensionRequiredVirtual),
		"Status": Equal(expectedVirtualStatus),
	})))
}

func waitForOperatorExtensionToBeDeleted(ctx context.Context, extension *operatorv1alpha1.Extension) {
	CEventually(ctx, func() error {
		return runtimeClient.Get(ctx, client.ObjectKeyFromObject(extension), extension)
	}).WithPolling(2 * time.Second).Should(BeNotFoundError())
}

func waitForExtensionToBeReconciled(ctx context.Context, extension *extensionsv1alpha1.Extension) {
	CEventually(ctx, func(g Gomega) gardencorev1beta1.LastOperationState {
		g.Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(extension), extension)).To(Succeed())
		if extension.Status.LastOperation == nil || extension.Status.ObservedGeneration != extension.Generation {
			return ""
		}
		return extension.Status.LastOperation.State
	}).WithPolling(2 * time.Second).Should(Equal(gardencorev1beta1.LastOperationStateSucceeded))
}

func waitForCertificateToBeReconciled(ctx context.Context, cert *certv1alpha1.Certificate, statusMatcher gomegatypes.GomegaMatcher) {
	CEventually(ctx, func(g Gomega) certv1alpha1.CertificateStatus {
		g.Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(cert), cert)).To(Succeed())
		return cert.Status
	}).WithPolling(2 * time.Second).Should(statusMatcher)
}

func createDummyTLSSecret(ctx context.Context, certificate *certv1alpha1.Certificate) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: certificate.Spec.SecretRef.Namespace,
			Name:      certificate.Spec.SecretRef.Name,
		},
	}
	_, err := controllerutils.GetAndCreateOrMergePatch(ctx, runtimeClient, secret, func() error {
		if secret.Data == nil {
			certPEM, keyPEM, err := createSelfSignedTLSSecret()
			if err != nil {
				return err
			}
			secret.Data = map[string][]byte{
				"tls.crt": certPEM,
				"tls.key": keyPEM,
			}
		}
		secret.Type = corev1.SecretTypeTLS
		secret.Labels = certificate.Spec.SecretLabels
		return nil
	})
	return err
}

func createSelfSignedTLSSecret() ([]byte, []byte, error) {
	certPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	certPrivateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certPrivateKey)})
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "dummy",
		},
		DNSNames:              []string{"dummy"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLen:            0,
	}

	certDerBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, certPrivateKey.Public(), certPrivateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDerBytes})
	return certPEM, certPrivateKeyPEM, nil
}

func waitForVirtualGardenKubeAPIServerPatched(ctx context.Context) {
	idFn := func(element interface{}) string {
		return fmt.Sprintf("%v", element)
	}
	expectedArg := "--tls-sni-cert-key=/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.crt,/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.key:api.vg.local.gardener.cloud"
	CEventually(ctx, func(g Gomega) []string {
		deployment := &appsv1.Deployment{}
		g.Expect(runtimeClient.Get(ctx, client.ObjectKey{Namespace: "garden", Name: "virtual-garden-kube-apiserver"}, deployment)).To(Succeed())
		return deployment.Spec.Template.Spec.Containers[0].Args
	}).WithPolling(2 * time.Second).Should(MatchElements(idFn, IgnoreExtras, Elements{expectedArg: Equal(expectedArg)}))
}

func getExtensionNamespace(ctx context.Context, controllerRegistrationName string) string {
	namespaces := &corev1.NamespaceList{}
	Expect(runtimeClient.List(ctx, namespaces, client.MatchingLabels{
		v1beta1constants.GardenRole:                      v1beta1constants.GardenRoleExtension,
		v1beta1constants.LabelControllerRegistrationName: controllerRegistrationName,
	})).To(Succeed())
	Expect(namespaces.Items).To(HaveLen(1))
	return namespaces.Items[0].Name
}

// ExecMake executes one or multiple make targets.
func execMake(ctx context.Context, targets ...string) error {
	cmd := exec.CommandContext(ctx, "make", targets...)
	cmd.Dir = os.Getenv("REPO_ROOT")
	for _, key := range []string{"PATH", "GOPATH", "HOME", "KUBECONFIG"} {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, os.Getenv(key)))
	}
	cmdString := fmt.Sprintf("running make %s", strings.Join(targets, " "))
	logf.Log.Info(cmdString)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %s\n%s", cmdString, err, string(output))
	}
	return nil
}
