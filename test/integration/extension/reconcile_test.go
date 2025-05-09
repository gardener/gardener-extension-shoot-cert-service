// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension_test

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/logger"
	gardenerutils "github.com/gardener/gardener/pkg/utils"
	. "github.com/gardener/gardener/pkg/utils/test"
	"github.com/gardener/gardener/pkg/utils/test/matchers"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	servicev1alpha1 "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/extension"
)

var (
	logLevel = flag.String("logLevel", "", "Log level (debug, info, error)")
)

const (
	defaultTimeout = 30 * time.Second
)

func validateFlags() {
	if len(*logLevel) == 0 {
		logLevel = ptr.To(logger.DebugLevel)
	} else {
		if !slices.Contains(logger.AllLogLevels, *logLevel) {
			panic("invalid log level: " + *logLevel)
		}
	}
}

var (
	ctx = context.Background()

	log       logr.Logger
	testEnv   *envtest.Environment
	mgrCancel context.CancelFunc
	c         client.Client
	contains  func(...client.Object) types.GomegaMatcher

	testName string

	namespace      *corev1.Namespace
	shoot          *gardencorev1beta1.Shoot
	cluster        *extensionsv1alpha1.Cluster
	providerConfig *runtime.RawExtension
)

var _ = BeforeSuite(func() {
	flag.Parse()
	validateFlags()

	repoRoot := filepath.Join("..", "..", "..")

	// enable manager logs
	logf.SetLogger(logger.MustNewZapLogger(*logLevel, logger.FormatJSON, zap.WriteTo(GinkgoWriter)))

	log = logf.Log.WithName("extension-test")

	DeferCleanup(func() {
		By("stopping manager")
		mgrCancel()

		By("running cleanup actions")
		framework.RunCleanupActions()

		By("tearing down shoot environment")
		teardownShootEnvironment(ctx, c, namespace, cluster)

		By("stopping test environment")
		Expect(testEnv.Stop()).To(Succeed())
	})

	By("generating randomized test resource identifiers")
	testName = fmt.Sprintf("shoot--foo--%s", randomString())
	namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
		},
	}
	providerConfig = createProviderConfig("my-acme-ref")
	shoot = &gardencorev1beta1.Shoot{
		Spec: gardencorev1beta1.ShootSpec{
			DNS: &gardencorev1beta1.DNS{
				Domain: ptr.To(testName + "example.com"),
			},
			Kubernetes: gardencorev1beta1.Kubernetes{
				Version: "1.31.0",
			},
			Extensions: []gardencorev1beta1.Extension{
				{
					Type:           "shoot-cert-service",
					ProviderConfig: providerConfig,
				},
			},
			Resources: []gardencorev1beta1.NamedResourceReference{
				{
					Name: "my-acme-ref",
					ResourceRef: autoscalingv1.CrossVersionObjectReference{
						APIVersion: "v1",
						Kind:       "Secret",
						Name:       "my-acme",
					},
				},
			},
		},
	}
	cluster = &extensionsv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			CloudProfile: runtime.RawExtension{Raw: []byte("{}")},
			Seed:         runtime.RawExtension{Raw: []byte("{}")},
			Shoot:        runtime.RawExtension{Raw: shootToBytes(shoot)},
		},
	}

	By("starting test environment")
	testEnv = &envtest.Environment{
		CRDInstallOptions: envtest.CRDInstallOptions{
			Paths: []string{
				filepath.Join(repoRoot, "example", "20-crd-cluster.yaml"),
				filepath.Join(repoRoot, "example", "20-crd-extension.yaml"),
				filepath.Join(repoRoot, "example", "20-crd-issuer.yaml"),
				filepath.Join(repoRoot, "example", "20-crd-managedresource.yaml"),
			},
		},
	}

	restConfig, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(restConfig).ToNot(BeNil())

	By("setting up manager")
	mgr, err := manager.New(restConfig, manager.Options{
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	Expect(extensionsv1alpha1.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(certv1alpha1.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(service.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(servicev1alpha1.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(resourcesv1alpha1.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(kubernetesscheme.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(vpaautoscalingv1.SchemeBuilder.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(monitoringv1.AddToScheme(mgr.GetScheme())).To(Succeed())

	Expect(extension.AddToManagerWithOptions(ctx, mgr, extension.AddOptions{})).To(Succeed())

	var mgrContext context.Context
	mgrContext, mgrCancel = context.WithCancel(ctx)

	By("starting manager")
	go func() {
		defer GinkgoRecover()
		err := mgr.Start(mgrContext)
		Expect(err).NotTo(HaveOccurred())
	}()

	// test client should be uncached and independent of the tested manager
	c, err = client.New(restConfig, client.Options{Scheme: mgr.GetScheme()})
	Expect(err).NotTo(HaveOccurred())
	Expect(c).NotTo(BeNil())

	By("setting up shoot environment")
	setupShootEnvironment(ctx, c, namespace, cluster)

	contains = matchers.NewManagedResourceContainsObjectsMatcher(c)
})

var _ = Describe("Extension tests", func() {
	It("it should reconcile extension with own issuer", func() {
		By("creating extension")
		ext := &extensionsv1alpha1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: testName,
			},
			Spec: extensionsv1alpha1.ExtensionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           "shoot-cert-service",
					ProviderConfig: providerConfig,
				},
			},
		}
		Expect(c.Create(ctx, ext)).To(Succeed())

		By("waiting for extension last operation to succeed")
		CEventually(ctx, func() bool {
			Expect(c.Get(ctx, client.ObjectKeyFromObject(ext), ext)).To(Succeed())
			return ext.Status.LastOperation != nil && ext.Status.LastOperation.State == gardencorev1beta1.LastOperationStateSucceeded
		}).WithPolling(1 * time.Second).WithTimeout(defaultTimeout).Should(BeTrue())

		By("check for managed resources")
		mrSeed := &resourcesv1alpha1.ManagedResource{}
		mrShoot := &resourcesv1alpha1.ManagedResource{}
		Expect(c.Get(ctx, client.ObjectKey{Namespace: testName, Name: "extension-shoot-cert-service-seed"}, mrSeed)).To(Succeed())
		Expect(c.Get(ctx, client.ObjectKey{Namespace: testName, Name: "extension-shoot-cert-service-shoot"}, mrShoot)).To(Succeed())

		issuer := &certv1alpha1.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myissuer",
				Namespace: testName,
			},
			Spec: certv1alpha1.IssuerSpec{
				ACME: &certv1alpha1.ACMESpec{
					Server: "https://my-own-acme.somewhere.com/directory",
					Email:  "someone@somewhere.com",
					PrivateKeySecretRef: &corev1.SecretReference{
						Name:      "ref-my-acme",
						Namespace: testName,
					},
				},
			},
		}
		Expect(mrSeed).To(contains(issuer))

		By("deleting extension")
		Expect(c.Delete(ctx, ext)).To(Succeed())
		CEventually(ctx, func() bool {
			err := c.Get(ctx, client.ObjectKeyFromObject(ext), ext)
			return err != nil && client.IgnoreNotFound(err) == nil
		}).WithPolling(1 * time.Second).WithTimeout(defaultTimeout).Should(BeTrue())
	})

	It("it should fail to reconcile extension with own issuer if reference is wrong", func() {
		By("creating extension")
		ext := &extensionsv1alpha1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: testName,
			},
			Spec: extensionsv1alpha1.ExtensionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           "shoot-cert-service",
					ProviderConfig: createProviderConfig("not-existing-resource-reference"),
				},
			},
		}
		Expect(c.Create(ctx, ext)).To(Succeed())

		By("waiting for extension last operation to fail")
		CEventually(ctx, func() bool {
			Expect(c.Get(ctx, client.ObjectKeyFromObject(ext), ext)).To(Succeed())
			return ext.Status.LastOperation != nil && ext.Status.LastOperation.State == gardencorev1beta1.LastOperationStateError
		}).WithPolling(1 * time.Second).WithTimeout(defaultTimeout).Should(BeTrue())

		By("deleting extension")
		Expect(c.Delete(ctx, ext)).To(Succeed())
		CEventually(ctx, func() bool {
			err := c.Get(ctx, client.ObjectKeyFromObject(ext), ext)
			return err != nil && client.IgnoreNotFound(err) == nil
		}).WithPolling(1 * time.Second).WithTimeout(defaultTimeout).Should(BeTrue())
	})
})

