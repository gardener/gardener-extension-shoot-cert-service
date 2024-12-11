package client

import (
	certmanv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apiextensionsinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	// ClusterScheme is the scheme used in garden runtime and unmanaged seed clusters.
	ClusterScheme = runtime.NewScheme()

	// ClusterSerializer is a YAML serializer using the 'ClusterScheme'.
	ClusterSerializer = json.NewSerializerWithOptions(json.DefaultMetaFactory, ClusterScheme, ClusterScheme, json.SerializerOptions{Yaml: true, Pretty: false, Strict: false})
	// ClusterCodec is a codec factory using the 'ClusterScheme'.
	ClusterCodec = serializer.NewCodecFactory(ClusterScheme)
)

func init() {
	clusterSchemeBuilder := runtime.NewSchemeBuilder(
		kubernetesscheme.AddToScheme,
		resourcesv1alpha1.AddToScheme,
		certmanv1alpha1.AddToScheme,
		vpaautoscalingv1.SchemeBuilder.AddToScheme,
		monitoringv1.AddToScheme,
	)

	utilruntime.Must(clusterSchemeBuilder.AddToScheme(ClusterScheme))
	apiextensionsinstall.Install(ClusterScheme)
}