func setupShootEnvironment(ctx context.Context, c client.Client, namespace *corev1.Namespace, cluster *extensionsv1alpha1.Cluster) {
	Expect(c.Create(ctx, namespace)).To(Succeed())
	Expect(c.Create(ctx, cluster)).To(Succeed())
}

func teardownShootEnvironment(ctx context.Context, c client.Client, namespace *corev1.Namespace, cluster *extensionsv1alpha1.Cluster) {
	Expect(client.IgnoreNotFound(c.Delete(ctx, cluster))).To(Succeed())
	Expect(c.DeleteAllOf(ctx, &corev1.Secret{}, client.InNamespace(namespace.Name), client.MatchingLabels{"resources.gardener.cloud/garbage-collectable-reference": "true"})).To(Succeed())
	Expect(client.IgnoreNotFound(c.Delete(ctx, namespace))).To(Succeed())
}

func randomString() string {
	rs, err := gardenerutils.GenerateRandomStringFromCharset(5, "0123456789abcdefghijklmnopqrstuvwxyz")
	Expect(err).NotTo(HaveOccurred())
	return rs
}

func shootToBytes(shoot *gardencorev1beta1.Shoot) []byte {
	data, err := json.Marshal(shoot)
	Expect(err).NotTo(HaveOccurred())
	return data
}

func createProviderConfig(ref string) *runtime.RawExtension {
	return &runtime.RawExtension{
		Raw: []byte(fmt.Sprintf(`{
  "apiVersion": "service.cert.extensions.gardener.cloud/v1alpha1",
  "issuers": [{
    "email": "someone@somewhere.com",
    "name": "myissuer",
    "privateKeySecretName": "%s",
    "server": "https://my-own-acme.somewhere.com/directory"
  }]
}`, ref)),
	}
}
